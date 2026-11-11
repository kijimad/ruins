package lintrule_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoErrorStringContains はテストで err.Error() を Contains に渡すことを禁止する。
// どのエラーかの同定は sentinel と require.ErrorIs で行い、値を埋め込む動的なメッセージの中身だけ
// assert.ErrorContains で確かめる。文字列の部分一致は、同じ文言の別のエラーでも通ってしまう。
func TestNoErrorStringContains(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)
	fset := token.NewFileSet()
	var found []string

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		// ビルドが通れば必ずパースできる。失敗は検査漏れになるのでハードエラーにする
		require.NoError(t, perr, "パースできない: %s", path)
		for _, pos := range errorStringContains(f) {
			p := fset.Position(pos)
			rel, relErr := filepath.Rel(root, p.Filename)
			if relErr != nil {
				rel = p.Filename
			}
			found = append(found, rel+":"+strconv.Itoa(p.Line))
		}
		return nil
	})
	require.NoError(t, err)

	sort.Strings(found)
	require.Empty(t, found, "err.Error() を Contains に渡さない。同定は require.ErrorIs、動的なメッセージの中身は assert.ErrorContains で確かめる:\n%s", strings.Join(found, "\n"))
}

func TestErrorStringContains(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want int
	}{
		{"assert.Contains に err.Error() を渡すと違反", `assert.Contains(t, err.Error(), "x")`, 1},
		{"require.Contains も違反", `require.Contains(t, err.Error(), "x")`, 1},
		{"NotContains も違反", `assert.NotContains(t, err.Error(), "x")`, 1},
		{"書式付きの Containsf も違反", `assert.Containsf(t, err.Error(), "x", "m")`, 1},
		{"書式付きの NotContainsf も違反", `assert.NotContainsf(t, err.Error(), "x", "m")`, 1},
		{"フィールドのエラーでも違反", `assert.Contains(t, got.err.Error(), "x")`, 1},
		{"ErrorContains は適合", `assert.ErrorContains(t, err, "x")`, 0},
		{"文字列の Contains は適合", `assert.Contains(t, s, "x")`, 0},
		{"引数を取る Error 呼び出しは対象外", `assert.Contains(t, log.Error("a"), "x")`, 0},
		{"完全一致の Equal は適合", `assert.Equal(t, "x", err.Error())`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := "package p\nfunc TestX() {\n" + tt.body + "\n}\n"
			f, err := parser.ParseFile(token.NewFileSet(), "x_test.go", src, 0)
			require.NoError(t, err)
			assert.Len(t, errorStringContains(f), tt.want)
		})
	}
}

// containsAsserts は err.Error() を渡すと違反になる testify の関数名
var containsAsserts = map[string]bool{
	"Contains":     true,
	"NotContains":  true,
	"Containsf":    true,
	"NotContainsf": true,
}

// errorStringContains は assert か require の Contains 系の第2引数が、引数なしの .Error() 呼び出しである位置を返す。
func errorStringContains(f *ast.File) []token.Pos {
	var found []token.Pos
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !containsAsserts[sel.Sel.Name] {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || (pkg.Name != "assert" && pkg.Name != "require") {
			return true
		}
		arg, ok := call.Args[1].(*ast.CallExpr)
		if !ok || len(arg.Args) != 0 {
			return true
		}
		if argSel, ok := arg.Fun.(*ast.SelectorExpr); ok && argSel.Sel.Name == "Error" {
			found = append(found, call.Pos())
		}
		return true
	})
	return found
}
