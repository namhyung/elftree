/* ELF tree - Tree viewer for ELF library dependency */
package main

import (
	"debug/elf"
	"fmt"
	"os"
	"path"
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

func init() {
	deps = make(map[string]bool)
	deflib = []string{"/lib/", "/usr/lib/"}
	envlib = os.Getenv("LD_LIBRARY_PATH")
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: elftree <executable>")
		os.Exit(1)
	}

	pathname := os.Args[1]
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
}
