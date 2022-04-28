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
	"strings"

	"github.com/milochristiansen/ledger/parse/lex"
)

type File struct {
	T []Transaction
	D []Directive // Must be ordered so that the FoundBefore values are ascending!
}

// ErrImproperInterleave is returned by File.Format if the lists do not interleave properly.
// Caused by bad FoundBefore values in the directives.
var ErrImproperInterleave = errors.New("Ledger file transaction and directive lists do not interleave properly.")

// Format writes out a ledger file, interleaving the transactions and directives according to the
// "FoundBefore" values in the directives.
func (f *File) Format(w io.Writer) error {
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

		// filter out some things that cause funny behovior
		if strings.Contains(acct.Name, "  ") || strings.ContainsAny(acct.Name, ";\t") {
			return nil, ErrMalformedAccountName{acct.Name, acct.Location}
		}

		for sdIx, sd := range d.Lines {
			if strings.HasPrefix(sd, "default") {
				// ledger is lax about directive parsing
				acct.Default = true
			} else if strings.HasPrefix(sd, "alias") {
				alias := strings.TrimSpace(sd[len("alias"):])
				// filter out some things that cause funny behovior
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
	DirectiveIndex int          // The index of this account in the list of all directives.
	Location       lex.Location // Line number where this account starts.
}

// Payee is a simple type representing a payee directive.
type Payee struct {
	Name    string   // The payee name to substitute if matched
	Aliases []string // One string for each regexp to match with.
	Uuids   []string // One string for each uuid to check.

	FoundBefore    int          // The transaction index this directive precedes.
	DirectiveIndex int          // The index of this directive in the list of all directives.
	Location       lex.Location // Line number where this directive starts.
}
