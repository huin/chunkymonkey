package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
)

func main() {
	flag.Parse()
	filenames := flag.Args()
	fset := token.NewFileSet()
	pkgMap, firstErr := parser.ParseFiles(fset, filenames, 0)
	if firstErr != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing: %v\n", firstErr)
	}

	v := NewNodeChecker(fset)
	v.InterfaceName = regexp.MustCompile("[Ii][A-Z][A-Za-z]+")
	v.InvalidFuncName = regexp.MustCompile("^Get.+") // can't do negative match in Go's regexp?

	for _, pkg := range pkgMap {
		ast.Walk(v, pkg)
	}
}

type NodeChecker struct {
	fset            *token.FileSet
	InterfaceName   *regexp.Regexp
	InvalidFuncName *regexp.Regexp
}

func NewNodeChecker(fset *token.FileSet) *NodeChecker {
	return &NodeChecker{
		fset: fset,
	}
}

func (v *NodeChecker) Visit(node ast.Node) (w ast.Visitor) {
	switch n := node.(type) {
	case *ast.TypeSpec:
		v.checkTypeName(n)
	case *ast.FuncDecl:
		if n.Recv != nil { // is a method.
			v.checkFunc(n)
		}
	}
	return v
}

// report displays a message about a particular position in the fileset.
func (v *NodeChecker) report(pos token.Pos, format string, args ...interface{}) {
	position := v.fset.Position(pos)
	allArgs := make([]interface{}, len(args)+3)

	allArgs[0] = position.Filename
	allArgs[1] = position.Line
	allArgs[2] = position.Column
	copy(allArgs[3:], args)

	fmt.Fprintf(
		os.Stderr,
		"%s:%d:%d: "+format,
		allArgs...)
}

func (v *NodeChecker) checkFunc(f *ast.FuncDecl) {
	if v.InvalidFuncName.MatchString(f.Name.String()) {
		v.report(f.Name.NamePos, "Bad name for method %q\n", f.Name)
	}
}

func (v *NodeChecker) checkTypeName(typeSpec *ast.TypeSpec) {
	name := typeSpec.Name.Name
	switch t := typeSpec.Type.(type) {
	case *ast.InterfaceType:
		if !v.InterfaceName.MatchString(name) {
			v.report(typeSpec.Name.NamePos, "Bad name for interface %q\n", name)
		}
	}
}
