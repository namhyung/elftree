/*
 * ELF tree - Tree viewer for ELF library dependency
 *
 * Copyright (C) 2017  Namhyung Kim <namhyung@gmail.com>
 *
 * Released under MIT license.
 */
package main

import (
	"bufio"
	"debug/elf"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type DepsNode struct {
	name   string
	parent *DepsNode
	child  []*DepsNode
	depth  int
}

type DynInfo struct {
	tag elf.DynTag
	val interface{}
}

type DepsInfo struct {
	path   string
	mach   elf.Machine
	bits   elf.Class
	endian binary.ByteOrder
	kind   elf.Type
	abi    elf.OSABI
	ver    uint8

	libs []string
	isym []elf.ImportedSymbol
	dsym []elf.Symbol
	syms []elf.Symbol
	prog []*elf.Prog
	sect []*elf.Section
	dyns []DynInfo
}

var (
	deps      map[string]DepsInfo
	deps_list []*DepsNode
	deps_root *DepsNode
	deflib    []string
	envlib    string
	conflib   []string
)

// command-line options
var (
	verbose   bool
	showPath  bool
	showTui   bool
	showStdio bool
)

func readLdSoConf(name string, libpath []string) []string {
	f, err := os.Open(name)
	if err != nil {
		return libpath
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		t := s.Text()

		if len(strings.TrimSpace(t)) == 0 {
			continue
		}
		if strings.HasPrefix(t, "#") {
			continue
		}

		if strings.HasPrefix(t, "include") {
			libs, err := filepath.Glob(t[8:])
			if err != nil {
				continue
			}
			for _, l := range libs {
				libpath = readLdSoConf(l, libpath)
			}
		} else {
			libpath = append(libpath, t)
		}
	}
	return libpath
}

func init() {
	deps = make(map[string]DepsInfo)
	deflib = []string{"/lib/", "/usr/lib/", "/lib64", "/usr/lib64"}
	envlib = os.Getenv("LD_LIBRARY_PATH")
	conflib = readLdSoConf("/etc/ld.so.conf", conflib)

	flag.BoolVar(&verbose, "v", false, "Show binary info")
	flag.BoolVar(&showPath, "p", false, "Show library path")
	flag.BoolVar(&showTui, "tui", true, "Show it with TUI")
	flag.BoolVar(&showStdio, "stdio", false, "Show it on standard IO")
}

// search shared libraries as described in `man ld.so(8)`
func findLib(name string, parent *DepsNode) string {
	if strings.Contains(name, "/") {
		return name
	}

	// check DT_RPATH attribute
	if parent != nil {
		info := deps[parent.name]
		for _, dyn := range info.dyns {
			if dyn.tag != elf.DT_RPATH {
				continue
			}

			fullpath := path.Join(dyn.val.(string), name)
			if _, err := os.Stat(fullpath); err == nil {
				return fullpath
			}
		}
	}

	// check LD_LIBRARY_PATH environ
	for _, libpath := range strings.Split(envlib, ":") {
		fullpath := path.Join(libpath, name)
		if _, err := os.Stat(fullpath); err == nil {
			return fullpath
		}
	}

	// check DT_RUNPATH attribute
	if parent != nil {
		info := deps[parent.name]
		for _, dyn := range info.dyns {
			if dyn.tag != elf.DT_RUNPATH {
				continue
			}

			fullpath := path.Join(dyn.val.(string), name)
			if _, err := os.Stat(fullpath); err == nil {
				return fullpath
			}
		}
	}

	// check libraries in /etc/ld.so.conf
	for _, libpath := range conflib {
		fullpath := path.Join(libpath, name)
		if _, err := os.Stat(fullpath); err == nil {
			return fullpath
		}
	}

	// check default library directories
	for _, libpath := range deflib {
		fullpath := path.Join(libpath, name)
		if _, err := os.Stat(fullpath); err == nil {
			return fullpath
		}
	}
	return ""
}

func realPath(pathname string) string {
	if pathname == "" {
		return ""
	}

	relpath, _ := filepath.EvalSymlinks(pathname)
	abspath, _ := filepath.Abs(relpath)

	return abspath
}

func readElfString(strtab []byte, i uint64) string {
	var len uint64

	for len = 0; strtab[i+len] != '\x00'; len++ {
		continue
	}

	return string(strtab[i : i+len])
}

func readDynamic(f *elf.File, info *DepsInfo) int {
	var i, count uint

	dyn := f.Section(".dynamic")
	if dyn == nil {
		return -1
	}

	data, err := dyn.Data()
	if err != nil {
		return -1
	}
	str := f.Section(".dynstr")
	stab, err := str.Data()
	if err != nil {
		return -1
	}

	count = uint(dyn.Size / dyn.Entsize)
	for i = 0; i < count; i++ {
		var tag, val uint64

		if f.Class == elf.ELFCLASS64 {
			tag = f.ByteOrder.Uint64(data[(i*2+0)*8 : (i*2+1)*8])
			val = f.ByteOrder.Uint64(data[(i*2+1)*8 : (i*2+2)*8])
		} else {
			tag = uint64(f.ByteOrder.Uint32(data[(i*2+0)*4 : (i*2+1)*4]))
			val = uint64(f.ByteOrder.Uint32(data[(i*2+1)*4 : (i*2+2)*4]))
		}

		dtag := elf.DynTag(tag)
		switch dtag {
		case elf.DT_NULL:
			break
		case elf.DT_NEEDED:
			fallthrough
		case elf.DT_RPATH:
			fallthrough
		case elf.DT_RUNPATH:
			fallthrough
		case elf.DT_SONAME:
			sval := readElfString(stab, val)
			info.dyns = append(info.dyns, DynInfo{dtag, sval})
			break
		default:
			info.dyns = append(info.dyns, DynInfo{dtag, val})
			break
		}
	}
	return 0
}

func processDep(dep *DepsNode) {
	// skip duplicate libraries
	if _, ok := deps[dep.name]; ok {
		return
	}

	info := DepsInfo{path: realPath(findLib(dep.name, dep.parent))}

	if dep.parent == nil {
		info.path = realPath(flag.Args()[0])
	}

	f, err := elf.Open(info.path)
	if err != nil {
		fmt.Printf("%v: %s (%s)\n", err, info.path, dep.name)
		os.Exit(1)
	}
	defer f.Close()

	info.mach = f.Machine
	info.bits = f.Class
	info.kind = f.Type
	info.abi = f.OSABI
	info.ver = f.ABIVersion
	info.endian = f.ByteOrder

	info.prog = f.Progs
	info.sect = f.Sections

	if f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN {
		fmt.Printf("elftree: `%s` seems not to be a valid ELF executable\n", dep.name)
		os.Exit(1)
	}

	if readDynamic(f, &info) < 0 {
		fmt.Printf("elftree: `%s` seems to be statically linked\n", dep.name)
		os.Exit(1)
	}

	libs, err := f.ImportedLibraries()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	isym, err := f.ImportedSymbols()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dsym, err := f.DynamicSymbols()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	syms, err := f.Symbols()
	if err == nil {
		info.syms = syms
	}

	info.libs = libs
	info.dsym = dsym
	info.isym = isym

	var L []*DepsNode
	for _, soname := range libs {
		N := new(DepsNode)
		N.name = soname
		N.parent = dep
		N.depth = dep.depth + 1

		L = append(L, N)
		dep.child = append(dep.child, N)
	}

	deps_list = append(L, deps_list...)
	deps[dep.name] = info
}

func printDepTree(n *DepsNode, f *elf.File) {
	for i := 0; i < n.depth; i++ {
		fmt.Printf("   ")
	}

	if showPath {
		fmt.Printf("%s  => %s\n", n.name, deps[n.name].path)
	} else {
		fmt.Println(n.name)
	}

	for _, v := range n.child {
		printDepTree(v, f)
	}

	if verbose && n.parent == nil {
		showDetails(f, deps[n.name].path)
	}
}

func showDetails(f *elf.File, pathname string) {
	s := f.Section(".interp")
	if s == nil {
		fmt.Printf("static linked executable: %s\n", pathname)
		os.Exit(1)
	}
	interp, err := s.Data()
	if err != nil {
		fmt.Printf("%v: %s\n", err, pathname)
		os.Exit(1)
	}

	di_deps, err := f.ImportedLibraries()
	if err != nil {
		fmt.Printf("imported libraries: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("%s: %s\n", path.Base(pathname), realPath(pathname))
	fmt.Printf("  type:                     %s  (%s / %s / %s)\n",
		f.Type, f.Machine, f.Class, f.ByteOrder)
	fmt.Printf("  interpreter:              %s\n", string(interp))
	fmt.Printf("  total dependency:         %d\n", len(deps)-1) // exclude itself
	fmt.Printf("  direct dependency:        %d\n", len(di_deps))
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: elftree [<options>] <executable>")
		os.Exit(1)
	}

	pathname := args[0]
	f, err := elf.Open(pathname)
	if err != nil {
		if strings.HasPrefix(err.Error(), "bad magic number") {
			fmt.Printf("elftree: `%s` is not an ELF file\n", pathname)
		} else {
			fmt.Printf("elftree: %v: %s\n", err, pathname)
		}
		os.Exit(1)
	}
	defer f.Close()

	deps_root = new(DepsNode)
	deps_root.name = path.Base(pathname)

	deps_list = append(deps_list, deps_root)
	for len(deps_list) > 0 {
		// pop first element
		dep := deps_list[0]
		deps_list = deps_list[1:]

		processDep(dep)
	}

	if showStdio {
		showTui = false
	}

	if showTui {
		ShowWithTUI(deps_root)
	} else {
		printDepTree(deps_root, f)
	}
}
