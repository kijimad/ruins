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

// uiCoreConsumerDirs は uicore を直接 import してよい画面側のディレクトリ。
// widgets の部品は atom を組み上げる側なので対象にしない。
var uiCoreConsumerDirs = []string{"internal/states", "internal/menuloop", "internal/systems"}

// screenUICoreSymbols は画面側が名指ししてよい uicore のシンボル。組み上がったツリーを
// 受け取って描くのに要るものだけを並べる。許可制なので、uicore に面が増えても画面へは
// 自動的には開かない。
//
// ここに無い装飾とレイアウトの面、BoxStyle・Panel・NineSlice・Container・Row・FlexColumn
// などを画面が名指しすると、意匠の手組みとレイアウトエンジンの迂回が画面へ漏れる。
// 配置は部品が済ませてから画面へ渡す。
var screenUICoreSymbols = map[string]bool{
	"Drawable":         true,
	"Text":             true,
	"NewText":          true,
	"NewGroup":         true,
	"EbitenCanvas":     true,
	"NewEbitenCanvas":  true,
	"MeasureText":      true,
	"MeasureTextWidth": true,
}

// TestScreenLayerStyleLint は画面層がスタイルを手組みしていないことを静的に検証する。
// 意匠は menuframe の部品と theme のトークンが持ち、画面はデータと配置だけを書く。
// uicore.NewText は color.Color を受けるため、theme を迂回した生の色リテラルは構文上
// 書けてしまう。それを AST で fail-closed に検知する。検知したら theme の色トークンへ寄せて直す。
func TestScreenLayerStyleLint(t *testing.T) {
	t.Parallel()

	violations := inspectScreenFiles(t, screenLayerDirs, func(report func(ast.Node, string), f *ast.File) {
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
	})

	require.Empty(t, violations, "画面層にスタイルの手組みがある。部品とトークンへ寄せる:\n%s", strings.Join(violations, "\n"))
}

// TestScreenLayerUICoreSurface は画面側が名指しする uicore のシンボルを許可制に保つ。
// depguard は import の可否までしか見ないので、import を許した先で何を使うかはここで縛る。
func TestScreenLayerUICoreSurface(t *testing.T) {
	t.Parallel()

	violations := inspectScreenFiles(t, uiCoreConsumerDirs, func(report func(ast.Node, string), f *ast.File) {
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "uicore" {
				return true
			}
			if !screenUICoreSymbols[sel.Sel.Name] {
				report(sel, "uicore."+sel.Sel.Name+" は画面が名指しできない面。組み立ては widgets の部品に置く")
			}
			return true
		})
	})

	require.Empty(t, violations, "画面が uicore の許可外の面に触れている:\n%s", strings.Join(violations, "\n"))
}

// inspectScreenFiles は dirs 配下の非テスト Go ファイルを AST で歩き、visit が報告した
// 違反をファイル位置付きで集めて返す。
func inspectScreenFiles(t *testing.T, dirs []string, visit func(report func(ast.Node, string), f *ast.File)) []string {
	t.Helper()

	root := moduleRootDir(t)
	fset := token.NewFileSet()
	var violations []string

	report := func(n ast.Node, msg string) {
		p := fset.Position(n.Pos())
		rel, err := filepath.Rel(root, p.Filename)
		if err != nil {
			rel = p.Filename
		}
		violations = append(violations, rel+":"+strconv.Itoa(p.Line)+": "+msg)
	}

	for _, dir := range dirs {
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
			visit(report, f)

			return nil
		})
		require.NoError(t, err)
	}
	sort.Strings(violations)

	return violations
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
