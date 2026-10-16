// Command sqlcheck — власне правило статичного аналізу для RoomFlow:
// забороняє передавати в database/sql (Query, QueryRow, Exec, ...) SQL, зібраний через fmt.Sprintf
// або конкатенацію рядків. Саме такий дефект SonarQube знайшов у ЛР2 (ризик SQL-ін'єкції).
//
// Використання: go run ./tools/sqlcheck ./...
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var sqlMethods = map[string]bool{
	"Query": true, "QueryRow": true, "Exec": true,
	"QueryContext": true, "QueryRowContext": true, "ExecContext": true,
}

// Finding is a single rule violation.
type Finding struct {
	Pos     token.Position
	Message string
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = strings.TrimSuffix(os.Args[1], "/...")
	}
	findings, err := CheckDir(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, f := range findings {
		fmt.Printf("%s: %s\n", f.Pos, f.Message)
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

// CheckDir analyses all non-test Go files under root (testdata directories are skipped).
func CheckDir(root string) ([]Finding, error) {
	var all []Finding
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "testdata" || d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".")) && path != root {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		all = append(all, CheckFile(fset, file)...)
		return nil
	})
	return all, err
}

// CheckFile reports dynamic SQL passed to database/sql query methods.
func CheckFile(fset *token.FileSet, file *ast.File) []Finding {
	built := dynamicStrings(file)
	var out []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !sqlMethods[sel.Sel.Name] {
			return true
		}
		q := call.Args[0]
		if strings.HasSuffix(sel.Sel.Name, "Context") && len(call.Args) > 1 {
			q = call.Args[1]
		}
		if isDynamic(q) || (isIdent(q) && built[q.(*ast.Ident).Name]) {
			out = append(out, Finding{fset.Position(q.Pos()), "SQL query is built dynamically; use placeholders (?) and pass values as arguments"})
		}
		return true
	})
	return out
}

// dynamicStrings collects variables assigned from fmt.Sprintf or string concatenation.
func dynamicStrings(file *ast.File) map[string]bool {
	res := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range as.Rhs {
			if i >= len(as.Lhs) || !isDynamic(rhs) {
				continue
			}
			if id, ok := as.Lhs[i].(*ast.Ident); ok {
				res[id.Name] = true
			}
		}
		return true
	})
	return res
}

func isDynamic(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.CallExpr:
		if sel, ok := v.Fun.(*ast.SelectorExpr); ok {
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fmt" && strings.HasPrefix(sel.Sel.Name, "Sprint") {
				return true
			}
		}
	case *ast.BinaryExpr:
		return v.Op == token.ADD && !(isLiteral(v.X) && isLiteral(v.Y))
	}
	return false
}

func isLiteral(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		return true
	case *ast.BinaryExpr:
		return isLiteral(v.X) && isLiteral(v.Y)
	}
	return false
}

func isIdent(e ast.Expr) bool { _, ok := e.(*ast.Ident); return ok }
