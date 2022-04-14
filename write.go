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
)

// ErrImproperInterleave is returned by WriteLedgerFile if the two provided lists do not interleave properly.
// Caused by bad FoundBefore values in the directives.
var ErrImproperInterleave = errors.New("Ledger file transaction and directive lists do not interleave properly.")

// WriteLedgerFile writes out a ledger file, interleaving the transactions and directives according to the
// "FoundBefore" values in the directives. drs must be ordered so that the FoundBefore values are ascending.
func WriteLedgerFile(w io.Writer, trs []Transaction, drs []Directive) error {
	ctr, cdr := 0, 0
	for ctr < len(trs) || cdr < len(drs) {
		// If we have remaining directives and the next directive goes before the current transaction
		if cdr < len(drs) && drs[cdr].FoundBefore == ctr {
			fmt.Fprintf(w, "\n%v", drs[cdr].String())
			cdr++
			continue
		}

		// If we have remaining directives and we are out of transactions
		if ctr >= len(trs) {
			return ErrImproperInterleave
		}

		// Write next transaction
		fmt.Fprintf(w, "\n%v", trs[ctr].String())
		ctr++
	}
	return nil
}
