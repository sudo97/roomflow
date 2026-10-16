// Package arch містить архітектурні тести: автоматичні перевірки того, що залежності
// між пакетами відповідають задуманій шаровій архітектурі (домен не залежить від інфраструктури).
package arch

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/sudo97/roomflow/"

// forbidden: пакет -> імпорти, яких у ньому бути не повинно.
var forbidden = map[string][]string{
	"internal/booking": {"net/http", "database/sql", module + "internal/store", module + "internal/httpapi"},
	"internal/store":   {"net/http", module + "internal/httpapi"},
	"internal/httpapi": {"database/sql", module + "internal/store"},
	// Наскрізні модулі (спостережуваність, feature flags) не залежать від бізнес-логіки.
	"internal/obs":       {module + "internal/booking", module + "internal/store", module + "internal/httpapi"},
	"internal/flags":     {module + "internal/booking", module + "internal/store", module + "internal/httpapi", "net/http"},
	"internal/cache":     {module + "internal/booking", module + "internal/store", module + "internal/httpapi"},
	"internal/trace":     {module + "internal/booking", module + "internal/store", module + "internal/httpapi"},
	"internal/ratelimit": {module + "internal/booking", module + "internal/store", module + "internal/httpapi"},
	"internal/chaos":     {module + "internal/booking", module + "internal/store", module + "internal/httpapi"},
}

func imports(t *testing.T, dir string) map[string]string {
	t.Helper()
	res := map[string]string{}
	files, _ := filepath.Glob(filepath.Join("..", "..", dir, "*.go"))
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		ast, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range ast.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			res[p] = filepath.Base(f)
		}
	}
	return res
}

func TestLayering(t *testing.T) {
	for pkg, banned := range forbidden {
		got := imports(t, pkg)
		if len(got) == 0 {
			t.Fatalf("no sources found for %s", pkg)
		}
		for _, b := range banned {
			if file, ok := got[b]; ok {
				t.Errorf("%s/%s imports %s: the dependency direction is violated", pkg, file, b)
			}
		}
	}
}
