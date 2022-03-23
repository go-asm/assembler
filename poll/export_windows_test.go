// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Export guts for testing on windows.
// Since testing imports os and os imports github.com/go-asm/assembler/poll,
// the github.com/go-asm/assembler/poll tests can not be in package poll.

package poll

var (
	LogInitFD = &logInitFD
)

func (fd *FD) IsPartOfNetpoll() bool {
	return fd.pd.runtimeCtx != 0
}
