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
	"github.com/milochristiansen/ledger"
)

var defaultAccount string = "Unknown:Account"

func FromOFX(file io.Reader, mainAccount string, matchers []ledger.Matcher) *ledger.File {
	// Load OFX file
	ofxd := HandleErrV(ofxgo.ParseResponse(file))

	// Convert it to ledger transactions
	HandleErrS(len(ofxd.Bank) != 1, "More banks than expected.")

	b, ok := ofxd.Bank[0].(*ofxgo.StatementResponse)
	HandleErrS(!ok, "Unexpected response type.")
	HandleErrS(len(b.BankTranList.Transactions) == 0, "No transactions.")

	trs := []ledger.Transaction{}
	for _, str := range b.BankTranList.Transactions {
		v := HandleErrV(ledger.ParseValueNumber(str.TrnAmt.String()))

		tr := ledger.Transaction{
			Description: string(str.Memo),
			Date:        str.DtPosted.Time,
			Status:      ledger.StatusClear,
			KVPairs: map[string]string{
				"ID":       <-ledger.IDService,
				"RID":      <-ledger.IDService,
				"FITID":    string(str.FiTID),
				"TrnTyp":   str.TrnType.String(),
				"FullDesc": string(str.Memo),
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

		trs = append(trs, tr)
	}
	return &ledger.File{T: trs, D: nil}
}
