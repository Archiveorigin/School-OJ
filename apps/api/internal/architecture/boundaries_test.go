package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Scan entire layers, including future subdirectories. Exceptions document the
// two existing adapter imports, never exempt an entire file or directory.
func TestApplicationDependencyDirection(t *testing.T) {
	exceptions := map[string]string{
		"services/audit.go:github.com/gin-gonic/gin": "legacy audit adapter takes request identity; migrate to an explicit actor DTO",
		"services/problem_package.go:net/http":       "DetectContentType only, no HTTP transport",
	}
	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel("..", path)
		rel = filepath.ToSlash(rel)
		var forbidden []string
		switch {
		case strings.HasPrefix(rel, "domain/"):
			forbidden = []string{"gin-gonic", "gorm.io", "redis", "/handlers", "/repositories", "/services", "net/http"}
		case strings.HasPrefix(rel, "services/"):
			forbidden = []string{"gin-gonic", "/handlers", "net/http"}
		case strings.HasPrefix(rel, "repositories/"):
			forbidden = []string{"gin-gonic", "/handlers", "/services", "net/http"}
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			dependency, _ := strconv.Unquote(spec.Path.Value)
			for _, rule := range forbidden {
				if strings.Contains(dependency, rule) && exceptions[rel+":"+dependency] == "" {
					t.Errorf("%s imports forbidden dependency %s", rel, dependency)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTeamJoinHandlerContainsNoPersistenceOperations(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "../handlers/team_join.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{"Where": true, "Exec": true, "Raw": true, "Transaction": true, "Create": true, "Updates": true, "Delete": true, "Clauses": true}
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok && forbidden[selector.Sel.Name] {
				t.Errorf("join handler contains %s", selector.Sel.Name)
			}
		}
		return true
	})
}

func TestAPIWorkerPinnedVersionContract(t *testing.T) {
	for _, path := range []string{"../models/models.go", "../../../worker/internal/models/models.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok || spec.Name.Name != "Submission" {
				return true
			}
			structure := spec.Type.(*ast.StructType)
			for _, field := range structure.Fields.List {
				for _, name := range field.Names {
					if name.Name == "ProblemVersionID" {
						found = true
						if field.Tag == nil || !strings.Contains(field.Tag.Value, "problem_version_id") {
							t.Errorf("%s: pinned version wire name changed", path)
						}
					}
				}
			}
			return false
		})
		if !found {
			t.Errorf("%s no immutable submission version", path)
		}
	}
}
