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

// TestNoTileNameCompare はタイルの名前の定数 consts.TileName* や、間取りのタイルの欄 Tiles.Floor・Wall・Void と比べることを禁止する。タイルが何かは
// 名前でなく、行の欄 wall・blockPass・earth などの性質で判定する。名前で比べると、同じ性質の別のタイルを
// 足したときに Go を直す必要が出る。定数でタイルを置いたり引いたりするのは判定でないので対象外。
func TestNoTileNameCompare(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)
	fset := token.NewFileSet()

	var found []string
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_gen.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		// ビルドが通れば必ずパースできる。失敗は検査漏れになるのでハードエラーにする
		require.NoError(t, perr, "パースできない: %s", path)
		for _, pos := range tileNameCompares(f) {
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
	require.Empty(t, found, "タイルの名前と比べない。タイルの行の性質で判定する:\n%s", strings.Join(found, "\n"))
}

func TestTileNameCompares(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want int
	}{
		{"== で比べると違反", `_ = id == consts.TileNameWall`, 1},
		{"!= で比べると違反", `_ = id != consts.TileNameFloor`, 1},
		{"左に置いても違反", `_ = consts.TileNameVoid == id`, 1},
		{"case に並べると違反", "switch id {\ncase consts.TileNameWall, consts.TileNameVoid:\n}", 2},
		{"タイルを置くのは適合", `cell = Cell{Terrain: consts.TileNameWall}`, 0},
		{"タイルを引くのは適合", `_ = get(consts.TileNameVoid)`, 0},
		{"性質で判定するのは適合", `_ = tile.BlockPass`, 0},
		{"タイル以外の定数は適合", `_ = n == consts.MapTileWidth`, 0},
		{"間取りのタイルの欄と比べると違反", `_ = tile.Id == r.Tiles.Floor`, 1},
		{"入れ子の間取りのタイルの欄と比べると違反", `_ = tile.Id != b.Layout.Tiles.Wall`, 1},
		{"間取りのタイルの欄を置くのは適合", `_ = get(r.Tiles.Void)`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := "package p\nfunc f() {\n" + tt.body + "\n}\n"
			f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, 0)
			require.NoError(t, err)
			assert.Len(t, tileNameCompares(f), tt.want)
		})
	}
}

// tileNameCompares はファイルの中でタイルの名前と比べる式の位置を返す
func tileNameCompares(f *ast.File) []token.Pos {
	isTileName := func(e ast.Expr) bool {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "consts" {
			return strings.HasPrefix(sel.Sel.Name, "TileName")
		}
		tiles, ok := sel.X.(*ast.SelectorExpr)
		return ok && tiles.Sel.Name == "Tiles" && (sel.Sel.Name == "Floor" || sel.Sel.Name == "Wall" || sel.Sel.Name == "Void")
	}
	var found []token.Pos
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) && (isTileName(n.X) || isTileName(n.Y)) {
				found = append(found, n.Pos())
			}
		case *ast.CaseClause:
			for _, e := range n.List {
				if isTileName(e) {
					found = append(found, e.Pos())
				}
			}
		}
		return true
	})
	return found
}
