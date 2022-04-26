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
	"fmt"
	"os"
)

// Why did I do this? Just because I could?

// HandleErrV takes a value+err and returns the value if and only if the error is nil. If the error is not nil,
// it is written to standard error and os.Exit(1) is called.
func HandleErrV[T any](t T, err error) T {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return t
}

// HandleErrS takes a condition+string and if the condition is true, the string is written to standard error
// and os.Exit(1) is called.
func HandleErrS(cond bool, err string) {
	if cond {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// HandleErr takes an error and if the error is not nil, it is written to standard error and os.Exit(1) is called.
func HandleErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
