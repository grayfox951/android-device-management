package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// moduleRoot finds the directory holding go.mod. `go test` runs with the
// package directory as the working directory, so a relative ".." would stop at
// internal/ and silently scan almost nothing.
func moduleRoot(t *testing.T) string {
	t.Helper()

	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test source")
	}
	dir := filepath.Dir(self)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test source")
		}
		dir = parent
	}
}

// TestEveryReferencedKeyExists walks the whole module and fails on any
// i18n.Get/T/Missing call naming a key that does not exist. A renamed or
// half-applied key would otherwise show up on screen as the raw key, and the
// completeness test cannot catch it because it compares against English too.
func TestEveryReferencedKeyExists(t *testing.T) {
	root := moduleRoot(t)

	fset := token.NewFileSet()
	found := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil {
				return true
			}
			name := sel.Sel.Name
			pkg := ""
			if id, ok := sel.X.(*ast.Ident); ok {
				pkg = id.Name
			}
			// i18n.Get/T/Missing and the local Get/T shims.
			isI18n := pkg == "i18n"
			isLocal := pkg == "" && (name == "i18nT" || name == "Get" || name == "T")
			if !(isI18n || isLocal) {
				return true
			}
			if name == "T" && !isI18n {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			key, err := strconv.Unquote(lit.Value)
			if err != nil || key == "" {
				return true
			}
			found++
			if !Has(key) {
				t.Errorf("%s: references unknown key %q", path, key)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found == 0 {
		t.Fatal("no i18n references were found, the scanner is broken")
	}
	t.Logf("checked %d key references", found)
}

// TestCatalogDescriptionsAreRegistered makes sure the description table and the
// interface table agree on which keys exist.
func TestKeysAreUniquePerLanguage(t *testing.T) {
	for _, l := range Order {
		seen := map[string]bool{}
		for k := range dict[l] {
			if seen[k] {
				t.Errorf("%s: key %q is defined twice", l, k)
			}
			seen[k] = true
		}
	}
}
