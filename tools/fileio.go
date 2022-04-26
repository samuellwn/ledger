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

// Guts contains random common code used for implementing the various tools.
package tools

import (
	"bufio"
	"fmt"
	"os"

	"github.com/milochristiansen/ledger"
	"github.com/milochristiansen/ledger/parse"
)

// LoadLedgerFile loads a ledger file from the given path. On any error the message is logged to standard error and the
// program exits with code 1.
func LoadLedgerFile(path string) ([]ledger.Transaction, []ledger.Directive) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	trs, drs, err := parse.ParseLedgerRaw(parse.NewRawCharReader(bufio.NewReader(f), 1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return trs, drs
}

