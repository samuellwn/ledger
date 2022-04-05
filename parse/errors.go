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

import "fmt"

type ErrBadDate int

func (err ErrBadDate) Error() string {
	return fmt.Sprintf("Malformed transaction date on line: %v", int(err))
}

type ErrBadAmount int

func (err ErrBadAmount) Error() string {
	return fmt.Sprintf("Amount value out of range on line: %v", int(err))
}

type ErrUnexpectedEnd int

func (err ErrUnexpectedEnd) Error() string {
	return fmt.Sprintf("Unexpected end of input on line: %v", int(err))
}

type ErrMalformed int

func (err ErrMalformed) Error() string {
	return fmt.Sprintf("Malformed transaction on line: %v", int(err))
}

type ErrMalformedTagLine int

func (err ErrMalformedTagLine) Error() string {
	return fmt.Sprintf("Malformed tags in transaction on line: %v", int(err))
}
