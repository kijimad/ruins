package menuframe_test

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

// screenLayerDirs はスタイルの手組みを禁止する画面層のディレクトリ。
// 画面 states とコントローラ menuloop に書いてよいのはデータ・列の宣言・文言・配置だけで、
// 意匠は menuframe の部品と theme のトークンが持つ。
var screenLayerDirs = []string{"internal/states", "internal/menuloop"}

// colorLiteralTypes は theme を迂回する生の色リテラルの型名。色は theme のトークンで持つ。
var colorLiteralTypes = map[string]bool{
	"RGBA":   true,
	"NRGBA":  true,
	"RGBA64": true,
	"Gray":   true,
	"Gray16": true,
}

// TestScreenLayerStyleLint は画面層がスタイルを手組みしていないことを静的に検証する。
// プリミティブ実体は widgets/uicore にあり、画面層は import 自体がコンパイル不能なので、
// 枠・塗り・装飾ミューテータの手組みは API の面で既に不可能になっている。
// ここでは型システムで塞げない残りだけを検知する。ui.NewText は color.Color を受けるため、
// theme を迂回した生の色リテラルは構文上書けてしまう。それを AST で fail-closed に検知する。
// 検知したら theme の色トークンへ寄せて直す。
func TestScreenLayerStyleLint(t *testing.T) {
	t.Parallel()

	root := moduleRootDir(t)
	var violations []string
	fset := token.NewFileSet()

	report := func(n ast.Node, msg string) {
		p := fset.Position(n.Pos())
		rel, err := filepath.Rel(root, p.Filename)
		if err != nil {
			rel = p.Filename
		}
		violations = append(violations, rel+":"+strconv.Itoa(p.Line)+": "+msg)
	}

	for _, dir := range screenLayerDirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(fset, path, nil, 0)
			// ビルドが通れば必ずパースできる。失敗は検査漏れになるのでハードエラーにする
			require.NoError(t, perr, "パースできない: %s", path)
			ast.Inspect(f, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				sel, ok := lit.Type.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if ident.Name == "color" && colorLiteralTypes[sel.Sel.Name] {
					report(lit, "生の色リテラル。色は theme のトークンで持つ")
				}
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}

	sort.Strings(violations)
	require.Empty(t, violations, "画面層にスタイルの手組みがある。部品とトークンへ寄せる:\n%s", strings.Join(violations, "\n"))
}

// moduleRootDir は go.mod のあるモジュールルートを返す。テストの実行位置に依存しない。
func moduleRootDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "go.mod が見つからない")
		dir = parent
	}
}
