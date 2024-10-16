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

package ledger

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/aclindsa/ofxgo"
	"github.com/samuellwn/ledger/parse/lex"
)

// File hold a parsed ledger file stored as lists of Directives and Transactions.
type File struct {
	T []Transaction
	D []Directive
}

// ErrImproperInterleave is returned by File.Format if the lists do not interleave properly.
// Caused by bad FoundBefore values in the directives.
var ErrImproperInterleave = errors.New("Ledger file transaction and directive lists do not interleave properly.")

// Format writes out a ledger file, interleaving the transactions and directives according to the
// "FoundBefore" values in the directives. The directive list is sorted on the FoundBefore values as
// part of this operation.
func (f *File) Format(w io.Writer) error {
	// Use a stable sort to be minimally disruptive.
	sort.SliceStable(f.D, func(i, j int) bool {
		return f.D[i].FoundBefore < f.D[j].FoundBefore
	})

	ctr, cdr := 0, 0
	for ctr < len(f.T) || cdr < len(f.D) {
		// If we have remaining directives and the next directive goes before the current transaction
		if cdr < len(f.D) && f.D[cdr].FoundBefore == ctr {
			fmt.Fprintf(w, "\n%v", f.D[cdr].String())
			cdr++
			continue
		}

		// If we have remaining directives and we are out of transactions
		if ctr >= len(f.T) {
			return ErrImproperInterleave
		}

		// Write next transaction
		fmt.Fprintf(w, "\n%v", f.T[ctr].String())
		ctr++
	}
	return nil
}

// ErrMalformedAccountName is returned by File.Accounts if an account name is malformed.
type ErrMalformedAccountName struct {
	Name     string
	Location lex.Location
}

func (err ErrMalformedAccountName) Error() string {
	return fmt.Sprintf("Malformed account name (%s) at %s", err.Name, err.Location)
}

// Accounts returns a slice of all account directives, in the order they are found in D.
// If any account directives fail to parse, Accounts returns an error.
func (f *File) Accounts() ([]Account, error) {
	accts := []Account{}
	for dIx, d := range f.D {
		if d.Type != "account" {
			continue
		}

		acct := Account{
			Name:           d.Argument,
			FoundBefore:    d.FoundBefore,
			Location:       d.Location,
			DirectiveIndex: dIx,
		}

		// filter out some things that cause funny behavior
		if strings.Contains(acct.Name, "  ") || strings.ContainsAny(acct.Name, ";\t") {
			return nil, ErrMalformedAccountName{acct.Name, acct.Location}
		}

		for sdIx, sd := range d.Lines {
			if strings.HasPrefix(sd, "default") {
				// ledger is lax about directive parsing
				acct.Default = true
			} else if strings.HasPrefix(sd, "alias") {
				alias := strings.TrimSpace(sd[len("alias"):])
				// filter out some things that cause funny behavior
				if strings.Contains(alias, "  ") || strings.ContainsAny(alias, ";\t") {
					return nil, ErrMalformedAccountName{
						Name:     alias,
						Location: acct.Location.L(acct.Location.Line() + uint64(sdIx)),
					}
				}
				acct.Aliases = append(acct.Aliases, alias)
			} else if strings.HasPrefix(sd, "payee") {
				payee := strings.TrimSpace(sd[len("payee"):])
				acct.Payees = append(acct.Payees, payee)
			} else if strings.HasPrefix(sd, "note") {
				note := strings.TrimSpace(sd[len("note"):])
				acct.Note = note
			}
		}

		accts = append(accts, acct)
	}
	return accts, nil
}

// Payees returns a slice of all payee directives, in the order they are found in D.
// if any payee directives fail to parse, Payees returns an error.
func (f *File) Payees() ([]Payee, error) {
	payees := []Payee{}
	for dIx, d := range f.D {
		if d.Type != "account" {
			continue
		}

		payee := Payee{
			Name:           d.Argument,
			FoundBefore:    d.FoundBefore,
			Location:       d.Location,
			DirectiveIndex: dIx,
		}

		for _, sd := range d.Lines {
			if strings.HasPrefix(sd, "alias") {
				alias := strings.TrimSpace(sd[len("alias"):])
				payee.Aliases = append(payee.Aliases, alias)
			} else if strings.HasPrefix(sd, "uuid") {
				uuid := strings.TrimSpace(sd[len("uuid"):])
				payee.Uuids = append(payee.Uuids, uuid)
			}
		}

		payees = append(payees, payee)
	}
	return payees, nil
}

// Account is a simple type representing an account directive. Subdirectives containing value expressions
// are not included.
type Account struct {
	Name    string   // The name of this account.
	Note    string   // The contents of the note subdirective.
	Aliases []string // One string for each alias subdirective.
	Payees  []string // One string for each payee subdirective.
	Default bool     // True if the default subdirective is present.

	FoundBefore    int          // The transaction index this account precedes.
	DirectiveIndex int          // The index of this account in the list of all directives. Calling File.Format may ruin this relationship.
	Location       lex.Location // Line number where this account starts.
}

// Payee is a simple type representing a payee directive.
type Payee struct {
	Name    string   // The payee name to substitute if matched
	Aliases []string // One string for each regexp to match with.
	Uuids   []string // One string for each uuid to check.

	FoundBefore    int          // The transaction index this directive precedes.
	DirectiveIndex int          // The index of this directive in the list of all directives. Calling File.Format may ruin this relationship.
	Location       lex.Location // Line number where this directive starts.
}

// Matched finds transactions by regexp on the description, and returns a slice of found transactions
// with postings and description modified by the first successful match from matchers. Only transactions
// with a posting containing the given account will be modified.
func (f *File) Matched(account string, matchers []Matcher) []Transaction {
	outTrs := []Transaction{}
	for _, ftr := range f.T {
		tr := *ftr.CleanCopy()
		if tr.Match(account, matchers) {
			tr.KVPairs["RID"] = <-IDService
			outTrs = append(outTrs, tr)
		}
	}
	return outTrs
}

// ParseMatchers parses matchers from the directives of this ledger file.
func (f *File) ParseMatchers() ([]Matcher, error) {
	accounts, err := f.Accounts()
	if err != nil {
		return nil, err
	}

	payees, err := f.Payees()
	if err != nil {
		return nil, err
	}

	matchers := []Matcher{}

	// fill matcher slice
	for _, acct := range accounts {
		account := acct.Name

		pm := []Matcher{}
		for _, reStr := range acct.Payees {
			re, err := regexp.Compile(reStr)
			if err != nil {
				return nil, err
			}

			pm = append(pm, Matcher{
				Account: account,
				R:       re,
			})
		}

		for _, payee := range payees {
			for _, m := range pm {
				if m.R.MatchString(payee.Name) {
					for _, alias := range payee.Aliases {
						re, err := regexp.Compile(alias)
						if err != nil {
							return nil, err
						}

						matchers = append(matchers, Matcher{
							Account: account,
							Payee:   payee.Name,
							R:       re,
						})
					}
				}
			}
		}

		matchers = append(matchers, pm...)
	}

	return matchers, nil
}

type OFXDescSrc int

const (
	OFXDescName OFXDescSrc = iota
	OFXDescMemo
	OFXDescNameMemo
)

// ImportOFX imports the OFX response/file into this file. Already imported transactions will be skipped.
func (f *File) ImportOFX(ofxFile io.Reader, descSrc OFXDescSrc, bankAcct, defaultAcct, mismatchAcct string) error {
	// Load OFX file
	ofxd, err := ofxgo.ParseResponse(ofxFile)
	if err != nil {
		return err
	}

	// Load set of seen transaction ids from ofx
	seenIds := map[string]bool{}
	for _, tr := range f.T {
		if tr.KVPairs["FITID"] == "" || tr.KVPairs["Account"] != bankAcct {
			continue
		}
		seenIds[tr.KVPairs["FITID"]] = true
	}

	// Convert it to ledger transactions
	if len(ofxd.Bank) == 0 && len(ofxd.CreditCard) == 0 {
		return errors.New("No banks or credit cards.")
	}

	ltrns := []Transaction{}
	for _, msg := range append(ofxd.Bank, ofxd.CreditCard...) {
		var trns []ofxgo.Transaction
		if b, ok := msg.(*ofxgo.StatementResponse); ok {
			trns = b.BankTranList.Transactions
		} else if cc, ok := msg.(*ofxgo.CCStatementResponse); ok {
			trns = cc.BankTranList.Transactions
		} else {
			return errors.New("Unexpected response type.")
		}

		for _, str := range trns {
			v, err := ParseValueNumber(str.TrnAmt.String())
			if err != nil {
				return err
			}

			desc := ""
			switch descSrc {
			case OFXDescName:
				desc = string(str.Name)
			case OFXDescMemo:
				desc = string(str.Memo)
			case OFXDescNameMemo: // because some banks output braindead OFX files
				desc = string(str.Name + str.Memo)
			}

			tr := Transaction{
				Description: desc,
				Date:        str.DtPosted.Time,
				Status:      StatusUndefined,
				KVPairs: map[string]string{
					"ID":      <-IDService,
					"RID":     <-IDService,
					"FITID":   string(str.FiTID),
					"TrnTyp":  str.TrnType.String(),
					"Memo":    string(str.Memo),
					"Name":    string(str.Name),
					"Account": bankAcct,
				},
				Postings: []Posting{
					{
						Account: bankAcct,
						Value:   v,
					},
					{
						Account: defaultAcct,
						Null:    true,
					},
				},
			}

			ltrns = append(ltrns, tr)
		}
	}

	for _, msg := range append(ofxd.Bank, ofxd.CreditCard...) {
		var bal ofxgo.Amount
		var asOf ofxgo.Date
		if b, ok := msg.(*ofxgo.StatementResponse); ok {
			bal = b.BalAmt
			asOf = b.DtAsOf
		} else if cc, ok := msg.(*ofxgo.CCStatementResponse); ok {
			bal = cc.BalAmt
			asOf = cc.DtAsOf
		} else {
			return errors.New("Unexpected response type.")
		}

		v, err := ParseValueNumber(bal.String())
		if err != nil {
			return err
		}

		var postings []Posting
		if len(mismatchAcct) == 0 {
			postings = []Posting{{
				Account:   bankAcct,
				Value:     0,
				Assert:    v,
				HasAssert: true,
			}}
		} else {
			postings = []Posting{{
				Account:   bankAcct,
				Null:      true,
				Assert:    v,
				HasAssert: true,
			}, {
				Account: mismatchAcct,
				Null:    true,
			}}
		}
		tr := Transaction{
			Description: "Statement Ending Balance",
			Date:        asOf.Time,
			Status:      StatusUndefined,
			KVPairs: map[string]string{
				"ID":            <-IDService,
				"RID":           <-IDService,
				"EndingBalance": bankAcct,
			},
			Postings: postings,
		}

		ltrns = append(ltrns, tr)
	}

	f.T = append(f.T, ltrns...)
	return nil
}

// CleanCopy takes a perfect copy of the file object. Any edits to the returned File
// will not modify this method's receiver.
func (f *File) CleanCopy() *File {
	nf := &File{[]Transaction{}, []Directive{}}

	for _, tr := range f.T {
		nf.T = append(nf.T, *tr.CleanCopy())
	}

	for _, dir := range f.D {
		nf.D = append(nf.D, *dir.CleanCopy())
	}

	return nf
}

// StripHistory removes all edit history. This method assumes all directives
// are at the beginning of the file. If any directive has a FoundBefore greater
// than 0 data corruption can occur.
func (f *File) StripHistory() {
	newTrs := []Transaction{}
	trIxs := map[string]int{}
	for _, tr := range f.T {
		id, ok := tr.KVPairs["ID"]
		if !ok || id == "" {
			newTrs = append(newTrs, tr)
			continue
		}

		if idx, ok := trIxs[id]; ok {
			newTrs[idx] = tr
			continue
		}

		trIxs[id] = len(newTrs)
		newTrs = append(newTrs, tr)
	}

	f.T = newTrs
}
