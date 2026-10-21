package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoInlineAdministratorBypass(t *testing.T) {
	// These are identity invariants/validation, not authorization bypasses.
	exceptions := map[string]string{
		"removeProblemAuthor":  "built-in administrator author grant cannot be removed independently",
		"updateUserPermission": "same built-in grant invariant",
		"addCourseMember":      "validate global identity versus course membership type",
	}
	files, err := filepath.Glob("../handlers/*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || exceptions[fn.Name.Name] != "" {
				continue
			}
			ast.Inspect(fn, func(node ast.Node) bool {
				expr, ok := node.(*ast.BinaryExpr)
				if !ok {
					return true
				}
				for _, operand := range []ast.Expr{expr.X, expr.Y} {
					if field, ok := operand.(*ast.SelectorExpr); ok && field.Sel.Name == "RoleAdmin" {
						t.Errorf("%s %s uses an inline administrator identity check", path, fn.Name.Name)
					}
				}
				return true
			})
		}
	}
}

func TestSensitiveCollectionsKeepSQLScopes(t *testing.T) {
	data, err := os.ReadFile("../handlers/server.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for fn, scope := range map[string]string{"listCourses": "visibility.Courses(user)", "listClasses": "visibility.Classes(user)", "listAssignments": "visibility.CourseWork(user)", "listExams": "visibility.CourseWork(user)", "listSubmissions": "visibility.Submissions(user)", "listPreparedProblems": "visibility.PreparedProblems(user)"} {
		start := strings.Index(source, "func (s Server) "+fn+"(")
		if start < 0 {
			t.Fatal("missing entry", fn)
		}
		rest := source[start:]
		end := strings.Index(rest[1:], "\nfunc ")
		if end >= 0 {
			rest = rest[:end+1]
		}
		if !strings.Contains(rest, scope) {
			t.Errorf("%s bypasses registered SQL scope %s", fn, scope)
		}
	}
}
