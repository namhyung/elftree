/*
 * ELF tree - Tree viewer for ELF library dependency
 *
 * Copyright (C) 2017-2018  Namhyung Kim <namhyung@gmail.com>
 *
 * Released under MIT license.
 */
package main

import (
	"debug/elf"
	"fmt"
	str "strings"
)

const (
	GNU_EH_FRAME = elf.PT_LOOS + 74769744
	GNU_STACK    = elf.PT_LOOS + 74769745
	GNU_RELRO    = elf.PT_LOOS + 74769746
)

func progHdrString(phdr *elf.Prog) string {
	var typeStr string
	var flagStr string

	switch phdr.Type {
	case GNU_EH_FRAME:
		typeStr = "GNU_EH_FRAME"
	case GNU_STACK:
		typeStr = "GNU_STACK"
	case GNU_RELRO:
		typeStr = "GNU_RELRO"
	default:
		typeStr = phdr.Type.String()[3:]
	}

	switch phdr.Flags {
	case elf.PF_X:
		flagStr = "__X"
	case elf.PF_W:
		flagStr = "_W_"
	case elf.PF_R:
		flagStr = "R__"
	case elf.PF_R | elf.PF_W:
		flagStr = "RW_"
	case elf.PF_R | elf.PF_X:
		flagStr = "R_X"
	case elf.PF_R | elf.PF_W | elf.PF_X:
		flagStr = "RWX"
	default:
		flagStr = "???"
	}

	return fmt.Sprintf("%-16s  %s    %#8x  %#8x  %#8x", typeStr, flagStr, phdr.Vaddr, phdr.Memsz, phdr.Align)
}

const (
	DT_GNU_HASH   = elf.DT_HIOS + 3829
	DT_RELACOUNT  = elf.DT_VERSYM + 9
	DT_RELCOUNT   = elf.DT_VERSYM + 10
	DT_FLAGS_1    = elf.DT_VERSYM + 11
	DT_VERDEF     = elf.DT_VERSYM + 12
	DT_VERDEFNUM  = elf.DT_VERSYM + 13
	DT_VERNEED    = elf.DT_VERSYM + 14
	DT_VERNEEDNUM = elf.DT_VERSYM + 15
)

// convert DT_FLAGS
func strFlags(val uint64) string {
	var ret []string

	if (val & 0x1) != 0 {
		ret = append(ret, "ORIGIN")
	}
	if (val & 0x2) != 0 {
		ret = append(ret, "SYMBOLIC")
	}
	if (val & 0x4) != 0 {
		ret = append(ret, "TEXTREL")
	}
	if (val & 0x8) != 0 {
		ret = append(ret, "BIND_NOW")
	}
	if (val & 0x10) != 0 {
		ret = append(ret, "STATIC_TLS")
	}

	return str.Join(ret, "|")
}

// convert DT_FLAGS_1
func strFlags1(val uint64) string {
	var ret []string

	if (val & 0x1) != 0 {
		ret = append(ret, "NOW")
	}
	if (val & 0x2) != 0 {
		ret = append(ret, "GLOBAL")
	}
	if (val & 0x4) != 0 {
		ret = append(ret, "GROUP")
	}
	if (val & 0x8) != 0 {
		ret = append(ret, "NODELETE")
	}
	if (val & 0x10) != 0 {
		ret = append(ret, "LOADFLTR")
	}
	if (val & 0x20) != 0 {
		ret = append(ret, "INITFIRST")
	}
	if (val & 0x40) != 0 {
		ret = append(ret, "NOOPEN")
	}
	if (val & 0x80) != 0 {
		ret = append(ret, "ORIGIN")
	}
	if (val & 0x100) != 0 {
		ret = append(ret, "DIRECT")
	}
	if (val & 0x200) != 0 {
		ret = append(ret, "TRANS")
	}
	if (val & 0x400) != 0 {
		ret = append(ret, "INTERPOSE")
	}
	if (val & 0x800) != 0 {
		ret = append(ret, "NODEFLIB")
	}
	if (val & 0x1000) != 0 {
		ret = append(ret, "NODUMP")
	}
	if (val & 0x2000) != 0 {
		ret = append(ret, "CONFLAT")
	}
	if (val & 0x4000) != 0 {
		ret = append(ret, "ENDFILTEE")
	}
	if (val & 0x8000) != 0 {
		ret = append(ret, "DISPRELDNE")
	}
	if (val & 0x10000) != 0 {
		ret = append(ret, "DISPRELPND")
	}
	if (val & 0x20000) != 0 {
		ret = append(ret, "NODIRECT")
	}
	if (val & 0x40000) != 0 {
		ret = append(ret, "IGNMULDEF")
	}
	if (val & 0x80000) != 0 {
		ret = append(ret, "NOKSYMS")
	}
	if (val & 0x100000) != 0 {
		ret = append(ret, "NOHDR")
	}
	if (val & 0x200000) != 0 {
		ret = append(ret, "EDITED")
	}
	if (val & 0x400000) != 0 {
		ret = append(ret, "NORELOC")
	}
	if (val & 0x800000) != 0 {
		ret = append(ret, "SYMINTPOSE")
	}
	if (val & 0x1000000) != 0 {
		ret = append(ret, "GLOBAUDIT")
	}
	if (val & 0x2000000) != 0 {
		ret = append(ret, "SINGLETON")
	}

	return str.Join(ret, "|")
}

func makeDynamicStrings(info *DepsInfo) []string {
	// dynamic attributes
	var dyns []string
	for _, v := range info.dyns {
		switch v.tag {
		case elf.DT_NEEDED:
			fallthrough
		case elf.DT_RPATH:
			fallthrough
		case elf.DT_RUNPATH:
			fallthrough
		case elf.DT_SONAME:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %s", v.tag, v.val.(string)))
		case DT_GNU_HASH:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %x", "DT_GNU_HASH", v.val))
		case DT_RELACOUNT:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %v", "DT_RELACOUNT", v.val))
		case DT_RELCOUNT:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %v", "DT_RELCOUNT", v.val))
		case elf.DT_FLAGS:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %s", "DT_FLAGS", strFlags(v.val.(uint64))))
		case DT_FLAGS_1:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %s", "DT_FLAGS_1", strFlags1(v.val.(uint64))))
		case DT_VERDEF:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %x", "DT_VERDEF", v.val))
		case DT_VERDEFNUM:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %v", "DT_VERDEFNUM", v.val))
		case DT_VERNEED:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %x", "DT_VERNEED", v.val))
		case DT_VERNEEDNUM:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %v", "DT_VERNEEDNUM", v.val))
		default:
			dyns = append(dyns, fmt.Sprintf("  %-16s  %x", v.tag, v.val))
		}
	}

	return dyns
}

func makeSymbolString(sym elf.Symbol) string {
	var t string
	switch elf.ST_TYPE(sym.Info) {
	case elf.STT_NOTYPE:
		t = "NON"
	case elf.STT_OBJECT:
		t = "OBJ"
	case elf.STT_FUNC:
		t = "FUN"
	case elf.STT_SECTION:
		t = "SEC"
	case elf.STT_FILE:
		t = "FIL"
	case elf.STT_COMMON:
		t = "COM"
	case elf.STT_TLS:
		t = "TLS"
	default:
		t = "XXX"
	}

	var b string
	switch elf.ST_BIND(sym.Info) {
	case elf.STB_LOCAL:
		b = "L"
	case elf.STB_GLOBAL:
		b = "G"
	case elf.STB_WEAK:
		b = "W"
	default:
		b = "X"
	}

	return fmt.Sprintf("  %8x %s %s %s", sym.Value, t, b, sym.Name)
}

func makeSectionString(idx int, sec *elf.Section) string {
	var flag []string

	val := sec.Flags
	if (val & 0x1) != 0 {
		flag = append(flag, "W") // write
	}
	if (val & 0x2) != 0 {
		flag = append(flag, "A") // alloc
	}
	if (val & 0x4) != 0 {
		flag = append(flag, "X") // execute
	}
	if (val & 0x10) != 0 {
		flag = append(flag, "M") // merge
	}
	if (val & 0x20) != 0 {
		flag = append(flag, "S") // string
	}
	if (val & 0x40) != 0 {
		flag = append(flag, "I") // info link
	}
	if (val & 0x80) != 0 {
		flag = append(flag, "L") // link order
	}
	if (val & 0x100) != 0 {
		flag = append(flag, "O") // OS non-conforming
	}
	if (val & 0x200) != 0 {
		flag = append(flag, "G") // group
	}
	if (val & 0x400) != 0 {
		flag = append(flag, "T") // TLS
	}
	if (val & 0x800) != 0 {
		flag = append(flag, "C") // compressed
	}
	f := str.Join(flag, "")

	t := sec.Type.String()
	return fmt.Sprintf("  [%2d] %-24s %-12s %8x %8x %4s",
		idx, sec.Name, t[4:len(t)], sec.Offset, sec.Size, f)
}
