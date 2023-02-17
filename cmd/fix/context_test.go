// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fix

func init() {
	addTestCases(contextTests, ctxfix)
}

var contextTests = []testCase{
	{
		Name: "context.0",
		In: `package fix

import "golang.org/x/net/context"

var _ = "golang.org/x/net/context"
`,
		Out: `package fix

import "context"

var _ = "golang.org/x/net/context"
`,
	},
	{
		Name: "context.1",
		In: `package fix

import ctx "golang.org/x/net/context"

var _ = ctx.Background()
`,
		Out: `package fix

import ctx "context"

var _ = ctx.Background()
`,
	},
}
