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

package main

import (
	"github.com/samuellwn/ledger/tools"
)

func main() {
	fs := tools.CommonFlagSet(tools.FlagDestFile|tools.FlagMasterFile|tools.FlagID|tools.FlagRID, usage)
	fs.Parse()

	f := tools.LoadLedgerFile(fs.MasterFile)

	rf := tools.LTail(f, fs.ID, fs.RID)

	tools.WriteLedgerFile(fs.DestFile, rf)
}

var usage = `Usage:

This program takes a ledger file and strips all content preceding the
transaction with the given ID.

For this to work properly, each transaction needs the "code" field to be a
unique transaction ID, otherwise it is not possible. Additionally, to ensure
proper operation on a file containing revision history, you may need to provide
the revision ID of the transaction to split upon.
`
