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

type DepsInfo struct {
	name  string
	path  string
	depth int
}

var (
	deps      map[string]bool
	deps_list []DepsInfo
	deflib    []string
	envlib    string
)

// command-line options
var (
	verbose bool
)

func init() {
	deps = make(map[string]bool)
	deflib = []string{"/lib/", "/usr/lib/"}
	envlib = os.Getenv("LD_LIBRARY_PATH")

	flag.BoolVar(&verbose, "v", false, "Show binary info")
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

func processDep(dep DepsInfo) {
	for i := 0; i < dep.depth; i++ {
		fmt.Printf("   ")
	}
	fmt.Println(dep.name)

	// skip duplicate libraries
	if _, ok := deps[dep.name]; ok {
		return
	}
	deps[dep.name] = true

	f, err := elf.Open(dep.path)
	if err != nil {
		fmt.Printf("%v: %s\n", err, dep.path)
		os.Exit(1)
	}
	defer f.Close()

	libs, err := f.ImportedLibraries()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var L []DepsInfo
	for _, soname := range libs {
		L = append(L, DepsInfo{soname, findLib(soname), dep.depth + 1})
	}

	deps_list = append(L, deps_list...)
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

	relpath, _ := filepath.EvalSymlinks(pathname)
	abspath, _ := filepath.Abs(relpath)

	fmt.Println()
	fmt.Printf("%s: %s\n", path.Base(pathname), abspath)
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

	deps_list = append(deps_list, DepsInfo{path.Base(pathname), pathname, 0})
	for len(deps_list) > 0 {
		// pop first element
		dep := deps_list[0]
		deps_list = deps_list[1:]

		processDep(dep)
	}

	if verbose {
		showDetails(f, pathname)
	}
}
