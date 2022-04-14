/*
Copyright 2021 by Milo Christiansen

This software is provided 'as-is', without any express or implied warranty. In
no event will the authors be held liable for any damages arising from the use of
this software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to
the following restrictions:

1. The origin of this software must not be misrepresented; you must not claim
that you wrote the original software. If you use this software in a product, an
acknowledgment in the product documentation would be appreciated but is not
required.

2. Altered source versions must be plainly marked as such, and must not be
misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
*/

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/milochristiansen/ledger"
	"github.com/milochristiansen/ledger/parse"
	"github.com/teris-io/shortid"
)

// Client does all the work of keeping a clear consistent view of the underlying transaction log for the UI.
// This handles loading and updating the log file, parsing and sorting all the data, and doing all the other
// calculating/transforming that may be needed.
type Client struct {
	ledger *os.File // The current ledger file, open for appending.

	// All the transactions in the ledger file, exactly as they appear and in source order.
	raw []ledger.Transaction

	// The simplified transactions, all edits and such removed, in chronological order then source order
	// for same-date transactions.
	simple   []ledger.Transaction
	simpleid map[string]int // ID to index map for the simplified list.

	// All transactions by ID, each group is then in source order (so the last item in each list is the
	// authoritative version).
	byid map[string][]ledger.Transaction

	lock sync.RWMutex

	// Events are sent on this channel.
	Events chan *Event
}

// Returned by GetClient if, during loading, a transaction is found that does not have an ID.
// Since all transactions written by this system are given IDs this means a corrupted or badly
// manually edited file. Go fix your mistake and try again.
var MissingIDError = errors.New("Transaction missing ID.")

const (
	EvntTypTrUpdate = iota // The transaction list has changed, refresh.
)

type Event struct {
	Typ int
}

// NewClient returns a client object or an error if the client was not able to initialize.
// Do not make multiple Clients! Each Client has associated, non-releasable resources!
func NewClient() (*Client, error) {
	client := &Client{
		Events: make(chan *Event),
	}
	var err error

	// First get use the current transactions log from the disk.
	client.ledger, err = os.OpenFile("transactions.ledger", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	// Now, since my parser is a little dumb, slurp the whole file
	data, err := ioutil.ReadAll(client.ledger)
	if err != nil {
		return nil, err
	}

	// and parse it into the raw transaction list.
	client.raw, err = parse.ParseLedger(string(data))
	if err != nil {
		return nil, err
	}

	// Now we need to transform the raw transaction list into the various filtered lists.
	client.byid = map[string][]ledger.Transaction{}
	client.simpleid = map[string]int{}
	for _, tr := range client.raw {
		if tr.Code == "" {
			return nil, MissingIDError
		}

		client.byid[tr.Code] = append(client.byid[tr.Code], tr)

		// The last transaction with a given ID is the authoritative version of that transaction.
		// However, "source order" of the transaction list is defined by the first version of that
		// transaction. This makes things really simple (not intended, it just worked out that way).
		if idx, ok := client.simpleid[tr.Code]; ok {
			client.simple[idx] = tr
			continue
		}
		client.simpleid[tr.Code] = len(client.simple)
		client.simple = append(client.simple, tr)
	}

	// Ok, we have the lists filled, but the simple list is in the source order of the original version of
	// each transaction. We want chronological first and then source so we need to do a stable sort by date.
	sort.Stable(ledger.TransactionDateSorter(client.simple))

	return client, nil
}

// Returned by AddTransactionEdit if there is not a parent transaction for the edit.
var MissingParentError = errors.New("Transaction edit does not have a parent.")

var transactionIDService <-chan string

func init() {
	go func() {
		c := make(chan string)
		transactionIDService = c

		idsource := shortid.MustNew(16, shortid.DefaultABC, uint64(time.Now().UnixNano()))

		for {
			c <- idsource.MustGenerate()
		}
	}()
}

// AddTransaction writes a transaction to the log and adds it to the internal lists.
// The transaction object passed in will be modified to have an ID in the Code field.
func (client *Client) AddTransaction(tr ledger.Transaction) error {
	// Before we do anything, make sure the transaction is well formed.
	err := tr.Canonicalize()
	if err != nil {
		return err
	}

	// Grab the write lock.
	client.lock.Lock()
	defer client.lock.Unlock()

	// Now that we have ruled out a malformed transaction, give the transaction an ID.
	// We should never ever need it, but just in case we make sure there are no collisions.
	tr.Code = <-transactionIDService
	for _, ok := client.simpleid[tr.Code]; ok; {
		tr.Code = <-transactionIDService
	}

	// Next, write the new transaction to the log file. This is the most likely step to fail somehow.
	_, err = fmt.Fprintf(client.ledger, "\n%v", tr)
	if err != nil {
		return err
	}

	// Ok, the error conditions are out of the way, pollute our internal state.
	client.simpleid[tr.Code] = len(client.simple)
	client.simple = append(client.simple, tr)
	client.byid[tr.Code] = []ledger.Transaction{tr}
	client.Events <- &Event{Typ: EvntTypTrUpdate}
	return nil
}

// AddTransactionEdit does the same basic thing as AddTransaction, except it ensures that the transaction
// being added replaces an existing one.
func (client *Client) AddTransactionEdit(tr ledger.Transaction) error {
	// Before we do anything, make sure the transaction is well formed.
	err := tr.Canonicalize()
	if err != nil {
		return err
	}

	// Grab the write lock.
	client.lock.Lock()
	defer client.lock.Unlock()

	// And make sure it has at least one parent.
	_, ok := client.simpleid[tr.Code]
	if !ok {
		return MissingParentError
	}

	// Next, write the new transaction to the log file.
	_, err = fmt.Fprintf(client.ledger, "\n%v", tr)
	if err != nil {
		return err
	}

	// Adding an edit to the internal structures is simpler than adding a new transaction.
	client.simple[client.simpleid[tr.Code]] = tr
	client.byid[tr.Code] = append(client.byid[tr.Code], tr)
	client.Events <- &Event{Typ: EvntTypTrUpdate}
	return nil
}

// GetAccountList returns a sorted list of accounts.
func (client *Client) GetAccountList() []string {
	// Grab the read lock.
	client.lock.RLock()
	defer client.lock.RUnlock()

	accounts := map[string]bool{}
	for _, tr := range client.simple {
		for _, post := range tr.Postings {
			accounts[post.Account] = true
		}
	}

	accountsList := []string{}
	for account := range accounts {
		accountsList = append(accountsList, account)
	}
	sort.Strings(accountsList)
	return accountsList
}

// Filter enum for dates.
const (
	FilterAllDates = iota - 1
	FilterThisMonth
	FilterLastMonth
	FilterThisYear
	FilterLastYear
)

// Filter enum for states.
const (
	FilterAllStatus = iota - 1
	FilterClearStatus
	FilterPendingStatus
)

// GetBalances returns a ready to display balance overview for all accounts with transactions that fit the filter.
// Each element in the slice is a display row consisting of two items, the name of the row with pre-applied
// indentation as needed and the pre-formatted value for the row.
func (client *Client) GetBalances(dfilter int) ([][]string, error) {
	// Grab the read lock.
	client.lock.RLock()
	defer client.lock.RUnlock()

	trs := client.GetTransactions(dfilter, FilterAllStatus, nil)

	accounts, err := ledger.SumTransactions(trs)
	if err != nil {
		return nil, err
	}
	return ledger.FormatSums(accounts, "    "), nil
}

// GetTransactions returns the simplified transaction list (all edits resolved, etc), sorted by date and
// then source order. This list is further filtered by a time period, status, and tags.
func (client *Client) GetTransactions(dfilter int, sfilter int, tfilter map[string]bool) []ledger.Transaction {
	// Grab the read lock.
	client.lock.RLock()
	defer client.lock.RUnlock()

	today := time.Now()
	trs := []ledger.Transaction{}
	for _, tr := range client.simple {
		switch dfilter {
		case FilterThisMonth:
			if tr.Date.Month() == today.Month() && tr.Date.Year() == today.Year() {
				trs = stateFilterNode(trs, tr, sfilter, tfilter)
			}
		case FilterLastMonth:
			if tr.Date.Month() == today.AddDate(0, -1, 0).Month() && tr.Date.Year() == today.AddDate(0, -1, 0).Year() {
				trs = stateFilterNode(trs, tr, sfilter, tfilter)
			}
		case FilterThisYear:
			if tr.Date.Year() == today.Year() {
				trs = stateFilterNode(trs, tr, sfilter, tfilter)
			}
		case FilterLastYear:
			if tr.Date.Year() == today.AddDate(-1, 0, 0).Year() {
				trs = stateFilterNode(trs, tr, sfilter, tfilter)
			}
		default:
			trs = stateFilterNode(trs, tr, sfilter, tfilter)
		}
	}
	return trs
}

func stateFilterNode(trs []ledger.Transaction, tr ledger.Transaction, sfilter int, tfilter map[string]bool) []ledger.Transaction {
	switch sfilter {
	case FilterClearStatus:
		if tr.Status == ledger.StatusClear {
			trs = tagFilterNode(trs, tr, tfilter)
		}
	case FilterPendingStatus:
		if tr.Status == ledger.StatusPending {
			trs = tagFilterNode(trs, tr, tfilter)
		}
	default:
		trs = tagFilterNode(trs, tr, tfilter)
	}
	return trs
}

func tagFilterNode(trs []ledger.Transaction, tr ledger.Transaction, tfilter map[string]bool) []ledger.Transaction {
	if tfilter == nil || len(tfilter) == 0 {
		return append(trs, tr)
	}

	for t := range tr.Tags {
		if tfilter[t] {
			return append(trs, *tr.CleanCopy())
		}
	}
	return trs
}

// GetTransactionWithHistory returns all existing versions of a transaction in source order. The last
// transaction in the list is the one currently in effect.
// In case of a non-existent ID, nil is returned.
func (client *Client) GetTransactionWithHistory(id string) []ledger.Transaction {
	// Grab the read lock.
	client.lock.RLock()
	defer client.lock.RUnlock()

	// Make a perfectly clean copy.
	trs := []ledger.Transaction{}
	for _, tr := range client.byid[id] {
		trs = append(trs, *tr.CleanCopy())
	}
	return trs
}

var attachmentIDService <-chan string

func init() {
	go func() {
		c := make(chan string)
		attachmentIDService = c

		idsource := shortid.MustNew(16, shortid.DefaultABC, uint64(time.Now().UnixNano()))

		for {
			c <- idsource.MustGenerate()
		}
	}()
}

// AddAttachment adds a attachments to a transaction, specified by an id.
func (client *Client) AddAttachment(id string, path string) error {
	// Grab an id for this attachment
	aid := <-attachmentIDService

	client.lock.RLock()

	// Get the transaction.
	trs, ok := client.byid[id]
	if !ok {
		client.lock.RUnlock()
		return errors.New("Transaction not found.")
	}

	// Get a clean copy of the transaction, ready to edit.
	tr := *trs[len(trs)-1].CleanCopy()

	client.lock.RUnlock()

	// Open the file for reading.
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Try to isolate the file extension of the original file name.
	parts := strings.Split(path, ".")
	ext := parts[len(parts)-1]
	if len(ext) > 4 {
		// Probably not actually an extension, so use a default.
		ext = "err"
	}

	// Open a location to write a copy of the file to our own storage location.
	nfile, err := os.Create("./attachments/" + aid + "." + ext)
	if err != nil {
		return err
	}
	defer nfile.Close()

	// And do the copying.
	nfile.ReadFrom(file)

	// Edit the transaction to include the attachment
	srawats, ok := tr.KVPairs["Attachments"]
	rawats := []byte(srawats)
	ats := []string{}
	if ok {
		err := json.Unmarshal(rawats, &ats)
		if err != nil {
			return err
		}
	}

	ats = append(ats, aid)

	rawats, err = json.Marshal(ats)
	if err != nil {
		return err
	}
	tr.KVPairs["Attachments"] = string(rawats)

	// Submit the transaction as an edit.
	client.AddTransactionEdit(tr)
	return nil
}
