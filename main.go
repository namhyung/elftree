/* ELF tree - Tree viewer for ELF library dependency */
package main

import (
	"debug/elf"
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

type DepsInfo struct {
	path string
}

var (
	deps      map[string]DepsInfo
	deps_list []*DepsNode
	deps_root *DepsNode
	deflib    []string
	envlib    string
)

// command-line options
var (
	verbose  bool
	showPath bool
)

func init() {
	deps = make(map[string]DepsInfo)
	deflib = []string{"/lib/", "/usr/lib/"}
	envlib = os.Getenv("LD_LIBRARY_PATH")

	flag.BoolVar(&verbose, "v", false, "Show binary info")
	flag.BoolVar(&showPath, "p", false, "Show library path")
}

func findLib(name string) string {
	if strings.Contains(name, "/") {
		return name
	}

	// check LD_LIBRARY_PATH environ
	for _, libpath := range strings.Split(envlib, ":") {
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
	relpath, _ := filepath.EvalSymlinks(pathname)
	abspath, _ := filepath.Abs(relpath)

	return abspath
}

func processDep(dep *DepsNode) {
	// skip duplicate libraries
	if _, ok := deps[dep.name]; ok {
		return
	}

	info := DepsInfo{realPath(findLib(dep.name))}

	if dep.parent == nil {
		info.path = realPath(flag.Args()[0])
	}

	f, err := elf.Open(info.path)
	if err != nil {
		fmt.Printf("%v: %s\n", err, info.path)
		os.Exit(1)
	}
	defer f.Close()

	libs, err := f.ImportedLibraries()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

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

func printDepTree(n *DepsNode) {
	for i := 0; i < n.depth; i++ {
		fmt.Printf("   ")
	}

	if showPath {
		fmt.Printf("%s  => %s\n", n.name, deps[n.name].path)
	} else {
		fmt.Println(n.name)
	}

	for _, v := range n.child {
		printDepTree(v)
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
		fmt.Printf("%v: %s\n", err, pathname)
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

	printDepTree(deps_root)

	if verbose {
		showDetails(f, pathname)
	}
}
