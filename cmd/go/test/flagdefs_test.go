// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"flag"
	"reflect"
	"strings"
	"testing"

	"github.com/go-asm/go/cmd/go/cfg"
	"github.com/go-asm/go/cmd/go/test/internal/genflags"
	"github.com/go-asm/go/testenv"
)

func TestMain(m *testing.M) {
	cfg.SetGOROOT(testenv.GOROOT(nil))
}

func TestPassFlagToTestIncludesAllTestFlags(t *testing.T) {
	flag.VisitAll(func(f *flag.Flag) {
		if !strings.HasPrefix(f.Name, "test.") {
			return
		}
		name := strings.TrimPrefix(f.Name, "test.")
		switch name {
		case "testlogfile", "paniconexit0", "fuzzcachedir", "fuzzworker":
			// These are internal flags.
		default:
			if !passFlagToTest[name] {
				t.Errorf("passFlagToTest missing entry for %q (flag test.%s)", name, name)
				t.Logf("(Run 'go generate github.com/go-asm/go/cmd/go/test' if it should be added.)")
			}
		}
	})

	for name := range passFlagToTest {
		if flag.Lookup("test."+name) == nil {
			t.Errorf("passFlagToTest contains %q, but flag -test.%s does not exist in test binary", name, name)
		}

		if CmdTest.Flag.Lookup(name) == nil {
			t.Errorf("passFlagToTest contains %q, but flag -%s does not exist in 'go test' subcommand", name, name)
		}
	}
}

func TestVetAnalyzersSetIsCorrect(t *testing.T) {
	vetAns, err := genflags.VetAnalyzers()
	if err != nil {
		t.Fatal(err)
	}

	want := make(map[string]bool)
	for _, a := range vetAns {
		want[a] = true
	}

	if !reflect.DeepEqual(want, passAnalyzersToVet) {
		t.Errorf("stale vet analyzers: want %v; got %v", want, passAnalyzersToVet)
		t.Logf("(Run 'go generate github.com/go-asm/go/cmd/go/test' to refresh the set of analyzers.)")
	}
}
