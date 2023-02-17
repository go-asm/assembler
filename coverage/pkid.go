// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coverage

// Building the runtime package with coverage instrumentation enabled
// is tricky.  For all other packages, you can be guaranteed that
// the package init function is run before any functions are executed,
// but this invariant is not maintained for packages such as "runtime",
// "github.com/go-asm/go/cpu", etc. To handle this, hard-code the package ID for
// the set of packages whose functions may be running before the
// init function of the package is complete.
//
// Hardcoding is unfortunate because it means that the tool that does
// coverage instrumentation has to keep a list of runtime packages,
// meaning that if someone makes changes to the pkg "runtime"
// dependencies, unexpected behavior will result for coverage builds.
// The coverage runtime will detect and report the unexpected
// behavior; look for an error of this form:
//
//    internal error in coverage meta-data tracking:
//    list of hard-coded runtime package IDs needs revising.
//    registered list:
//    slot: 0 path='github.com/go-asm/go/cpu'  hard-coded id: 1
//    slot: 1 path='github.com/go-asm/go/goarch'  hard-coded id: 2
//    slot: 2 path='runtime/github.com/go-asm/go/atomic'  hard-coded id: 3
//    slot: 3 path='github.com/go-asm/go/goos'
//    slot: 4 path='runtime/github.com/go-asm/go/sys'  hard-coded id: 5
//    slot: 5 path='github.com/go-asm/go/abi'  hard-coded id: 4
//    slot: 6 path='runtime/github.com/go-asm/go/math'  hard-coded id: 6
//    slot: 7 path='github.com/go-asm/go/bytealg'  hard-coded id: 7
//    slot: 8 path='github.com/go-asm/go/goexperiment'
//    slot: 9 path='runtime/github.com/go-asm/go/syscall'  hard-coded id: 8
//    slot: 10 path='runtime'  hard-coded id: 9
//    fatal error: runtime.addCovMeta
//
// For the error above, the hard-coded list is missing "github.com/go-asm/go/goos"
// and "github.com/go-asm/go/goexperiment" ; the developer in question will need
// to copy the list above into "rtPkgs" below.
//
// Note: this strategy assumes that the list of dependencies of
// package runtime is fixed, and doesn't vary depending on OS/arch. If
// this were to be the case, we would need a table of some sort below
// as opposed to a fixed list.

var rtPkgs = [...]string{
	"github.com/go-asm/go/cpu",
	"github.com/go-asm/go/goarch",
	"runtime/github.com/go-asm/go/atomic",
	"github.com/go-asm/go/goos",
	"runtime/github.com/go-asm/go/sys",
	"github.com/go-asm/go/abi",
	"runtime/github.com/go-asm/go/math",
	"github.com/go-asm/go/bytealg",
	"github.com/go-asm/go/goexperiment",
	"runtime/github.com/go-asm/go/syscall",
	"runtime",
}

// Scoping note: the constants and apis in this file are internal
// only, not expected to ever be exposed outside of the runtime (unlike
// other coverage file formats and APIs, which will likely be shared
// at some point).

// NotHardCoded is a package pseudo-ID indicating that a given package
// is not part of the runtime and doesn't require a hard-coded ID.
const NotHardCoded = -1

// HardCodedPkgID returns the hard-coded ID for the specified package
// path, or -1 if we don't use a hard-coded ID. Hard-coded IDs start
// at -2 and decrease as we go down the list.
func HardCodedPkgID(pkgpath string) int {
	for k, p := range rtPkgs {
		if p == pkgpath {
			return (0 - k) - 2
		}
	}
	return NotHardCoded
}
