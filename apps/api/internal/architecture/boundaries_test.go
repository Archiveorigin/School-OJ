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

func TestTeamApplicationDependencyDirection(t *testing.T) {
	cases := []struct {
		path      string
		forbidden []string
	}{
		{"../domain/team", []string{"gin-gonic", "gorm.io", "redis", "/handlers", "/repositories", "/services", "net/http"}},
		{"../services/team_access.go", []string{"gin-gonic", "gorm.io", "redis", "/handlers", "/repositories", "net/http"}},
		{"../repositories", []string{"gin-gonic", "/handlers", "/services", "net/http"}},
	}
	for _, item := range cases {
		err := filepath.WalkDir(item.path, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				dependency, _ := strconv.Unquote(spec.Path.Value)
				for _, forbidden := range item.forbidden {
					if strings.Contains(dependency, forbidden) {
						t.Errorf("%s imports forbidden dependency %s", path, dependency)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
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
