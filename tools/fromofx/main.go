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

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/milochristiansen/ledger"
	"github.com/teris-io/shortid"

	"github.com/aclindsa/ofxgo"
)

func main() {
	if len(os.Args) < 4 || (len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help")) {
		fmt.Print(usage)
		return
	}

	dest := os.Args[1]
	fp := os.Args[2]
	account := os.Args[3]

	fr, err := os.Open(fp)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Load OFX file
	ofxd, err := ofxgo.ParseResponse(fr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Convert it to ledger transactions
	if len(ofxd.Bank) != 1 {
		fmt.Println("More banks than expected.")
		os.Exit(1)
	}

	b, ok := ofxd.Bank[0].(*ofxgo.StatementResponse)
	if !ok {
		fmt.Println("Unexpected response type.")
		os.Exit(1)
	}

	if len(b.BankTranList.Transactions) == 0 {
		fmt.Println("No transactions.")
		os.Exit(1)
	}


	idsource := shortid.MustNew(16, shortid.DefaultABC, uint64(time.Now().UnixNano()))

	trs := []ledger.Transaction{}
	for i, str := range b.BankTranList.Transactions {
		v, err := ledger.ParseValueNumber(str.TrnAmt.String())
		if err != nil {
			fmt.Printf("Error parsing the amount on transaction %v:%v", i, str.Memo)
			os.Exit(1)
		}

		tr := ledger.Transaction{
			Description: string(str.Memo),
			Date: str.DtPosted.Time,
			Status: ledger.StatusClear,
			KVPairs: map[string]string{
				"ID": idsource.MustGenerate(),
				"RID": idsource.MustGenerate(),
				"FITID": string(str.FiTID),
				"TrnTyp": str.TrnType.String(),
			},
			Postings: []ledger.Posting{
				{
					Account: account,
					Value: v,
				},
				{
					Account: "Unknown:Account",
					Null: true,
				},
			},
		}

		trs = append(trs, tr)
	}

	out, err := os.Create(dest)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = ledger.WriteLedgerFile(out, trs, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	out.Close()
}

var usage = `Usage:

  fromofx dest source account

This program takes an OFX file and converts it to a ledger file.
`
