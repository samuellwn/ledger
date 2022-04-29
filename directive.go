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
	"bytes"

	"github.com/samuellwn/ledger/parse/lex"
	"golang.org/x/exp/slices"
)

// Directive is a simple type to represent a partially parsed, but not validated, command directive.
type Directive struct {
	Type        string       // The keyword that starts the directive.
	Argument    string       // Any remaining content that was on the first line of the directive.
	Lines       []string     // Subsequent indented lines. Stored here unparsed.
	FoundBefore int          // The transaction index this directive precedes.
	Location    lex.Location // Line number this directive begins at.
}

func (d *Directive) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(d.Type)
	buf.WriteRune(' ')
	buf.WriteString(d.Argument)
	buf.WriteRune('\n')

	for _, line := range d.Lines {
		buf.WriteRune('\t')
		buf.WriteString(line)
		buf.WriteRune('\n')
	}

	return buf.String()
}

// Compare two directives to see if they are identical.
func (d *Directive) Compare(d2 Directive) bool {
	ok := d.Type == d2.Type && d.Argument == d2.Argument && len(d.Lines) == len(d2.Lines)
	if !ok {
		return false
	}
	for i := 0; i < len(d.Lines); i++ {
		if d.Lines[i] != d2.Lines[i] {
			return false
		}
	}
	return true
}

// CleanCopy takes a perfect copy of this directive. Any edits to the returned Directive
// will not modify this method's receiver.
func (d *Directive) CleanCopy() *Directive {
	nd := *d
	nd.Lines = slices.Clone(d.Lines)
	return &nd
}
