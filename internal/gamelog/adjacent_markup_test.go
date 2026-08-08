package gamelog

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

	"github.com/stretchr/testify/require"
)

// TestNoAdjacentMarkup は同じ Logger への隣接 Markup 呼び出しを禁止する。
// Markup は断片を1つの Logger に積んで自身を返すため、logger.Markup(a).Markup(b) の隣接
// チェーンは1ログ行を断片へ割ったものになり、語順ごとの翻訳を妨げる。1ログ行は1 Markup 文字列にする。
// Markup は gamelog.Logger 固有のメソッド名なので、型情報なしの構文一致で拾える。
func TestNoAdjacentMarkup(t *testing.T) {
	t.Parallel()
	root := moduleRootForLint(t)
	fset := token.NewFileSet()
	var found []string

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Logf("パース失敗をスキップする: %s: %v", path, perr)
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			outer, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := markupSelector(outer)
			if !ok {
				return true
			}
			inner, ok := sel.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			if _, ok := markupSelector(inner); !ok {
				return true
			}
			pos := fset.Position(sel.Sel.Pos())
			rel, relErr := filepath.Rel(root, pos.Filename)
			if relErr != nil {
				rel = pos.Filename
			}
			found = append(found, rel+":"+strconv.Itoa(pos.Line))
			return true
		})
		return nil
	})
	require.NoError(t, err)

	sort.Strings(found)
	require.Empty(t, found, "隣接 Markup は1つに統合する。1ログ行は1 Markup 文字列にし、断片連結を避ける:\n%s", strings.Join(found, "\n"))
}

// markupSelector は式が .Markup(...) 呼び出しならそのセレクタを返す。
func markupSelector(c *ast.CallExpr) (*ast.SelectorExpr, bool) {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Markup" {
		return nil, false
	}
	return sel, true
}

// moduleRootForLint は go.mod を上位へ辿ってリポジトリルートを返す。
func moduleRootForLint(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod が見つからない")
		dir = parent
	}
}
