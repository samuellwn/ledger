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

	"github.com/samuellwn/ledger"
)

var defaultAccount string = "Unknown:Account"

// FromOFX pulls transaction data from an OFX file and converts it to a File. On error os.Exit is called and
// the error is logged to standard error.
//
// This function makes a lot of assumptions about the structure of the input OFX file, and will error out if
// they are not met.
func FromOFX(file io.Reader, mainAccount string, descSrc ledger.OFXDescSrc, matchers []ledger.Matcher) *ledger.File {
	journal := &ledger.File{T: []ledger.Transaction{}, D: nil}

	HandleErr(journal.ImportOFX(file, descSrc, mainAccount, defaultAccount, "Equity:Balance Error"))
	journal.T = append(journal.T, journal.Matched(mainAccount, matchers)...)
	journal.StripHistory()

	return journal
}

func MergeOFX(journal *ledger.File, file io.Reader, mainAccount string, descSrc ledger.OFXDescSrc, matchers []ledger.Matcher) {
	HandleErr(journal.ImportOFX(file, descSrc, mainAccount, defaultAccount, "Equity:Balance Error"))
	journal.T = append(journal.T, journal.Matched(mainAccount, matchers)...)
}
