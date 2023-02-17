// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fix

func init() {
	addTestCases(gotypesTests, gotypes)
}

var gotypesTests = []testCase{
	{
		Name: "gotypes.0",
		In: `package fix

import "golang.org/x/tools/go/types"
import "golang.org/x/tools/go/exact"

var _ = exact.Kind

func f() {
	_ = exact.MakeBool(true)
}
`,
		Out: `package fix

import "go/types"
import "go/constant"

var _ = constant.Kind

func f() {
	_ = constant.MakeBool(true)
}
`,
	},
	{
		Name: "gotypes.1",
		In: `package fix

import "golang.org/x/tools/go/types"
import foo "golang.org/x/tools/go/exact"

var _ = foo.Kind

func f() {
	_ = foo.MakeBool(true)
}
`,
		Out: `package fix

import "go/types"
import "go/constant"

var _ = foo.Kind

func f() {
	_ = foo.MakeBool(true)
}
`,
	},
	{
		Name: "gotypes.0",
		In: `package fix

import "golang.org/x/tools/go/types"
import "golang.org/x/tools/go/exact"

var _ = exact.Kind
var constant = 23 // Use of new package name.

func f() {
	_ = exact.MakeBool(true)
}
`,
		Out: `package fix

import "go/types"
import "go/constant"

var _ = constant_.Kind
var constant = 23 // Use of new package name.

func f() {
	_ = constant_.MakeBool(true)
}
`,
	},
}
