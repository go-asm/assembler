// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

// Note: this program must be run with the GOROOT
// environment variable set to the root of this tree.
//   GOROOT=...
//   cd $GOROOT/src/github.com/go-asm/go/cmd/compile/ir
//   ../../../../../bin/go run -mod=mod mknode.go

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"io/ioutil"
	"log"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

var irPkg *types.Package
var buf bytes.Buffer

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes,
	}
	pkgs, err := packages.Load(cfg, "github.com/go-asm/go/cmd/compile/ir")
	if err != nil {
		log.Fatal(err)
	}
	irPkg = pkgs[0].Types

	fmt.Fprintln(&buf, "// Code generated by mknode.go. DO NOT EDIT.")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "package ir")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, `import "fmt"`)

	scope := irPkg.Scope()
	for _, name := range scope.Names() {
		if strings.HasPrefix(name, "mini") {
			continue
		}

		obj, ok := scope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		typ := obj.Type().(*types.Named)
		if !implementsNode(types.NewPointer(typ)) {
			continue
		}

		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "func (n *%s) Format(s fmt.State, verb rune) { fmtNode(n, s, verb) }\n", name)

		switch name {
		case "Name", "Func":
			// Too specialized to automate.
			continue
		}

		forNodeFields(typ,
			"func (n *%[1]s) copy() Node { c := *n\n",
			"",
			"c.%[1]s = copy%[2]s(c.%[1]s)",
			"return &c }\n")

		forNodeFields(typ,
			"func (n *%[1]s) doChildren(do func(Node) bool) bool {\n",
			"if n.%[1]s != nil && do(n.%[1]s) { return true }",
			"if do%[2]s(n.%[1]s, do) { return true }",
			"return false }\n")

		forNodeFields(typ,
			"func (n *%[1]s) editChildren(edit func(Node) Node) {\n",
			"if n.%[1]s != nil { n.%[1]s = edit(n.%[1]s).(%[2]s) }",
			"edit%[2]s(n.%[1]s, edit)",
			"}\n")
	}

	makeHelpers()

	out, err := format.Source(buf.Bytes())
	if err != nil {
		// write out mangled source so we can see the bug.
		out = buf.Bytes()
	}

	err = ioutil.WriteFile("node_gen.go", out, 0666)
	if err != nil {
		log.Fatal(err)
	}
}

// needHelper maps needed slice helpers from their base name to their
// respective slice-element type.
var needHelper = map[string]string{}

func makeHelpers() {
	var names []string
	for name := range needHelper {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(&buf, sliceHelperTmpl, name, needHelper[name])
	}
}

const sliceHelperTmpl = `
func copy%[1]s(list []%[2]s) []%[2]s {
	if list == nil {
		return nil
	}
	c := make([]%[2]s, len(list))
	copy(c, list)
	return c
}
func do%[1]s(list []%[2]s, do func(Node) bool) bool {
	for _, x := range list {
		if x != nil && do(x) {
			return true
		}
	}
	return false
}
func edit%[1]s(list []%[2]s, edit func(Node) Node) {
	for i, x := range list {
		if x != nil {
			list[i] = edit(x).(%[2]s)
		}
	}
}
`

func forNodeFields(named *types.Named, prologue, singleTmpl, sliceTmpl, epilogue string) {
	fmt.Fprintf(&buf, prologue, named.Obj().Name())

	anyField(named.Underlying().(*types.Struct), func(f *types.Var) bool {
		if f.Embedded() {
			return false
		}
		name, typ := f.Name(), f.Type()

		slice, _ := typ.Underlying().(*types.Slice)
		if slice != nil {
			typ = slice.Elem()
		}

		tmpl, what := singleTmpl, types.TypeString(typ, types.RelativeTo(irPkg))
		if what == "go/constant.Value" {
			return false
		}
		if implementsNode(typ) {
			if slice != nil {
				helper := strings.TrimPrefix(what, "*") + "s"
				needHelper[helper] = what
				tmpl, what = sliceTmpl, helper
			}
		} else if what == "*Field" {
			// Special case for *Field.
			tmpl = sliceTmpl
			if slice != nil {
				what = "Fields"
			} else {
				what = "Field"
			}
		} else {
			return false
		}

		if tmpl == "" {
			return false
		}

		// Allow template to not use all arguments without
		// upsetting fmt.Printf.
		s := fmt.Sprintf(tmpl+"\x00 %[1]s %[2]s", name, what)
		fmt.Fprintln(&buf, s[:strings.LastIndex(s, "\x00")])
		return false
	})

	fmt.Fprintf(&buf, epilogue)
}

func implementsNode(typ types.Type) bool {
	if _, ok := typ.Underlying().(*types.Interface); ok {
		// TODO(mdempsky): Check the interface implements Node.
		// Worst case, node_gen.go will fail to compile if we're wrong.
		return true
	}

	if ptr, ok := typ.(*types.Pointer); ok {
		if str, ok := ptr.Elem().Underlying().(*types.Struct); ok {
			return anyField(str, func(f *types.Var) bool {
				return f.Embedded() && f.Name() == "miniNode"
			})
		}
	}

	return false
}

func anyField(typ *types.Struct, pred func(f *types.Var) bool) bool {
	for i, n := 0, typ.NumFields(); i < n; i++ {
		if value, ok := reflect.StructTag(typ.Tag(i)).Lookup("mknode"); ok {
			if value != "-" {
				panic(fmt.Sprintf("unexpected tag value: %q", value))
			}
			continue
		}

		f := typ.Field(i)
		if pred(f) {
			return true
		}
		if f.Embedded() {
			if typ, ok := f.Type().Underlying().(*types.Struct); ok {
				if anyField(typ, pred) {
					return true
				}
			}
		}
	}
	return false
}
