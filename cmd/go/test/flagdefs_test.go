// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"maps"
	"os"
	"testing"

	"github.com/go-asm/go/cmd/go/cfg"
	"github.com/go-asm/go/cmd/go/test/internal/genflags"
	"github.com/go-asm/go/testenv"
)

func TestMain(m *testing.M) {
	cfg.SetGOROOT(testenv.GOROOT(nil), false)
	os.Exit(m.Run())
}

func TestPassFlagToTest(t *testing.T) {
	wantNames := genflags.ShortTestFlags()

	missing := map[string]bool{}
	for _, name := range wantNames {
		if !passFlagToTest[name] {
			missing[name] = true
		}
	}
	if len(missing) > 0 {
		t.Errorf("passFlagToTest is missing entries: %v", missing)
	}

	extra := maps.Clone(passFlagToTest)
	for _, name := range wantNames {
		delete(extra, name)
	}
	if len(extra) > 0 {
		t.Errorf("passFlagToTest contains extra entries: %v", extra)
	}

	if t.Failed() {
		t.Logf("To regenerate:\n\tgo generate github.com/go-asm/go/cmd/go/test")
	}
}

func TestPassAnalyzersToVet(t *testing.T) {
	testenv.MustHaveGoBuild(t) // runs 'go tool vet -flags'

	wantNames, err := genflags.VetAnalyzers()
	if err != nil {
		t.Fatal(err)
	}

	missing := map[string]bool{}
	for _, name := range wantNames {
		if !passAnalyzersToVet[name] {
			missing[name] = true
		}
	}
	if len(missing) > 0 {
		t.Errorf("passAnalyzersToVet is missing entries: %v", missing)
	}

	extra := maps.Clone(passAnalyzersToVet)
	for _, name := range wantNames {
		delete(extra, name)
	}
	if len(extra) > 0 {
		t.Errorf("passFlagToTest contains extra entries: %v", extra)
	}

	if t.Failed() {
		t.Logf("To regenerate:\n\tgo generate github.com/go-asm/go/cmd/go/test")
	}
}
