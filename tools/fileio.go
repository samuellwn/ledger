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
	"bufio"
	"encoding/csv"
	"io"
	"os"
	"regexp"

	"github.com/samuellwn/ledger"
	"github.com/samuellwn/ledger/parse"
)

// LoadLedgerFile loads a ledger file from the given path. On any error the message is logged to standard error and the
// program exits with code 1.
func LoadLedgerFile(f *os.File) *ledger.File {
	lf, err := parse.ParseLedger(parse.NewRawCharReader(bufio.NewReader(f), 1))
	HandleErr(err)
	return lf
}

// WriteLedgerFile writes out a ledger file to the given path. On any error the message is logged to standard error
// and the program exits with code 1.
func WriteLedgerFile(f *os.File, d *ledger.File) {
	_ = f.Truncate(0) // sometimes this gets called on os.Stdout
	HandleErr(d.Format(f))
}

// LoadMatchFile loads a csv match file and parses it into a list of Matchers. On any error the message is logged to
// standard error and the program exits with code 1.
func LoadMatchFile(mr *os.File) []ledger.Matcher {
	mrdr := csv.NewReader(mr)
	mrdr.FieldsPerRecord = 3
	mrdr.Comment = '#'

	matchers := []ledger.Matcher{}
	for {
		line, err := mrdr.Read()
		if err == io.EOF {
			break
		}
		HandleErr(err)

		reg := HandleErrV(regexp.Compile(line[0]))

		matchers = append(matchers, ledger.Matcher{
			R:       reg,
			Account: line[1],
			Payee:   line[2],
		})
	}
	return matchers
}
