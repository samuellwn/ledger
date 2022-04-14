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
	f1 := os.Args[2]
	f2 := os.Args[3]

	f1r, err := os.Open(f1)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	f1trs, f1drs, err := parse.ParseLedgerRaw(parse.NewRawCharReader(bufio.NewReader(f1r), 1))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	f2r, err := os.Open(f2)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	f2trs, f2drs, err := parse.ParseLedgerRaw(parse.NewRawCharReader(bufio.NewReader(f2r), 1))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Merge the directives. This is painful, but I'm too lazy to figure out a better way.
	drs := []ledger.Directive{}
	drs = append(drs, f1drs...)
	outer: for _, d2 := range f2drs {
		for _, d1 := range f1drs {
			if d2.Compare(d1) {
				continue outer
			}
		}
		drs = append(drs, d2)
	}
	for _, d := range drs {
		d.FoundBefore = 0
	}

	// Merge transactions.
	trs := []ledger.Transaction{}

	// First, zoom through the master file until we find the sync point.
	syncPoint := len(f1trs)-1
	for ; syncPoint >= 0; syncPoint-- {
		if f1trs[syncPoint].Code == f2trs[0].Code {
			break
		}
	}
	if syncPoint == len(f1trs) {
		fmt.Println("No sync point found!")
		os.Exit(1)
	}

	// Add transactions from the master up to the sync point
	for i := 0; i <= syncPoint; i++ {
		trs = append(trs, f1trs[i])
	}

	// Now continue adding files from the master up until the last transaction that matches.
	i1, i2 := syncPoint+1, 1
	for i1 < len(f1trs) || i2 < len(f2trs) {
		if f1trs[i1].Code != f2trs[i2].Code {
			break
		}
		trs = append(trs, f1trs[i1])
		i1++
		i2++
	}

	// Now zipper the differences together from the sync point
	for i1 < len(f1trs) || i2 < len(f2trs) {
		// If only one side is left, just append it and bail.
		if i1 >= len(f1trs) {
			trs = append(trs, f2trs[i2])
			i2++
			continue
		}
		if i2 >= len(f2trs) {
			trs = append(trs, f1trs[i1])
			i1++
			continue
		}

		// If there is a clear difference between the times, the earlier one goes first.
		if f1trs[i1].Date.Before(f2trs[i2].Date) {
			trs = append(trs, f1trs[i1])
			i1++
			continue
		}
		if f1trs[i1].Date.After(f2trs[i2].Date) {
			trs = append(trs, f2trs[i2])
			i2++
			continue
		}

		// Otherwise, order them by lexical comparison of the ID
		if f1trs[i1].Code < f2trs[i2].Code {
			trs = append(trs, f1trs[i1])
			i1++
			continue
		}
		trs = append(trs, f2trs[i2])
		i2++
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

  zipper dest master source

This program takes two ledger files and "zips" them together to make a single
file. All directives will be moved to the beginning of the file!

For this to work properly, each transaction needs the "code" field to be a
unique transaction ID, otherwise it is not possible to sync partial files
and syncing full files is not deterministic.
`
