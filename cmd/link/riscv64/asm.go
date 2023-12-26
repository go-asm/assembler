// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package riscv64

import (
	"debug/elf"
	"fmt"
	"log"
	"sort"

	"github.com/go-asm/go/cmd/link/ld"
	"github.com/go-asm/go/cmd/link/loader"
	"github.com/go-asm/go/cmd/link/sym"
	"github.com/go-asm/go/cmd/obj/riscv"
	"github.com/go-asm/go/cmd/objabi"
	"github.com/go-asm/go/cmd/sys"
)

// fakeLabelName matches the RISCV_FAKE_LABEL_NAME from binutils.
const fakeLabelName = ".L0 "

func gentext(ctxt *ld.Link, ldr *loader.Loader) {
}

func genSymsLate(ctxt *ld.Link, ldr *loader.Loader) {
	if ctxt.LinkMode != ld.LinkExternal {
		return
	}

	// Generate a local text symbol for each relocation target, as the
	// R_RISCV_PCREL_LO12_* relocations generated by elfreloc1 need it.
	if ctxt.Textp == nil {
		log.Fatal("genSymsLate called before Textp has been assigned")
	}
	var hi20Syms []loader.Sym
	for _, s := range ctxt.Textp {
		relocs := ldr.Relocs(s)
		for ri := 0; ri < relocs.Count(); ri++ {
			r := relocs.At(ri)
			if r.Type() != objabi.R_RISCV_PCREL_ITYPE && r.Type() != objabi.R_RISCV_PCREL_STYPE &&
				r.Type() != objabi.R_RISCV_TLS_IE_ITYPE && r.Type() != objabi.R_RISCV_TLS_IE_STYPE {
				continue
			}
			if r.Off() == 0 && ldr.SymType(s) == sym.STEXT {
				// Use the symbol for the function instead of creating
				// an overlapping symbol.
				continue
			}

			// TODO(jsing): Consider generating ELF symbols without needing
			// loader symbols, in order to reduce memory consumption. This
			// would require changes to genelfsym so that it called
			// putelfsym and putelfsyment as appropriate.
			sb := ldr.MakeSymbolBuilder(fakeLabelName)
			sb.SetType(sym.STEXT)
			sb.SetValue(ldr.SymValue(s) + int64(r.Off()))
			sb.SetLocal(true)
			sb.SetReachable(true)
			sb.SetVisibilityHidden(true)
			sb.SetSect(ldr.SymSect(s))
			if outer := ldr.OuterSym(s); outer != 0 {
				ldr.AddInteriorSym(outer, sb.Sym())
			}
			hi20Syms = append(hi20Syms, sb.Sym())
		}
	}
	ctxt.Textp = append(ctxt.Textp, hi20Syms...)
	ldr.SortSyms(ctxt.Textp)
}

func findHI20Symbol(ctxt *ld.Link, ldr *loader.Loader, val int64) loader.Sym {
	idx := sort.Search(len(ctxt.Textp), func(i int) bool { return ldr.SymValue(ctxt.Textp[i]) >= val })
	if idx >= len(ctxt.Textp) {
		return 0
	}
	if s := ctxt.Textp[idx]; ldr.SymValue(s) == val && ldr.SymType(s) == sym.STEXT {
		return s
	}
	return 0
}

func elfreloc1(ctxt *ld.Link, out *ld.OutBuf, ldr *loader.Loader, s loader.Sym, r loader.ExtReloc, ri int, sectoff int64) bool {
	elfsym := ld.ElfSymForReloc(ctxt, r.Xsym)
	switch r.Type {
	case objabi.R_ADDR, objabi.R_DWARFSECREF:
		out.Write64(uint64(sectoff))
		switch r.Size {
		case 4:
			out.Write64(uint64(elf.R_RISCV_32) | uint64(elfsym)<<32)
		case 8:
			out.Write64(uint64(elf.R_RISCV_64) | uint64(elfsym)<<32)
		default:
			ld.Errorf(nil, "unknown size %d for %v relocation", r.Size, r.Type)
			return false
		}
		out.Write64(uint64(r.Xadd))

	case objabi.R_RISCV_CALL, objabi.R_RISCV_CALL_TRAMP:
		out.Write64(uint64(sectoff))
		out.Write64(uint64(elf.R_RISCV_JAL) | uint64(elfsym)<<32)
		out.Write64(uint64(r.Xadd))

	case objabi.R_RISCV_PCREL_ITYPE, objabi.R_RISCV_PCREL_STYPE, objabi.R_RISCV_TLS_IE_ITYPE, objabi.R_RISCV_TLS_IE_STYPE:
		// Find the text symbol for the AUIPC instruction targeted
		// by this relocation.
		relocs := ldr.Relocs(s)
		offset := int64(relocs.At(ri).Off())
		hi20Sym := findHI20Symbol(ctxt, ldr, ldr.SymValue(s)+offset)
		if hi20Sym == 0 {
			ld.Errorf(nil, "failed to find text symbol for HI20 relocation at %d (%x)", sectoff, ldr.SymValue(s)+offset)
			return false
		}
		hi20ElfSym := ld.ElfSymForReloc(ctxt, hi20Sym)

		// Emit two relocations - a R_RISCV_PCREL_HI20 relocation and a
		// corresponding R_RISCV_PCREL_LO12_I or R_RISCV_PCREL_LO12_S relocation.
		// Note that the LO12 relocation must point to a target that has a valid
		// HI20 PC-relative relocation text symbol, which in turn points to the
		// given symbol. For further details see the ELF specification for RISC-V:
		//
		//   https://github.com/riscv-non-isa/riscv-elf-psabi-doc/blob/master/riscv-elf.adoc#pc-relative-symbol-addresses
		//
		var hiRel, loRel elf.R_RISCV
		switch r.Type {
		case objabi.R_RISCV_PCREL_ITYPE:
			hiRel, loRel = elf.R_RISCV_PCREL_HI20, elf.R_RISCV_PCREL_LO12_I
		case objabi.R_RISCV_PCREL_STYPE:
			hiRel, loRel = elf.R_RISCV_PCREL_HI20, elf.R_RISCV_PCREL_LO12_S
		case objabi.R_RISCV_TLS_IE_ITYPE:
			hiRel, loRel = elf.R_RISCV_TLS_GOT_HI20, elf.R_RISCV_PCREL_LO12_I
		case objabi.R_RISCV_TLS_IE_STYPE:
			hiRel, loRel = elf.R_RISCV_TLS_GOT_HI20, elf.R_RISCV_PCREL_LO12_S
		}
		out.Write64(uint64(sectoff))
		out.Write64(uint64(hiRel) | uint64(elfsym)<<32)
		out.Write64(uint64(r.Xadd))
		out.Write64(uint64(sectoff + 4))
		out.Write64(uint64(loRel) | uint64(hi20ElfSym)<<32)
		out.Write64(uint64(0))

	default:
		return false
	}

	return true
}

func elfsetupplt(ctxt *ld.Link, plt, gotplt *loader.SymbolBuilder, dynamic loader.Sym) {
	log.Fatalf("elfsetupplt")
}

func machoreloc1(*sys.Arch, *ld.OutBuf, *loader.Loader, loader.Sym, loader.ExtReloc, int64) bool {
	log.Fatalf("machoreloc1 not implemented")
	return false
}

func archreloc(target *ld.Target, ldr *loader.Loader, syms *ld.ArchSyms, r loader.Reloc, s loader.Sym, val int64) (o int64, nExtReloc int, ok bool) {
	rs := r.Sym()
	pc := ldr.SymValue(s) + int64(r.Off())

	// If the call points to a trampoline, see if we can reach the symbol
	// directly. This situation can occur when the relocation symbol is
	// not assigned an address until after the trampolines are generated.
	if r.Type() == objabi.R_RISCV_CALL_TRAMP {
		relocs := ldr.Relocs(rs)
		if relocs.Count() != 1 {
			ldr.Errorf(s, "trampoline %v has %d relocations", ldr.SymName(rs), relocs.Count())
		}
		tr := relocs.At(0)
		if tr.Type() != objabi.R_RISCV_PCREL_ITYPE {
			ldr.Errorf(s, "trampoline %v has unexpected relocation %v", ldr.SymName(rs), tr.Type())
		}
		trs := tr.Sym()
		if ldr.SymValue(trs) != 0 && ldr.SymType(trs) != sym.SDYNIMPORT && ldr.SymType(trs) != sym.SUNDEFEXT {
			trsOff := ldr.SymValue(trs) + tr.Add() - pc
			if trsOff >= -(1<<20) && trsOff < (1<<20) {
				r.SetType(objabi.R_RISCV_CALL)
				r.SetSym(trs)
				r.SetAdd(tr.Add())
				rs = trs
			}
		}

	}

	if target.IsExternal() {
		switch r.Type() {
		case objabi.R_RISCV_CALL, objabi.R_RISCV_CALL_TRAMP:
			return val, 1, true

		case objabi.R_RISCV_PCREL_ITYPE, objabi.R_RISCV_PCREL_STYPE, objabi.R_RISCV_TLS_IE_ITYPE, objabi.R_RISCV_TLS_IE_STYPE:
			return val, 2, true
		}

		return val, 0, false
	}

	off := ldr.SymValue(rs) + r.Add() - pc

	switch r.Type() {
	case objabi.R_RISCV_CALL, objabi.R_RISCV_CALL_TRAMP:
		// Generate instruction immediates.
		imm, err := riscv.EncodeJImmediate(off)
		if err != nil {
			ldr.Errorf(s, "cannot encode R_RISCV_CALL relocation offset for %s: %v", ldr.SymName(rs), err)
		}
		immMask := int64(riscv.JTypeImmMask)

		val = (val &^ immMask) | int64(imm)

		return val, 0, true

	case objabi.R_RISCV_TLS_IE_ITYPE, objabi.R_RISCV_TLS_IE_STYPE:
		// TLS relocations are not currently handled for internal linking.
		// For now, TLS is only used when cgo is in use and cgo currently
		// requires external linking. However, we need to accept these
		// relocations so that code containing TLS variables will link,
		// even when they're not being used. For now, replace these
		// instructions with EBREAK to detect accidental use.
		const ebreakIns = 0x00100073
		return ebreakIns<<32 | ebreakIns, 0, true

	case objabi.R_RISCV_PCREL_ITYPE, objabi.R_RISCV_PCREL_STYPE:
		// Generate AUIPC and second instruction immediates.
		low, high, err := riscv.Split32BitImmediate(off)
		if err != nil {
			ldr.Errorf(s, "R_RISCV_PCREL_ relocation does not fit in 32 bits: %d", off)
		}

		auipcImm, err := riscv.EncodeUImmediate(high)
		if err != nil {
			ldr.Errorf(s, "cannot encode R_RISCV_PCREL_ AUIPC relocation offset for %s: %v", ldr.SymName(rs), err)
		}

		var secondImm, secondImmMask int64
		switch r.Type() {
		case objabi.R_RISCV_PCREL_ITYPE:
			secondImmMask = riscv.ITypeImmMask
			secondImm, err = riscv.EncodeIImmediate(low)
			if err != nil {
				ldr.Errorf(s, "cannot encode R_RISCV_PCREL_ITYPE I-type instruction relocation offset for %s: %v", ldr.SymName(rs), err)
			}
		case objabi.R_RISCV_PCREL_STYPE:
			secondImmMask = riscv.STypeImmMask
			secondImm, err = riscv.EncodeSImmediate(low)
			if err != nil {
				ldr.Errorf(s, "cannot encode R_RISCV_PCREL_STYPE S-type instruction relocation offset for %s: %v", ldr.SymName(rs), err)
			}
		default:
			panic(fmt.Sprintf("Unknown relocation type: %v", r.Type()))
		}

		auipc := int64(uint32(val))
		second := int64(uint32(val >> 32))

		auipc = (auipc &^ riscv.UTypeImmMask) | int64(uint32(auipcImm))
		second = (second &^ secondImmMask) | int64(uint32(secondImm))

		return second<<32 | auipc, 0, true
	}

	return val, 0, false
}

func archrelocvariant(*ld.Target, *loader.Loader, loader.Reloc, sym.RelocVariant, loader.Sym, int64, []byte) int64 {
	log.Fatalf("archrelocvariant")
	return -1
}

func extreloc(target *ld.Target, ldr *loader.Loader, r loader.Reloc, s loader.Sym) (loader.ExtReloc, bool) {
	switch r.Type() {
	case objabi.R_RISCV_CALL, objabi.R_RISCV_CALL_TRAMP:
		return ld.ExtrelocSimple(ldr, r), true

	case objabi.R_RISCV_PCREL_ITYPE, objabi.R_RISCV_PCREL_STYPE, objabi.R_RISCV_TLS_IE_ITYPE, objabi.R_RISCV_TLS_IE_STYPE:
		return ld.ExtrelocViaOuterSym(ldr, r, s), true
	}
	return loader.ExtReloc{}, false
}

func trampoline(ctxt *ld.Link, ldr *loader.Loader, ri int, rs, s loader.Sym) {
	relocs := ldr.Relocs(s)
	r := relocs.At(ri)

	switch r.Type() {
	case objabi.R_RISCV_CALL:
		pc := ldr.SymValue(s) + int64(r.Off())
		off := ldr.SymValue(rs) + r.Add() - pc

		// Relocation symbol has an address and is directly reachable,
		// therefore there is no need for a trampoline.
		if ldr.SymValue(rs) != 0 && off >= -(1<<20) && off < (1<<20) && (*ld.FlagDebugTramp <= 1 || ldr.SymPkg(s) == ldr.SymPkg(rs)) {
			break
		}

		// Relocation symbol is too far for a direct call or has not
		// yet been given an address. See if an existing trampoline is
		// reachable and if so, reuse it. Otherwise we need to create
		// a new trampoline.
		var tramp loader.Sym
		for i := 0; ; i++ {
			oName := ldr.SymName(rs)
			name := fmt.Sprintf("%s-tramp%d", oName, i)
			if r.Add() != 0 {
				name = fmt.Sprintf("%s%+x-tramp%d", oName, r.Add(), i)
			}
			tramp = ldr.LookupOrCreateSym(name, int(ldr.SymVersion(rs)))
			ldr.SetAttrReachable(tramp, true)
			if ldr.SymType(tramp) == sym.SDYNIMPORT {
				// Do not reuse trampoline defined in other module.
				continue
			}
			if oName == "runtime.deferreturn" {
				ldr.SetIsDeferReturnTramp(tramp, true)
			}
			if ldr.SymValue(tramp) == 0 {
				// Either trampoline does not exist or we found one
				// that does not have an address assigned and will be
				// laid down immediately after the current function.
				break
			}

			trampOff := ldr.SymValue(tramp) - (ldr.SymValue(s) + int64(r.Off()))
			if trampOff >= -(1<<20) && trampOff < (1<<20) {
				// An existing trampoline that is reachable.
				break
			}
		}
		if ldr.SymType(tramp) == 0 {
			trampb := ldr.MakeSymbolUpdater(tramp)
			ctxt.AddTramp(trampb)
			genCallTramp(ctxt.Arch, ctxt.LinkMode, ldr, trampb, rs, int64(r.Add()))
		}
		sb := ldr.MakeSymbolUpdater(s)
		if ldr.SymValue(rs) == 0 {
			// In this case the target symbol has not yet been assigned an
			// address, so we have to assume a trampoline is required. Mark
			// this as a call via a trampoline so that we can potentially
			// switch to a direct call during relocation.
			sb.SetRelocType(ri, objabi.R_RISCV_CALL_TRAMP)
		}
		relocs := sb.Relocs()
		r := relocs.At(ri)
		r.SetSym(tramp)
		r.SetAdd(0)

	default:
		ctxt.Errorf(s, "trampoline called with non-jump reloc: %d (%s)", r.Type(), sym.RelocName(ctxt.Arch, r.Type()))
	}
}

func genCallTramp(arch *sys.Arch, linkmode ld.LinkMode, ldr *loader.Loader, tramp *loader.SymbolBuilder, target loader.Sym, offset int64) {
	tramp.AddUint32(arch, 0x00000f97) // AUIPC	$0, X31
	tramp.AddUint32(arch, 0x000f8067) // JALR		X0, (X31)

	r, _ := tramp.AddRel(objabi.R_RISCV_PCREL_ITYPE)
	r.SetSiz(8)
	r.SetSym(target)
	r.SetAdd(offset)
}
