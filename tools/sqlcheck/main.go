// Command sqlcheck — власне правило статичного аналізу для RoomFlow:
// забороняє передавати в database/sql (Query, QueryRow, Exec, ...) SQL, зібраний через fmt.Sprintf
// або конкатенацію рядків. Саме такий дефект SonarQube пропустив у ЛР2 (ризик SQL-ін'єкції).
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

var skippedDirs = map[string]bool{"testdata": true, "vendor": true}

const message = "SQL query is built dynamically; use placeholders (?) and pass values as arguments"

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

// CheckDir analyses all non-test Go files under root (testdata, vendor and hidden directories are skipped).
func CheckDir(root string) ([]Finding, error) {
	var all []Finding
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipDir(path, root, d.Name())
		}
		if !isSource(path) {
			return nil
		}
		found, err := checkPath(path)
		all = append(all, found...)
		return err
	})
	return all, err
}

func skipDir(path, root, name string) error {
	if path != root && (skippedDirs[name] || strings.HasPrefix(name, ".")) {
		return filepath.SkipDir
	}
	return nil
}

func isSource(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

func checkPath(path string) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	return CheckFile(fset, file), nil
}

// CheckFile reports dynamic SQL passed to database/sql query methods.
func CheckFile(fset *token.FileSet, file *ast.File) []Finding {
	built := dynamicStrings(file)
	var out []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		if q := queryArg(n); q != nil && (isDynamic(q) || isBuiltVar(q, built)) {
			out = append(out, Finding{fset.Position(q.Pos()), message})
		}
		return true
	})
	return out
}

// queryArg returns the SQL text argument of a database/sql query call, or nil.
func queryArg(n ast.Node) ast.Expr {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !sqlMethods[sel.Sel.Name] {
		return nil
	}
	if strings.HasSuffix(sel.Sel.Name, "Context") && len(call.Args) > 1 {
		return call.Args[1]
	}
	return call.Args[0]
}

func isBuiltVar(e ast.Expr, built map[string]bool) bool {
	id, ok := e.(*ast.Ident)
	return ok && built[id.Name]
}

// dynamicStrings collects variables assigned from fmt.Sprintf or string concatenation.
func dynamicStrings(file *ast.File) map[string]bool {
	res := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		if as, ok := n.(*ast.AssignStmt); ok {
			markDynamic(as, res)
		}
		return true
	})
	return res
}

func markDynamic(as *ast.AssignStmt, res map[string]bool) {
	for i, rhs := range as.Rhs {
		if i >= len(as.Lhs) || !isDynamic(rhs) {
			continue
		}
		if id, ok := as.Lhs[i].(*ast.Ident); ok {
			res[id.Name] = true
		}
	}
}

func isDynamic(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.CallExpr:
		return isSprint(v)
	case *ast.BinaryExpr:
		return v.Op == token.ADD && (!isLiteral(v.X) || !isLiteral(v.Y))
	}
	return false
}

func isSprint(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "fmt" && strings.HasPrefix(sel.Sel.Name, "Sprint")
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
