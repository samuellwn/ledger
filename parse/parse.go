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

import "time"
import "strings"
import "github.com/milochristiansen/ledger"

/*

Each element is either an xact or a directive.

Each xact is either plain, periodic, or automated.

The entire file is made up of these four types of entry.

*/

// ParseLedger parses a ledger file from a string into a list of Transactions.
func ParseLedger(input string) ([]*ledger.Transaction, error) {
	return ParseLedgerRaw(NewCharReader(input, 1))
}

// ParseLedgerRaw parses a ledger file from a CharReader into a list of Transactions.
func ParseLedgerRaw(cr *CharReader) ([]*ledger.Transaction, error) {
	rtn := []*ledger.Transaction{}
	for !cr.EOF {
		// Eat any leading white space, also lines that are blank.
		cr.Eat(" \t")
		if cr.C == '\n' {
			cr.Next()
			continue
		}

		// Consume comments that are not part of the body of a transaction.
		if cr.C == ';' {
			cr.EatUntil("\n")
			cr.Next()
			continue
		}

		// Anything that is left must be a transaction. We will treat transactions and directives
		// we don't support (yet) as an error.
		current := &ledger.Transaction{
			Tags:    map[string]bool{},
			KVPairs: map[string]string{},
			Line:    cr.L,
		}

		// Parse the leading dates(s)
		date, err := ParseDate(cr)
		if err != nil {
			return nil, err
		}
		current.Date = date
		if cr.C == '=' {
			cr.Next()
			date, err := ParseDate(cr)
			if err != nil {
				return nil, err
			}
			current.ClearDate = date
		}

		// Whitespace
		cr.Eat(" \t")
		if cr.EOF {
			return nil, ErrUnexpectedEnd(cr.L)
		}

		// The optional cleared indicator
		if cr.C == '*' {
			current.Status = ledger.StatusClear
			cr.Next()
		} else if cr.C == '!' {
			current.Status = ledger.StatusPending
			cr.Next()
		} else {
			current.Status = ledger.StatusUndefined
		}

		// Maybe more whitespace (only if there was a cleared indicator)
		cr.Eat(" \t")
		if cr.EOF {
			return nil, ErrUnexpectedEnd(cr.L)
		}

		// An optional "code"
		if cr.C == '(' {
			cr.Next()
			cr.Eat(" \t")
			desc, err := ReadUntilTrimmed(cr, ")\n")
			if err != nil {
				return nil, err
			}
			if cr.C == '\n' {
				return nil, ErrMalformed(cr.L)
			}
			current.Code = desc
			cr.Next()
		}

		// Even more ws
		cr.Eat(" \t")
		if cr.EOF {
			return nil, ErrUnexpectedEnd(cr.L)
		}

		// And, to cap the first line off, the description.
		desc, err := ReadUntilTrimmed(cr, "\n")
		if err != nil {
			return nil, err
		}
		current.Description = desc
		cr.Next()

		// Now parse the individual postings or comment lines.
		for cr.Match(" \t") {
			cr.Eat(" \t")
			if cr.EOF {
				return nil, ErrUnexpectedEnd(cr.L)
			}

			// Is a comment that is attached to the transaction
			if cr.C == ';' {
				cr.Next()

				cr.Eat(" \t")
				if cr.EOF {
					return nil, ErrUnexpectedEnd(cr.L)
				}

				// OK, we are going to read the line into a buffer, trying to look for patterns as we go.
				ln := []rune{}
				key := ""

				// 0: Starting.
				// 1: Found a colon first, read tags.
				// 2: Read at least one character, possible k/v
				// 3: Found a colon+space after state 2, finish reading k/v
				// 4: Not consistent with other states, just read as comment.
				state := 0
				for !cr.Match("\n") {
					// The first character is a colon, transition to state 1
					if state == 0 && cr.C == ':' {
						cr.Next()
						if cr.EOF {
							return nil, ErrUnexpectedEnd(cr.L)
						}
						state = 1
						continue
					}

					// The first character is anything other than a colon, transition to state 2
					if state == 0 {
						ln = append(ln, cr.C)
						cr.Next()
						if cr.EOF {
							return nil, ErrUnexpectedEnd(cr.L)
						}
						state = 2
						continue
					}

					// Found a leading colon, read tags.
					if state == 1 {
						if cr.C == ':' {
							tag := strings.TrimSpace(string(ln))
							if tag != "" {
								current.Tags[tag] = true
								ln = ln[:0]
							}
							cr.Next()
							cr.Eat(" \t")
							if cr.EOF {
								return nil, ErrUnexpectedEnd(cr.L)
							}
							continue
						}

						ln = append(ln, cr.C)
						cr.Next()
						if cr.EOF {
							return nil, ErrUnexpectedEnd(cr.L)
						}
						continue
					}

					// Possible k/v
					if state == 2 {
						if cr.C == ':' {
							if cr.NMatch(" \t") {
								// Dump ln and save aside as the key.
								key = string(ln)
								ln = ln[:0]

								// Get ready to read value.
								cr.Next()
								cr.Eat(" \t")
								if cr.EOF {
									return nil, ErrUnexpectedEnd(cr.L)
								}
								state = 3
								continue
							}

							// No space after colon.
							state = 4
							ln = append(ln, cr.C)
							cr.Next()
							if cr.EOF {
								return nil, ErrUnexpectedEnd(cr.L)
							}
							continue
						}

						if cr.Match(" \t") {
							// Key cannot have white space.
							state = 4
							ln = append(ln, cr.C)
							cr.Next()
							if cr.EOF {
								return nil, ErrUnexpectedEnd(cr.L)
							}
							continue
						}

						// Still reading possible key.
						ln = append(ln, cr.C)
						cr.Next()
						if cr.EOF {
							return nil, ErrUnexpectedEnd(cr.L)
						}
						continue
					}

					// Is a k/v, read value.
					if state == 3 {
						ln = append(ln, cr.C)
						cr.Next()
						if cr.EOF {
							return nil, ErrUnexpectedEnd(cr.L)
						}
						continue
					}

					// state == 4: Is not formatted, just read and dump to comments.
					ln = append(ln, cr.C)
					cr.Next()
					if cr.EOF {
						return nil, ErrUnexpectedEnd(cr.L)
					}
					continue
				}
				cr.Next()

				if state == 1 {
					for _, c := range ln {
						if c != ' ' && c != '\t' {
							// Error. Character on a tag line that is not part of tags.
							return nil, ErrMalformedTagLine(cr.L)
						}
					}

					continue
				}

				if state == 3 {
					current.KVPairs[key] = strings.TrimSpace(string(ln))
					continue
				}

				if state == 2 || state == 4 {
					current.Comments = append(current.Comments, strings.TrimSpace(string(ln)))
				}
				continue
			}

			// Otherwise must be a actual posting
			post := ledger.Posting{}

			// The optional cleared indicator, TBH I didn't even know this was a thing until I looked at the spec.
			if cr.C == '*' {
				post.Status = ledger.StatusClear
				cr.Next()
			} else if cr.C == '!' {
				post.Status = ledger.StatusPending
				cr.Next()
			} else {
				post.Status = ledger.StatusUndefined
			}

			cr.Eat(" \t")
			if cr.EOF {
				return nil, ErrUnexpectedEnd(cr.L)
			}

			// OK, now for the actual hard part.
			// Parsing the account name.
			// The spec doesn't seem to tell you the rules for account names, but they *can* include spaces.
			// I am going to allow spaces in account names, but only one in a row. Two or more spaces or a tab
			// ends the name.

			buf := []rune{}
			for {
				if cr.C == '\t' || cr.C == '\n' || (cr.C == ' ' && cr.NC == ' ') {
					break
				}

				buf = append(buf, cr.C)
				cr.Next()
				if cr.EOF {
					return nil, ErrUnexpectedEnd(cr.L)
				}
			}
			if len(buf) == 0 {
				return nil, ErrMalformed(cr.L)
			}
			post.Account = string(buf)

			cr.Eat(" \t")
			if cr.EOF {
				return nil, ErrUnexpectedEnd(cr.L)
			}

			// Read the amount. Currently only supporting USD with or without the leading $
			if cr.C == '$' {
				cr.Next()

				// Just in case...
				cr.Eat(" \t")
				if cr.EOF {
					return nil, ErrUnexpectedEnd(cr.L)
				}
			}

			neg := false
			if cr.C == '-' {
				cr.Next()
				neg = true
			}

			// Read the numeric part of the amount
			// This is probably shitty, and maybe wrong, but I hope not. I 1000% need to write tests for this.
			whole := int64(0)
			part := int64(0)
			cur := &whole
			null := true
			for cr.MatchNumeric() || cr.C == '.' || cr.C == ',' {
				if cr.C == '.' {
					if cur == &part || null == true {
						return nil, ErrBadAmount(cr.L)
					}
					cr.Next()
					cur = &part
					continue
				}
				if cr.C == ',' {
					cr.Next()
					continue
				}

				*cur = *cur*10 + int64(cr.C-'0')
				null = false
				cr.Next()
				if cr.EOF {
					return nil, ErrUnexpectedEnd(cr.L)
				}
			}
			if !null {
				whole = whole * 10000
				if part > 9999 {
					return nil, ErrBadAmount(cr.L)
				}
				switch {
				case part < 9:
					part = part * 1000
				case part < 99:
					part = part * 100
				case part < 9999:
					part = part * 10
				}
				post.Value = whole + part
				if neg {
					post.Value = -post.Value
				}
			}
			post.Null = null

			cr.Eat(" \t")
			if cr.EOF {
				return nil, ErrUnexpectedEnd(cr.L)
			}

			// Optional note
			if cr.C == ';' {
				cr.Next()
				line, err := ReadUntilTrimmed(cr, "\n")
				if err != nil {
					return nil, err
				}
				cr.Next()
				post.Note = line
				current.Postings = append(current.Postings, post)
				continue
			}

			cr.Eat(" \t")
			if cr.EOF {
				return nil, ErrUnexpectedEnd(cr.L)
			}

			if cr.C != '\n' {
				return nil, ErrMalformed(cr.L)
			}
			cr.Next()

			current.Postings = append(current.Postings, post)
		}

		rtn = append(rtn, current)
	}

	return rtn, nil
}

// ReadUntilTrimmed reads characters from the CharReader until one of the characters in `chars` is found.
// The result then has all the whitespace trimmed from the ends.
func ReadUntilTrimmed(cr *CharReader, chars string) (string, error) {
	ln := []rune{}
	ln = cr.ReadUntil(chars, ln)
	if cr.EOF {
		return "", ErrUnexpectedEnd(cr.L)
	}
	// Trim trailing ws
	for i := len(ln) - 1; i > 0; i-- {
		if ln[i] != ' ' && ln[i] != '\t' {
			break
		}
		ln = ln[:i]
	}
	// Trim leading ws
	for i := 0; i < len(ln); i++ {
		if ln[0] != ' ' && ln[0] != '\t' {
			break
		}
		ln = ln[1:]
	}
	return string(ln), nil
}

// ParseDate reads a date (in yyyy/mm/dd format) from the CharReader.
func ParseDate(cr *CharReader) (time.Time, error) {
	date := []rune{}
	ok := false
	var t time.Time

	ok, date = cr.ReadMatchLimit("0123456789", date, 4)
	if !ok {
		return t, ErrBadDate(cr.L)
	}
	if cr.EOF {
		return t, ErrUnexpectedEnd(cr.L)
	}

	if !cr.Match("/-.") {
		return t, ErrBadDate(cr.L)
	}
	date = append(date, '/')
	cr.Next()

	ok, date = cr.ReadMatchLimit("0123456789", date, 2)
	if !ok {
		return t, ErrBadDate(cr.L)
	}
	if cr.EOF {
		return t, ErrUnexpectedEnd(cr.L)
	}

	if !cr.Match("/-.") {
		return t, ErrBadDate(cr.L)
	}
	date = append(date, '/')
	cr.Next()

	ok, date = cr.ReadMatchLimit("0123456789", date, 2)
	if !ok {
		return t, ErrBadDate(cr.L)
	}
	if cr.EOF {
		return t, ErrUnexpectedEnd(cr.L)
	}

	return time.Parse("2006/01/02", string(date))
}
