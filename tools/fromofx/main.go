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
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/milochristiansen/ledger"
	"github.com/teris-io/shortid"

	"github.com/aclindsa/ofxgo"
)

type Matcher struct {
	R       *regexp.Regexp
	Account string
	Payee   string
}

func main() {
	if len(os.Args) < 4 || (len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help")) {
		fmt.Print(usage)
		return
	}

	dest := os.Args[1]
	fp := os.Args[2]
	masterAccount := os.Args[3]
	matchf := ""
	if len(os.Args) > 4 {
		matchf = os.Args[4]
	}

	fr, err := os.Open(fp)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	matchers := []Matcher{}
	if matchf != "" {
		mr, err := os.Open(matchf)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		mrdr := csv.NewReader(mr)
		mrdr.FieldsPerRecord = 3
		mrdr.Comment = '#'

		for {
			line, err := mrdr.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			reg, err := regexp.Compile(line[0])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			matchers = append(matchers, Matcher{
				reg,
				line[1],
				line[2],
			})
		}
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

		account, payee := "Unknown:Account", string(str.Memo)
		for _, matcher := range matchers {
			if matcher.R.MatchString(payee) {
				if matcher.Account != "" {
					account = matcher.Account
				}
				if matcher.Payee != "" {
					payee = matcher.Payee
				}
				break
			}
		}

		tr := ledger.Transaction{
			Description: payee,
			Date:        str.DtPosted.Time,
			Status:      ledger.StatusClear,
			KVPairs: map[string]string{
				"ID":       idsource.MustGenerate(),
				"RID":      idsource.MustGenerate(),
				"FITID":    string(str.FiTID),
				"TrnTyp":   str.TrnType.String(),
				"FullDesc": string(str.Memo),
			},
			Postings: []ledger.Posting{
				{
					Account: masterAccount,
					Value:   v,
				},
				{
					Account: account,
					Null:    true,
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

  fromofx dest source account [matchfile]

This program takes an OFX file and converts it to a ledger file.
`
