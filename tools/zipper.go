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
	"errors"

	"github.com/milochristiansen/ledger"
)

// Zipper takes two ledger flies and zips them together in a deterministic manner. On error os.Exit is called and
// the error is logged to standard error.
// All directives are deduplicated and moved to the top of the file.
func Zipper(a *ledger.File, b *ledger.File) *ledger.File {
	return HandleErrV(ZipperHTTP(a, b))
}

// ZipperHTTP is like Zipper, but intended for use in HTTPhandlers and the like where the standard command
// error handling is not desirable.
func ZipperHTTP(a *ledger.File, b *ledger.File) (*ledger.File, error) {
	drs := []ledger.Directive{}
	drs = append(drs, a.D...)
outer:
	for _, d2 := range b.D {
		for _, d1 := range a.D {
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
	syncPoint := len(a.T) - 1
	for ; syncPoint >= 0; syncPoint-- {
		if a.T[syncPoint].Code == b.T[0].Code {
			break
		}
	}
	if syncPoint == len(a.T) {
		return nil, errors.New("No sync point found!")
	}

	// Add transactions from the master up to the sync point
	for i := 0; i <= syncPoint; i++ {
		trs = append(trs, a.T[i])
	}

	// Now continue adding files from the master up until the last transaction that matches.
	i1, i2 := syncPoint+1, 1
	for i1 < len(a.T) || i2 < len(b.T) {
		if a.T[i1].Code != b.T[i2].Code {
			break
		}
		trs = append(trs, a.T[i1])
		i1++
		i2++
	}

	// Now zipper the differences together from the last sync point
	for i1 < len(a.T) || i2 < len(b.T) {
		// If only one side is left, just append it and bail.
		if i1 >= len(a.T) {
			trs = append(trs, b.T[i2])
			i2++
			continue
		}
		if i2 >= len(b.T) {
			trs = append(trs, a.T[i1])
			i1++
			continue
		}

		// If there is a clear difference between the times, the earlier one goes first.
		if a.T[i1].Date.Before(b.T[i2].Date) {
			trs = append(trs, a.T[i1])
			i1++
			continue
		}
		if a.T[i1].Date.After(b.T[i2].Date) {
			trs = append(trs, b.T[i2])
			i2++
			continue
		}

		// if the times are the same, try to order lexically by ID to preserve determinism.
		dir := chooseAB(a.T[i1].KVPairs, b.T[i2].KVPairs, "ID")
		if dir < 0 {
			trs = append(trs, a.T[i1])
			i1++
			continue
		}
		if dir > 0 {
			trs = append(trs, b.T[i2])
			i2++
			continue
		}

		// Well, we can't order by ID for some reason. Try to order by the revision ID (only present in edits)
		dir = chooseAB(a.T[i1].KVPairs, b.T[i2].KVPairs, "RID")
		if dir < 0 {
			trs = append(trs, a.T[i1])
			i1++
			continue
		}
		if dir > 0 {
			trs = append(trs, b.T[i2])
			i2++
			continue
		}

		// If all else fails, try to use a financial institution ID (only present in imported data)
		dir = chooseAB(a.T[i1].KVPairs, b.T[i2].KVPairs, "FITID")
		if dir < 0 {
			trs = append(trs, a.T[i1])
			i1++
			continue
		}
		if dir > 0 {
			trs = append(trs, b.T[i2])
			i2++
			continue
		}
		return nil, errors.New("Error: Could not order some transactions. Ensure all transactions have ID and RID keys as appropriate.")
	}
	return &ledger.File{T: trs, D: drs}, nil
}

// -1 == a, 0 == neither, 1 == b
func chooseAB(a, b map[string]string, key string) int {
	id1, ok1 := a[key]
	id2, ok2 := b[key]

	// If only one has an ID, the ID goes first.
	if ok1 && !ok2 {
		return -1
	}
	if !ok1 && ok2 {
		return 1
	}

	// If neither has an ID
	if !ok1 && !ok2 {
		return 0
	}

	// If both have identical IDs
	if id1 == id2 {
		return 0
	}

	// If both have an ID then order by ID lexically.
	if id1 < id2 {
		return -1
	}
	return 1
}
