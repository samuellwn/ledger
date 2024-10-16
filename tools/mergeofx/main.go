/*
Copyright 2024 by Milo Christiansen

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

	"github.com/samuellwn/ledger"
	"github.com/samuellwn/ledger/tools"
)

func main() {
	fs := tools.CommonFlagSet(tools.FlagMasterFile|tools.FlagSourceFile|tools.FlagAccountName|tools.FlagMatchFile, usage)
	var descSrc ledger.OFXDescSrc
	fs.Flags.Func("desc", "Where to get the `description` from. \"name\", \"memo\", or \"name+memo\". (default \"name\")", func(s string) error {
		switch s {
		case "name":
			descSrc = ledger.OFXDescName
		case "memo":
			descSrc = ledger.OFXDescMemo
		case "name+memo":
			descSrc = ledger.OFXDescNameMemo
		default:
			return fmt.Errorf("Unknown description source: %q", s)
		}
		return nil
	})
	fs.Parse()

	fr := tools.HandleErrV(os.Open(fs.SourceFile))

	matchers := []ledger.Matcher{}
	if fs.MatchFile != "" {
		matchers = tools.LoadMatchFile(fs.MatchFile)
	}

	journal := tools.LoadLedgerFile(fs.MasterFile)

	tools.MergeOFX(journal, fr, fs.AccountName, descSrc, matchers)

	tools.WriteLedgerFile(fs.MasterFile, journal)
}

var usage = `Usage:

This program takes an OFX file and adds any unseen transactions to a ledger file.
`
