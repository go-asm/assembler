// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mips64

import (
	"github.com/go-asm/go/buildcfg"
	"github.com/go-asm/go/cmd/compile/ssa"
	"github.com/go-asm/go/cmd/compile/ssagen"
	"github.com/go-asm/go/cmd/obj/mips"
)

func Init(arch *ssagen.ArchInfo) {
	arch.LinkArch = &mips.Linkmips64
	if buildcfg.GOARCH == "mips64le" {
		arch.LinkArch = &mips.Linkmips64le
	}
	arch.REGSP = mips.REGSP
	arch.MAXWIDTH = 1 << 50
	arch.SoftFloat = buildcfg.GOMIPS64 == "softfloat"
	arch.ZeroRange = zerorange
	arch.Ginsnop = ginsnop

	arch.SSAMarkMoves = func(s *ssagen.State, b *ssa.Block) {}
	arch.SSAGenValue = ssaGenValue
	arch.SSAGenBlock = ssaGenBlock
}
