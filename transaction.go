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

/*
Package Ledger contains a parser for Ledger CLI transactions.

This should support the spec more-or-less fully for simple transactions,
but I did not add support for automated transactions or budgeting.

Additionally, I properly implemented String on everything so you can dump
Transactions to a file and read it with Ledger again.

Finally, there are a bunch of functions and methods for dealing with
transactions that should be helpful to anyone trying to use this for
real work.
*/
package ledger

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type status int

// Status constants for Transaction.Status
const (
	StatusUndefined = status(iota)
	StatusPending
	StatusClear
)

// Transaction is a single transaction from a ledger file.
type Transaction struct {
	Date        time.Time // 2020/10/10
	ClearDate   time.Time // =2020/10/10 (optional)
	Status      status    //   | ! | * (optional)
	Code        string    // ( Stuff ) (optional)
	Description string    // Spent monie on stuf

	Postings []Posting

	Comments []string // ; Stuff...

	Tags    map[string]bool   // ; :tag:tag:tag:
	KVPairs map[string]string // ; Key: Value

	Line int // The line number where the transaction starts.
}

// Posting is a single line item in a Transaction.
type Posting struct {
	Status  status //   | ! | *  (optional)
	Account string // Account:Name
	Value   int64  // $20.00 (currently only supporting USD, in thousandths of a cent)
	Null    bool   // True if the Value is implied. Value may or may not contain a valid amount.
	Note    string // ; Stuff
}

// CleanCopy takes a perfect copy of the transaction object, safe for editing without making any changes to the parent.
func (t *Transaction) CleanCopy() *Transaction {
	nt := *t
	nt.Postings = slices.Clone(t.Postings)
	nt.Comments = slices.Clone(t.Comments)
	nt.Tags = maps.Clone(t.Tags)
	nt.KVPairs = maps.Clone(t.KVPairs)
	return &nt
}

// Balance ensures that all postings in the transaction add up to 0 or there is a single null posting.
// Returns false, nil if there is more than one null posting, otherwise returns the ending balances of
// all accounts with postings and true if the transaction balances to 0 or there was a null posting.
func (t *Transaction) Balance() (bool, map[string]int64) {
	bal := int64(0)
	null := -1
	accounts := map[string]int64{}

	for i, p := range t.Postings {
		if p.Null && null != -1 {
			return false, nil // Multiple null postings
		}
		if p.Null {
			null = i
			continue
		}
		bal += p.Value
		accounts[p.Account] += p.Value
	}
	if null != -1 {
		accounts[t.Postings[null].Account] += -bal
		return true, accounts
	}
	return bal == 0, accounts
}

// Canonicalize takes a transaction and sets the value of any null postings that may exist to
// the required value to make it balance. Returns an error if there are multiple null postings or
// if there are no null postings and the transaction does not balance.
func (t *Transaction) Canonicalize() error {
	bal := int64(0)
	null := -1

	for i, p := range t.Postings {
		if p.Null && null != -1 {
			return MultipleNullError([2]int{-1, t.Line})
		}
		if p.Null {
			null = i
			continue
		}
		bal += p.Value
	}
	if null != -1 {
		t.Postings[null].Value = -bal
		return nil
	}
	if bal != 0 {
		return BalanceError([2]int{-1, t.Line})
	}
	return nil
}

// SumTransactions balances a list of transactions, and returns a map of accounts to their ending values.
func SumTransactions(ts []Transaction) (map[string]int64, error) {
	accounts := map[string]int64{}

	for i, t := range ts {
		ok, ac := t.Balance()
		if !ok {
			return nil, BalanceError([2]int{i, t.Line})
		}

		for k, v := range ac {
			accounts[k] += v
		}
	}

	return accounts, nil
}

type sumTree struct {
	children map[string]*sumTree
	value    int64
}

func (st *sumTree) render(name, lvl, pad string, res [][]string) [][]string {
	if len(st.children) == 1 {
		// Maybe I'm being an idiot, but there isn't a way to get an unknown key from a map that isn't a loop.
		for key, child := range st.children {
			return child.render(name+":"+key, lvl, pad, res)
		}
	}

	padding := ""
	if name != "" {
		padding = pad
		res = append(res, []string{lvl + name, FormatValue(st.value)})
	}

	keys := make([]string, 0, len(st.children))
	for key := range st.children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		res = st.children[key].render(key, lvl+padding, pad, res)
	}
	return res
}

// FormatSums takes a map of accounts to sums and turns it into a list of name/value pairs
// with indentation applied to the names.
func FormatSums(accounts map[string]int64, pad string) [][]string {
	// Generate an accounts tree
	root := &sumTree{children: map[string]*sumTree{}}

	for account, value := range accounts {
		parts := strings.Split(account, ":")

		level := root
		for _, part := range parts {
			if level.children == nil {
				level.children = map[string]*sumTree{}
			}
			if level.children[part] == nil {
				level.children[part] = &sumTree{}
			}
			level.children[part].value += value
			level = level.children[part]
		}
	}

	return root.render("", "", pad, nil)
}

func (t *Transaction) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(t.Date.Format("2006/01/02"))
	if !t.ClearDate.IsZero() {
		fmt.Fprintf(buf, "=%v", t.ClearDate.Format("2006/01/02"))
	}

	switch t.Status {
	case StatusClear:
		buf.WriteString(" * ")
	case StatusPending:
		buf.WriteString(" ! ")
	default:
		buf.WriteString("   ")
	}

	if t.Code != "" {
		fmt.Fprintf(buf, "(%v) ", t.Code)
	}

	fmt.Fprintf(buf, "%v\n", t.Description)

	// We don't know if the comments and postings were interleaved in any way,
	// so canonically we will just do the comments and metadata first.
	for _, line := range t.Comments {
		fmt.Fprintf(buf, "\t; %v\n", line)
	}
	if len(t.Tags) != 0 {
		fmt.Fprint(buf, "\t; ")
		for tag := range t.Tags {
			fmt.Fprintf(buf, ":%v", tag)
		}
		fmt.Fprint(buf, ":\n")
	}
	for k, v := range t.KVPairs {
		fmt.Fprintf(buf, "\t; %v: %v\n", k, v)
	}

	for _, p := range t.Postings {
		fmt.Fprintf(buf, "\t%v\n", p)
	}

	return buf.String()
}

func (p *Posting) String() string {
	buf := new(bytes.Buffer)

	switch p.Status {
	case StatusClear:
		buf.WriteString("* ")
	case StatusPending:
		buf.WriteString("! ")
	default:
		// This would pad all lines to the same length, but since these clear indicators are not common
		// adding them would just look like a bug (ask me how I know...)
		//buf.WriteString("  ")
	}

	// TODO: It would be nice to align on the decimal point instead of the first
	// digit, although that would be a lot harder.
	if !p.Null {
		fmt.Fprintf(buf, "%-50s", p.Account)

		if p.Value >= 0 {
			buf.WriteString(" ")
		}

		buf.WriteString(FormatValue(p.Value))
	} else {
		buf.WriteString(p.Account)
	}

	if p.Note != "" {
		fmt.Fprintf(buf, " ; %v", p.Note)
	}

	return buf.String()
}

// ParseValueNumber takes a decimal number and converts it to a integer with a precision of .
// Rounding is done via the round to even method.
func ParseValueNumber(v string) (int64, error) {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}

	ip := int64(f)
	fp := f - float64(ip)
	return ip*10000 + int64(fp*10000), nil
}

// FormatValue takes a amount of money in thousandths of a cent and formats it for display.
// Rounding is done via the round to even method.
func FormatValue(v int64) string {
	ms, ls1, ls2 := formatHelper(v)
	return fmt.Sprintf("$%v.%v%v", ms, ls1, ls2)
}

// FormatValueNumber is exactly the same as FormatValue, but it does not add any currency indicators.
func FormatValueNumber(v int64) string {
	ms, ls1, ls2 := formatHelper(v)
	return fmt.Sprintf("%v.%v%v", ms, ls1, ls2)
}

func formatHelper(v int64) (ms, ls1, ls2 int64) {
	// This is a little complicated because I not only need to separate the parts, but I also
	// want to round the result to even. There is probably a better way to do this.
	ms = v / 10000
	ls := v % 10000 / 100
	if ls < 0 {
		ls = -ls
	}
	ls = roundToEven(ls, v%100/10)
	if ls > 99 {
		ls = 0
		ms++
	}

	ls1 = ls / 10
	ls2 = ls % 10

	return
}

// Loosely based on the standard library math function.
// ls must be a value between 0 and 9 (a single decimal digit)
func roundToEven(ms, ls int64) int64 {
	odd := (ms % 2) != 0
	if ls > 5 || (ls == 5 && odd) {
		if ms > 0 {
			return ms + 1
		}
		return ms - 1
	}
	return ms
}

// TransactionDateSorter is a helper for sorting a list of transactions by date.
type TransactionDateSorter []Transaction

func (tds TransactionDateSorter) Len() int {
	return len(tds)
}

func (tds TransactionDateSorter) Less(i, j int) bool {
	return tds[i].Date.Before(tds[j].Date)
}

func (tds TransactionDateSorter) Swap(i, j int) {
	tds[i], tds[j] = tds[j], tds[i]
}

// Error types

// BalanceError is returned by functions that validate transactions in some way when the transaction isn't balanced.
type BalanceError [2]int

func (err BalanceError) Error() string {
	if err[0] < 0 {
		return fmt.Sprintf("Transaction (defined on line %v) does not balance.", err[1])
	}
	return fmt.Sprintf("Transaction %v (defined on line %v) does not balance.", err[0], err[1])
}

// MultipleNullError is returned by functions that validate transactions in some way when the transaction has more
// than one null posting.
type MultipleNullError [2]int

func (err MultipleNullError) Error() string {
	if err[0] < 0 {
		return fmt.Sprintf("Transaction (defined on line %v) has multiple null postings.", err[1])
	}
	return fmt.Sprintf("Transaction %v (defined on line %v) has multiple null postings.", err[0], err[1])
}
