// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nm

import (
	"testing"

	"github.com/go-asm/go/testenv"
)

func TestInternalLinkerCgoExec(t *testing.T) {
	testenv.MustHaveCGO(t)
	testenv.MustInternalLink(t, true)
	testGoExec(t, true, false)
}

func TestExternalLinkerCgoExec(t *testing.T) {
	testenv.MustHaveCGO(t)
	testGoExec(t, true, true)
}

func TestCgoLib(t *testing.T) {
	testenv.MustHaveCGO(t)
	testGoLib(t, true)
}
