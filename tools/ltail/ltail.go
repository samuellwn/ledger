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
	"bufio"
	"fmt"
	"os"

	"github.com/milochristiansen/ledger"
	"github.com/milochristiansen/ledger/parse"
)

func main() {
	if len(os.Args) < 4 || (len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help")) {
		fmt.Print(usage)
		return
	}

	dest := os.Args[1]
	fp := os.Args[2]
	id := os.Args[3]
	rid := ""
	if len(os.Args) >= 5 {
		rid = os.Args[4]
	}

	f1r, err := os.Open(fp)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ftrs, fdrs, err := parse.ParseLedgerRaw(parse.NewRawCharReader(bufio.NewReader(f1r), 1))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Go through the transactions *in reverse* looking for the ID (and also the revision ID if specified)
	i := len(ftrs)-1
	for ; i >= 0; i-- {
		if fid, ok := ftrs[i].KVPairs["ID"]; ok && fid == id {
			if rid != "" {
				if frid, ok := ftrs[i].KVPairs["RID"]; ok && frid == rid {
					break
				}
				continue
			}
			break
		}
	} 

	// slice the transaction list to remove everything before that point.
	trs := ftrs[i:]

	// Now drop all the directives that come before the selected transaction
	drs := fdrs
	if len(fdrs) > 0 {
		j := 0
		for ; j < len(fdrs); j-- {
			if fdrs[j].FoundBefore > i {
				break
			}
		}
		drs = fdrs[j:]
	}


	out, err := os.Create(dest)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = ledger.WriteLedgerFile(out, trs, drs)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	out.Close()
}

var usage = `Usage:

  ltail dest master id [revision]

This program takes a ledger file and strips all content preceding the
transaction with the given ID.

For this to work properly, each transaction needs the "code" field to be a
unique transaction ID, otherwise it is not possible. Additionally, to ensure
proper operation on a file containing revision history, you may need to provide
the revision ID of the transaction to split upon.
`
