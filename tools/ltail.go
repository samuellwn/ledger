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

import "github.com/samuellwn/ledger"

// LTail tails a ledger file based on a ID and RID. There are no error cases (if the ID doesn't exist you just get an empty file)
func LTail(f *ledger.File, id, rid string) *ledger.File {
	// Go through the transactions *in reverse* looking for the ID (and also the revision ID if specified)
	i := len(f.T) - 1
	for ; i >= 0; i-- {
		if fid, ok := f.T[i].KVPairs["ID"]; ok && fid == id {
			if rid != "" {
				if frid, ok := f.T[i].KVPairs["RID"]; ok && frid == rid {
					break
				}
				continue
			}
			break
		}
	}

	// slice the transaction list to remove everything before that point.
	rtrs := f.T[i:]

	// Now drop all the directives that come before the selected transaction
	rdrs := f.D
	if len(f.D) > 0 {
		j := 0
		for ; j < len(f.D); j++ {
			if f.D[j].FoundBefore > i {
				break
			}
		}
		rdrs = f.D[j:]

		// Adjust FoundBefore values
		for k := range rdrs {
			rdrs[k].FoundBefore -= i
		}
	}

	return &ledger.File{T: rtrs, D: rdrs}
}
