// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkginit

import (
	"github.com/go-asm/go/cmd/compile/base"
	"github.com/go-asm/go/cmd/compile/ir"
	"github.com/go-asm/go/cmd/compile/noder"
	"github.com/go-asm/go/cmd/compile/objw"
	"github.com/go-asm/go/cmd/compile/staticinit"
	"github.com/go-asm/go/cmd/compile/typecheck"
	"github.com/go-asm/go/cmd/compile/types"
	"github.com/go-asm/go/cmd/obj"
	"github.com/go-asm/go/cmd/objabi"
	"github.com/go-asm/go/cmd/src"
)

// MakeTask makes an initialization record for the package, if necessary.
// See runtime/proc.go:initTask for its layout.
// The 3 tasks for initialization are:
//  1. Initialize all of the packages the current package depends on.
//  2. Initialize all the variables that have initializers.
//  3. Run any init functions.
func MakeTask() {
	var deps []*obj.LSym // initTask records for packages the current package depends on
	var fns []*obj.LSym  // functions to call for package initialization

	// Find imported packages with init tasks.
	for _, pkg := range typecheck.Target.Imports {
		n, ok := pkg.Lookup(".inittask").Def.(*ir.Name)
		if !ok {
			continue
		}
		if n.Op() != ir.ONAME || n.Class != ir.PEXTERN {
			base.Fatalf("bad inittask: %v", n)
		}
		deps = append(deps, n.Linksym())
	}
	if base.Flag.ASan {
		// Make an initialization function to call runtime.asanregisterglobals to register an
		// array of instrumented global variables when -asan is enabled. An instrumented global
		// variable is described by a structure.
		// See the _asan_global structure declared in src/runtime/asan/asan.go.
		//
		// func init {
		// 		var globals []_asan_global {...}
		// 		asanregisterglobals(&globals[0], len(globals))
		// }
		for _, n := range typecheck.Target.Externs {
			if canInstrumentGlobal(n) {
				name := n.Sym().Name
				InstrumentGlobalsMap[name] = n
				InstrumentGlobalsSlice = append(InstrumentGlobalsSlice, n)
			}
		}
		ni := len(InstrumentGlobalsMap)
		if ni != 0 {
			// Make an init._ function.
			pos := base.AutogeneratedPos
			base.Pos = pos

			sym := noder.Renameinit()
			fnInit := ir.NewFunc(pos, pos, sym, types.NewSignature(nil, nil, nil))
			typecheck.DeclFunc(fnInit)

			// Get an array of instrumented global variables.
			globals := instrumentGlobals(fnInit)

			// Call runtime.asanregisterglobals function to poison redzones.
			// runtime.asanregisterglobals(unsafe.Pointer(&globals[0]), ni)
			asancall := ir.NewCallExpr(base.Pos, ir.OCALL, typecheck.LookupRuntime("asanregisterglobals"), nil)
			asancall.Args.Append(typecheck.ConvNop(typecheck.NodAddr(
				ir.NewIndexExpr(base.Pos, globals, ir.NewInt(base.Pos, 0))), types.Types[types.TUNSAFEPTR]))
			asancall.Args.Append(typecheck.DefaultLit(ir.NewInt(base.Pos, int64(ni)), types.Types[types.TUINTPTR]))

			fnInit.Body.Append(asancall)
			typecheck.FinishFuncBody()
			ir.CurFunc = fnInit
			typecheck.Stmts(fnInit.Body)
			ir.CurFunc = nil

			typecheck.Target.Inits = append(typecheck.Target.Inits, fnInit)
		}
	}

	// Record user init functions.
	for _, fn := range typecheck.Target.Inits {
		if fn.Sym().Name == "init" {
			// Synthetic init function for initialization of package-scope
			// variables. We can use staticinit to optimize away static
			// assignments.
			s := staticinit.Schedule{
				Plans: make(map[ir.Node]*staticinit.Plan),
				Temps: make(map[ir.Node]*ir.Name),
			}
			for _, n := range fn.Body {
				s.StaticInit(n)
			}
			fn.Body = s.Out
			ir.WithFunc(fn, func() {
				typecheck.Stmts(fn.Body)
			})

			if len(fn.Body) == 0 {
				fn.Body = []ir.Node{ir.NewBlockStmt(src.NoXPos, nil)}
			}
		}

		// Skip init functions with empty bodies.
		if len(fn.Body) == 1 {
			if stmt := fn.Body[0]; stmt.Op() == ir.OBLOCK && len(stmt.(*ir.BlockStmt).List) == 0 {
				continue
			}
		}
		fns = append(fns, fn.Nname.Linksym())
	}

	if len(deps) == 0 && len(fns) == 0 && types.LocalPkg.Path != "main" && types.LocalPkg.Path != "runtime" {
		return // nothing to initialize
	}

	// Make an .inittask structure.
	sym := typecheck.Lookup(".inittask")
	task := ir.NewNameAt(base.Pos, sym, types.Types[types.TUINT8]) // fake type
	task.Class = ir.PEXTERN
	sym.Def = task
	lsym := task.Linksym()
	ot := 0
	ot = objw.Uint32(lsym, ot, 0) // state: not initialized yet
	ot = objw.Uint32(lsym, ot, uint32(len(fns)))
	for _, f := range fns {
		ot = objw.SymPtr(lsym, ot, f, 0)
	}

	// Add relocations which tell the linker all of the packages
	// that this package depends on (and thus, all of the packages
	// that need to be initialized before this one).
	for _, d := range deps {
		r := obj.Addrel(lsym)
		r.Type = objabi.R_INITORDER
		r.Sym = d
	}
	// An initTask has pointers, but none into the Go heap.
	// It's not quite read only, the state field must be modifiable.
	objw.Global(lsym, int32(ot), obj.NOPTR)
}
