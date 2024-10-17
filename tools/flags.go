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
	"flag"
	"fmt"
	"os"
)

const (
	FlagDestFile    = 1 << iota // The output ledger file
	FlagMasterFile              // The master ledger file
	FlagSourceFile              // The source data file, both for merging and import
	FlagMatchFile               // Match file (csv account match data)
	FlagAccountName             // Account name
	FlagID                      // Transaction ID
	FlagRID                     // Transaction revision ID
)

// FlagSet is used to store the results from the common flags. Not all of these values will be valid, even if
// their flag is in the set.
type FlagSet struct {
	DestFile    *os.File
	MasterFile  *os.File
	SourceFile  *os.File
	MatchFile   *os.File
	AccountName string
	ID          string
	RID         string

	Flags *flag.FlagSet
}

// CommonFlagSet returns a flagset filled out with your choice of several common flags.
func CommonFlagSet(flags int, usage string) *FlagSet {
	fs := &FlagSet{
		DestFile:   os.Stdout,
		SourceFile: os.Stdin,
		Flags:      flag.NewFlagSet(os.Args[0], flag.ExitOnError),
	}

	if flags&FlagDestFile != 0 {
		fs.Flags.Func("dest", "The output file `path`.", func(s string) (err error) {
			if s != "-" {
				fs.DestFile, err = os.Create(s)
			}
			return
		})
	}

	if flags&FlagMasterFile != 0 {
		fs.Flags.Func("master", "The master ledger file `path`.", func(s string) (err error) {
			fs.MasterFile, err = os.OpenFile(s, os.O_RDWR|os.O_CREATE, 0666)
			return
		})
	}

	if flags&FlagSourceFile != 0 {
		fs.Flags.Func("source", "The data source file `path`.", func(s string) (err error) {
			if s != "-" {
				fs.SourceFile, err = os.Open(s)
			}
			return
		})
	}

	if flags&FlagMatchFile != 0 {
		fs.Flags.Func("source", "Path to the match information `csv` file.", func(s string) (err error) {
			if s != "-" {
				fs.MatchFile, err = os.Open(s)
			} else {
				fs.MatchFile = os.Stdin
			}
			return
		})
	}

	if flags&FlagAccountName != 0 {
		fs.Flags.StringVar(&fs.AccountName, "account", "Example:Account", "The `account` name.")
	}

	if flags&FlagID != 0 {
		fs.Flags.StringVar(&fs.ID, "id", "NIL", "A transaction `ID` used to specify the point in the file to act from.")
	}

	if flags&FlagRID != 0 {
		fs.Flags.StringVar(&fs.RID, "rid", "NIL", "A transaction revision `ID` used to specify the point in the file to act from.")
	}

	fs.Flags.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		fs.Flags.PrintDefaults()
	}

	return fs
}

func (fs *FlagSet) Parse() {
	fs.Flags.Parse(os.Args[1:])
}
