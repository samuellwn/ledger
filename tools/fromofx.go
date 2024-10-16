/*
Copyright 2022 by Milo Christiansen

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

package tools

import (
	"io"

	"github.com/aclindsa/ofxgo"
	"github.com/samuellwn/ledger"
)

var defaultAccount string = "Unknown:Account"

type OFXDescSrc int

const (
	OFXDescName OFXDescSrc = iota
	OFXDescMemo
	OFXDescNameMemo
)

// FromOFX pulls transaction data from an OFX file and converts it to a File. On error os.Exit is called and
// the error is logged to standard error.
//
// This function makes a lot of assumptions about the structure of the input OFX file, and will error out if
// they are not met.
func FromOFX(file io.Reader, mainAccount string, descSrc OFXDescSrc, matchers []ledger.Matcher) *ledger.File {
	journal := &ledger.File{T: []ledger.Transaction{}, D: nil}

	MergeOFX(journal, file, mainAccount, descSrc, matchers)

	return journal
}

func MergeOFX(journal *ledger.File, file io.Reader, mainAccount string, descSrc OFXDescSrc, matchers []ledger.Matcher) {
	// Load OFX file
	ofxd := HandleErrV(ofxgo.ParseResponse(file))

	// Convert it to ledger transactions
	HandleErrS(len(ofxd.Bank) == 0 && len(ofxd.CreditCard) == 0, "No banks or credit cards.")

	// Load set of seen transaction ids from ofx
	seenIds := map[string]bool{}
	for _, tr := range journal.T {
		if tr.KVPairs["FITID"] == "" {
			continue
		}
		for _, p := range tr.Postings {
			if p.Account == mainAccount {
				seenIds[tr.KVPairs["FITID"]] = true
			}
		}
	}

	for _, msg := range append(ofxd.Bank, ofxd.CreditCard...) {
		var trns []ofxgo.Transaction
		if b, ok := msg.(*ofxgo.StatementResponse); ok {
			trns = b.BankTranList.Transactions
		} else if cc, ok := msg.(*ofxgo.CCStatementResponse); ok {
			trns = cc.BankTranList.Transactions
		} else {
			HandleErrS(true, "Unexpected response type.")
		}

		for _, str := range trns {
			v := HandleErrV(ledger.ParseValueNumber(str.TrnAmt.String()))

			if seenIds[string(str.FiTID)] {
				continue
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

			tr := ledger.Transaction{
				Description: desc,
				Date:        str.DtPosted.Time,
				Status:      ledger.StatusUndefined,
				KVPairs: map[string]string{
					"ID":     <-ledger.IDService,
					"RID":    <-ledger.IDService,
					"FITID":  string(str.FiTID),
					"TrnTyp": str.TrnType.String(),
					"Memo":   string(str.Memo),
					"Name":   string(str.Name),
				},
				Postings: []ledger.Posting{
					{
						Account: mainAccount,
						Value:   v,
					},
					{
						Account: defaultAccount,
						Null:    true,
					},
				},
			}

			tr.Match(defaultAccount, matchers)

			journal.T = append(journal.T, tr)
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
			HandleErrS(true, "Unexpected response type.")
		}

		tr := ledger.Transaction{
			Description: "Statement Ending Balance",
			Date:        asOf.Time,
			Status:      ledger.StatusUndefined,
			KVPairs: map[string]string{
				"ID":  <-ledger.IDService,
				"RID": <-ledger.IDService,
			},
			Postings: []ledger.Posting{{
				Account:   mainAccount,
				Value:     0,
				Assert:    HandleErrV(ledger.ParseValueNumber(bal.String())),
				HasAssert: true,
			}},
		}

		journal.T = append(journal.T, tr)
	}
}
