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

// styleMutators は装飾を直接いじる ui.Container のメソッド名。画面層では呼ばない。
// メソッド名の構文一致で拾う。この名前のメソッドは ui.Container にしか無い。
var styleMutators = map[string]bool{
	"SetStyle":               true,
	"SetPadding":             true,
	"SetBackgroundNineSlice": true,
	"SetBottomLine":          true,
}

// colorLiteralTypes は theme を迂回する生の色リテラルの型名。色は theme のトークンで持つ。
var colorLiteralTypes = map[string]bool{
	"RGBA":   true,
	"NRGBA":  true,
	"RGBA64": true,
	"Gray":   true,
	"Gray16": true,
}

// TestScreenLayerStyleLint は画面層がスタイルを手組みしていないことを静的に検証する。
// 装飾の直接指定はビルドを通っても見た目の一貫性を静かに壊し、レビューでは繰り返し
// すり抜けてきたため、AST で fail-closed に検知する。
// 禁止するのは装飾だけで、ui.Row・ui.VBox・ui.NewText・Layout などの構成は画面の自由にする。
// 検知したら menuframe の部品 TabScreen・PanelScreen・PanelBox・RenderList か、
// theme のトークンへ寄せて直す。
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
				switch node := n.(type) {
				case *ast.CallExpr:
					sel, ok := node.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "ui" && sel.Sel.Name == "Panel" {
						report(node, "ui.Panel の直接呼び出し。枠は menuframe.TabScreen/PanelScreen/PanelBox が持つ")
					}
					if styleMutators[sel.Sel.Name] {
						report(node, sel.Sel.Name+" の呼び出し。装飾は menuframe の部品が持つ")
					}
				case *ast.CompositeLit:
					sel, ok := node.Type.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					ident, ok := sel.X.(*ast.Ident)
					if !ok {
						return true
					}
					if ident.Name == "ui" && sel.Sel.Name == "BoxStyle" {
						report(node, "ui.BoxStyle の指定。塗りと枠は menuframe の部品が持つ")
					}
					if ident.Name == "color" && colorLiteralTypes[sel.Sel.Name] {
						report(node, "生の色リテラル。色は theme のトークンで持つ")
					}
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
