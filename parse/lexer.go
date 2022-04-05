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

package parse

/*
This is a greatly simplified and stripped down version of the core Lexer code that many of my parsers used for years.
*/

import "strings"
import "unicode"

// CharReader is a simple way to read from a string character by character, with line info and lookahead.
type CharReader struct {
	source *strings.Reader

	// The current character
	L   int  // Line
	C   rune // Character
	EOF bool // true if current C and L are invalid, at end of input

	// The lookahead (next) character
	NL   int
	NC   rune
	NEOF bool // true if current NC and NL are invalid, will be at end of input with next advance
}

// Returns a new CharReader with the input preadvanced so that all fields are valid.
func NewCharReader(source string, line int) *CharReader {
	cr := new(CharReader)

	cr.source = strings.NewReader(source)

	cr.L = line
	cr.NL = line

	// prime the pump
	cr.Next()
	cr.Next()

	return cr
}

// Match returns true if C matches one of the chars in the string.
func (cr *CharReader) Match(chars string) bool {
	if cr.EOF {
		return false
	}

	for _, char := range chars {
		if cr.C == char {
			return true
		}
	}
	return false
}

// NMatch returns true if NC matches one of the chars in the string.
func (cr *CharReader) NMatch(chars string) bool {
	if cr.NEOF {
		return false
	}

	for _, char := range chars {
		if cr.NC == char {
			return true
		}
	}
	return false
}

// MatchAlpha returns true if C is an underscore or unicode letter
func (cr *CharReader) MatchAlpha() bool {
	if cr.EOF {
		return false
	}

	return cr.C == '_' || unicode.IsLetter(cr.C)
}

// MatchNumeric returns true if C is a decimal digit.
func (cr *CharReader) MatchNumeric() bool {
	if cr.EOF {
		return false
	}
	// It would be WAY too complicated to support non-arabic numerals.
	return cr.C >= '0' && cr.C <= '9'
}

// Next advances the reader one character position.
// C, L, and EOF gain the previous values of NC, NL, and NEOF, additionally a newly read character becomes NC.
// All carriage returns are simply ignored, and newlines will advance the current value of NL.
// If the end of input is reached NEOF is set to true.
func (cr *CharReader) Next() {
	if cr.EOF {
		return
	}
	if cr.NEOF {
		cr.EOF = true
		return
	}

	var err error

	cr.C = cr.NC
	cr.L = cr.NL

again:
	cr.NC, _, err = cr.source.ReadRune() // err should only ever be io.EOF
	if err != nil {
		cr.NEOF = true
		return
	}

	// We simply strip carriage returns.
	if cr.NC == '\r' {
		// This isn't a loop because this is an exception and it looks friggin weird to wrap
		// the whole thing in a loop for what amounts to an error case.
		goto again
	}

	if cr.NC == '\n' {
		cr.NL++
		return
	}
}

// Eat the given characters until something else is found or EOF.
func (cr *CharReader) Eat(chars string) {
	for cr.Match(chars) {
		cr.Next()
		if cr.EOF {
			return
		}
	}
}

// Eat all characters until one of the given chars are found or EOF.
func (cr *CharReader) EatUntil(chars string) {
	for !cr.Match(chars) {
		cr.Next()
		if cr.EOF {
			return
		}
	}
}

// ReadMatch reads all matching characters into a buffer until a nonmatching character is found or EOF.
func (cr *CharReader) ReadMatch(chars string, buf []rune) []rune {
	for cr.Match(chars) {
		buf = append(buf, cr.C)
		cr.Next()
		if cr.EOF {
			return buf
		}
	}
	return buf
}

// ReadMatchLimit reads all matching characters into a buffer until a nonmatching character is found,
// the limit is reached, or EOF. Returns true if the read stopped due to the limit.
func (cr *CharReader) ReadMatchLimit(chars string, buf []rune, limit int) (bool, []rune) {
	i := 0
	for cr.Match(chars) && i < limit {
		buf = append(buf, cr.C)
		cr.Next()
		if cr.EOF {
			return false, buf
		}
		i++
	}

	return i == limit, buf
}

// ReadUntil reads all characters into a buffer until a matching character is found or EOF.
func (cr *CharReader) ReadUntil(chars string, buf []rune) []rune {
	for !cr.Match(chars) {
		buf = append(buf, cr.C)
		cr.Next()
		if cr.EOF {
			return buf
		}
	}
	return buf
}
