// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fix

import "go/ast"

func init() {
	addTestCases(importTests, nil)
}

var importTests = []testCase{
	{
		Name: "import.0",
		Fn:   addImportFn("os"),
		In: `package fix

import (
	"os"
)
`,
		Out: `package fix

import (
	"os"
)
`,
	},
	{
		Name: "import.1",
		Fn:   addImportFn("os"),
		In: `package fix
`,
		Out: `package fix

import "os"
`,
	},
	{
		Name: "import.2",
		Fn:   addImportFn("os"),
		In: `package fix

// Comment
import "C"
`,
		Out: `package fix

// Comment
import "C"
import "os"
`,
	},
	{
		Name: "import.3",
		Fn:   addImportFn("os"),
		In: `package fix

// Comment
import "C"

import (
	"io"
	"utf8"
)
`,
		Out: `package fix

// Comment
import "C"

import (
	"io"
	"os"
	"utf8"
)
`,
	},
	{
		Name: "import.4",
		Fn:   deleteImportFn("os"),
		In: `package fix

import (
	"os"
)
`,
		Out: `package fix
`,
	},
	{
		Name: "import.5",
		Fn:   deleteImportFn("os"),
		In: `package fix

// Comment
import "C"
import "os"
`,
		Out: `package fix

// Comment
import "C"
`,
	},
	{
		Name: "import.6",
		Fn:   deleteImportFn("os"),
		In: `package fix

// Comment
import "C"

import (
	"io"
	"os"
	"utf8"
)
`,
		Out: `package fix

// Comment
import "C"

import (
	"io"
	"utf8"
)
`,
	},
	{
		Name: "import.7",
		Fn:   deleteImportFn("io"),
		In: `package fix

import (
	"io"   // a
	"os"   // b
	"utf8" // c
)
`,
		Out: `package fix

import (
	// a
	"os"   // b
	"utf8" // c
)
`,
	},
	{
		Name: "import.8",
		Fn:   deleteImportFn("os"),
		In: `package fix

import (
	"io"   // a
	"os"   // b
	"utf8" // c
)
`,
		Out: `package fix

import (
	"io" // a
	// b
	"utf8" // c
)
`,
	},
	{
		Name: "import.9",
		Fn:   deleteImportFn("utf8"),
		In: `package fix

import (
	"io"   // a
	"os"   // b
	"utf8" // c
)
`,
		Out: `package fix

import (
	"io" // a
	"os" // b
	// c
)
`,
	},
	{
		Name: "import.10",
		Fn:   deleteImportFn("io"),
		In: `package fix

import (
	"io"
	"os"
	"utf8"
)
`,
		Out: `package fix

import (
	"os"
	"utf8"
)
`,
	},
	{
		Name: "import.11",
		Fn:   deleteImportFn("os"),
		In: `package fix

import (
	"io"
	"os"
	"utf8"
)
`,
		Out: `package fix

import (
	"io"
	"utf8"
)
`,
	},
	{
		Name: "import.12",
		Fn:   deleteImportFn("utf8"),
		In: `package fix

import (
	"io"
	"os"
	"utf8"
)
`,
		Out: `package fix

import (
	"io"
	"os"
)
`,
	},
	{
		Name: "import.13",
		Fn:   rewriteImportFn("utf8", "encoding/utf8"),
		In: `package fix

import (
	"io"
	"os"
	"utf8" // thanks ken
)
`,
		Out: `package fix

import (
	"encoding/utf8" // thanks ken
	"io"
	"os"
)
`,
	},
	{
		Name: "import.14",
		Fn:   rewriteImportFn("asn1", "encoding/asn1"),
		In: `package fix

import (
	"asn1"
	"crypto"
	"crypto/rsa"
	_ "crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"
)

var x = 1
`,
		Out: `package fix

import (
	"crypto"
	"crypto/rsa"
	_ "crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"time"
)

var x = 1
`,
	},
	{
		Name: "import.15",
		Fn:   rewriteImportFn("url", "net/url"),
		In: `package fix

import (
	"bufio"
	"net"
	"path"
	"url"
)

var x = 1 // comment on x, not on url
`,
		Out: `package fix

import (
	"bufio"
	"net"
	"net/url"
	"path"
)

var x = 1 // comment on x, not on url
`,
	},
	{
		Name: "import.16",
		Fn:   rewriteImportFn("http", "net/http", "template", "text/template"),
		In: `package fix

import (
	"flag"
	"http"
	"log"
	"template"
)

var addr = flag.String("addr", ":1718", "http service address") // Q=17, R=18
`,
		Out: `package fix

import (
	"flag"
	"log"
	"net/http"
	"text/template"
)

var addr = flag.String("addr", ":1718", "http service address") // Q=17, R=18
`,
	},
	{
		Name: "import.17",
		Fn:   addImportFn("x/y/z", "x/a/c"),
		In: `package fix

// Comment
import "C"

import (
	"a"
	"b"

	"x/w"

	"d/f"
)
`,
		Out: `package fix

// Comment
import "C"

import (
	"a"
	"b"

	"x/a/c"
	"x/w"
	"x/y/z"

	"d/f"
)
`,
	},
	{
		Name: "import.18",
		Fn:   addDelImportFn("e", "o"),
		In: `package fix

import (
	"f"
	"o"
	"z"
)
`,
		Out: `package fix

import (
	"e"
	"f"
	"z"
)
`,
	},
}

func addImportFn(path ...string) func(*ast.File) bool {
	return func(f *ast.File) bool {
		fixed := false
		for _, p := range path {
			if !imports(f, p) {
				addImport(f, p)
				fixed = true
			}
		}
		return fixed
	}
}

func deleteImportFn(path string) func(*ast.File) bool {
	return func(f *ast.File) bool {
		if imports(f, path) {
			deleteImport(f, path)
			return true
		}
		return false
	}
}

func addDelImportFn(p1 string, p2 string) func(*ast.File) bool {
	return func(f *ast.File) bool {
		fixed := false
		if !imports(f, p1) {
			addImport(f, p1)
			fixed = true
		}
		if imports(f, p2) {
			deleteImport(f, p2)
			fixed = true
		}
		return fixed
	}
}

func rewriteImportFn(oldnew ...string) func(*ast.File) bool {
	return func(f *ast.File) bool {
		fixed := false
		for i := 0; i < len(oldnew); i += 2 {
			if imports(f, oldnew[i]) {
				rewriteImport(f, oldnew[i], oldnew[i+1])
				fixed = true
			}
		}
		return fixed
	}
}
