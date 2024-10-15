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

import "github.com/samuellwn/ledger/tools"

func main() {
	fs := tools.CommonFlagSet(tools.FlagDestFile | tools.FlagMasterFile | tools.FlagSourceFile, usage)
	fs.Parse()

	a := tools.LoadLedgerFile(fs.MasterFile)
	b := tools.LoadLedgerFile(fs.SourceFile)

	f := tools.Zipper(a, b)

	tools.WriteLedgerFile(fs.DestFile, f)
}

var usage = `Usage:

This program takes two ledger files and "zips" them together to make a single
file. All directives will be moved to the beginning of the file!

For this to work properly, each transaction needs an "ID" K/V to be set to a
unique transaction ID, otherwise it is not possible to sync partial files
and syncing full files is not deterministic. Any non-deterministic result is
an error.
`
