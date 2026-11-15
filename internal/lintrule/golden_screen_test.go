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

// TestGoldenScreenFromNewScreen は vrt.AssertFrameGolden に渡す描き先を ebiten.NewImage で作ることを禁止する。
// managed の描き先は共有アトラスに載り、配置が並行テストの生成順で変わって golden が揺れる。vrt.NewScreen で作る。
func TestGoldenScreenFromNewScreen(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)
	fset := token.NewFileSet()
	var found []string

	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		// ビルドが通れば必ずパースできる。失敗は検査漏れになるのでハードエラーにする
		require.NoError(t, perr, "パースできない: %s", path)
		for _, pos := range managedGoldenScreens(f) {
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
	require.Empty(t, found, "golden の描き先は vrt.NewScreen で作る:\n%s", strings.Join(found, "\n"))
}

func TestManagedGoldenScreens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want int
	}{
		{"NewImage で作った描き先を渡すと違反", "screen := ebiten.NewImage(2, 2)\nvrt.AssertFrameGolden(t, \"x\", screen)", 1},
		{"NewImageWithOptions も違反", "screen := ebiten.NewImageWithOptions(r, nil)\nvrt.AssertFrameGolden(t, \"x\", screen)", 1},
		{"引数で直に作っても違反", "vrt.AssertFrameGolden(t, \"x\", ebiten.NewImage(2, 2))", 1},
		{"var で宣言しても違反", "var screen = ebiten.NewImage(2, 2)\nvrt.AssertFrameGolden(t, \"x\", screen)", 1},
		{"NewScreen は適合", "screen := vrt.NewScreen(2, 2)\nvrt.AssertFrameGolden(t, \"x\", screen)", 0},
		{"描画元の NewImage は適合", "src := ebiten.NewImage(2, 2)\nscreen := vrt.NewScreen(2, 2)\nscreen.DrawImage(src, nil)\nvrt.AssertFrameGolden(t, \"x\", screen)", 0},
		{"受け取った描き先は適合", "vrt.AssertFrameGolden(t, \"x\", frame)", 0},
		{"vrt の中の修飾なしの呼び出しも違反", "screen := ebiten.NewImage(2, 2)\nAssertFrameGolden(t, \"x\", screen)", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := "package p\nfunc TestX(t *testing.T, r image.Rectangle, frame *ebiten.Image) {\n" + tt.body + "\n}\n"
			f, err := parser.ParseFile(token.NewFileSet(), "x_test.go", src, 0)
			require.NoError(t, err)
			assert.Len(t, managedGoldenScreens(f), tt.want)
		})
	}
}

// managedGoldenScreens は vrt.AssertFrameGolden の描き先が ebiten.NewImage 系で作られた呼び出しの位置を返す。
// 描き先の変数は関数ごとに名前で突き合わせる。import の別名は追わない
func managedGoldenScreens(f *ast.File) []token.Pos {
	var out []token.Pos
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		managed := map[string]bool{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.AssignStmt:
				for i, lhs := range s.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && i < len(s.Rhs) && isNewImageCall(s.Rhs[i]) {
						managed[id.Name] = true
					}
				}
			case *ast.ValueSpec:
				for i, id := range s.Names {
					if i < len(s.Values) && isNewImageCall(s.Values[i]) {
						managed[id.Name] = true
					}
				}
			}
			return true
		})
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isAssertFrameGolden(call.Fun) || len(call.Args) < 3 {
				return true
			}
			arg := call.Args[2]
			if id, ok := arg.(*ast.Ident); ok && managed[id.Name] || isNewImageCall(arg) {
				out = append(out, call.Pos())
			}
			return true
		})
	}
	return out
}

// isAssertFrameGolden は式が vrt.AssertFrameGolden か、vrt パッケージの中の修飾なしの呼び出しかを返す
func isAssertFrameGolden(e ast.Expr) bool {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name == "AssertFrameGolden"
	}
	return isSelector(e, "vrt", "AssertFrameGolden")
}

// isNewImageCall は式が ebiten.NewImage か ebiten.NewImageWithOptions の呼び出しかを返す
func isNewImageCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	return ok && (isSelector(call.Fun, "ebiten", "NewImage") || isSelector(call.Fun, "ebiten", "NewImageWithOptions"))
}

// isSelector は式が pkg.name の形かを返す
func isSelector(e ast.Expr, pkg, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == pkg
}
