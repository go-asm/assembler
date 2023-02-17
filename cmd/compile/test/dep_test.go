// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"strings"
	"testing"

	"github.com/go-asm/go/testenv"
)

func TestDeps(t *testing.T) {
	out, err := testenv.Command(t, testenv.GoToolPath(t), "list", "-f", "{{.Deps}}", "github.com/go-asm/go/cmd/compile/gc").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, dep := range strings.Fields(strings.Trim(string(out), "[]")) {
		switch dep {
		case "go/build", "go/scanner":
			// github.com/go-asm/go/cmd/compile/importer introduces a dependency
			// on go/build and go/token; github.com/go-asm/go/cmd/compile/ uses
			// go/constant which uses go/token in its API. Once we
			// got rid of those dependencies, enable this check again.
			// TODO(gri) fix this
			// t.Errorf("undesired dependency on %q", dep)
		}
	}
}
