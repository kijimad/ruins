package lintrule_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoRowEnumCompare は raw に行を持つ enum の値と比べることを禁止する。値に行があるなら、
// 値との比較は行に書ける性質の代わりになっている。判定は行の欄を引いて行い、段階や種類を
// 足したり調整したりしても Go を変えずに済むようにする。行を持たない enum は仕組みを選ぶので対象外。
func TestNoRowEnumCompare(t *testing.T) {
	t.Parallel()
	root := moduleRoot(t)
	fset := token.NewFileSet()

	models, err := parser.ParseFile(fset, filepath.Join(root, "internal", "oapi", "models.gen.go"), nil, 0)
	require.NoError(t, err)
	consts := rowEnumConsts(models)
	// 生成物の形が変わって集められなくなると検査が黙って空振りするので、既知の値で確かめる
	require.Subset(t, slices.Collect(maps.Keys(consts)), []string{"Fresh", "Outdoor", "HungerStageIdStarving"})

	var found []string
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && path == filepath.Join(root, "internal", "oapi") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_gen.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		// ビルドが通れば必ずパースできる。失敗は検査漏れになるのでハードエラーにする
		require.NoError(t, perr, "パースできない: %s", path)
		for _, pos := range rowEnumCompares(f, consts) {
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
	require.Empty(t, found, "raw に行を持つ enum の値と比べない。行に性質の欄を足して引く:\n%s", strings.Join(found, "\n"))
}

func TestRowEnumConsts(t *testing.T) {
	t.Parallel()
	src := `package oapi
type Stage string
const (
	Fresh  Stage = "fresh"
	Rotten Stage = "rotten"
)
type Rank int
const (
	First Rank = iota
	Second
	Untyped = 9
)
type Kind string
const Furniture Kind = "furniture"
type StageRow struct {
	Id   Stage
	Name string
}
type RankRow struct {
	Id Rank
}
type Placed struct {
	Kind Kind
}
`
	f, err := parser.ParseFile(token.NewFileSet(), "models.gen.go", src, 0)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"Fresh", "Rotten", "First", "Second"}, slices.Collect(maps.Keys(rowEnumConsts(f))))
}

func TestRowEnumCompares(t *testing.T) {
	t.Parallel()
	consts := map[string]bool{"Fresh": true, "Rotten": true}
	tests := []struct {
		name string
		body string
		want int
	}{
		{"== で比べると違反", `_ = s == oapi.Rotten`, 1},
		{"!= で比べると違反", `_ = s != oapi.Fresh`, 1},
		{"左に置いても違反", `_ = oapi.Rotten == s`, 1},
		{"case に並べると違反", "switch s {\ncase oapi.Fresh, oapi.Rotten:\n}", 2},
		{"行を持たない enum は適合", `_ = k == oapi.Furniture`, 0},
		{"代入は適合", `s = oapi.Rotten`, 0},
		{"map のキーは適合", `_ = map[oapi.Stage]int{oapi.Fresh: 1}`, 0},
		{"行の欄で引くのは適合", `_ = defs[s].Marker`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := "package p\nfunc f() {\n" + tt.body + "\n}\n"
			f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, 0)
			require.NoError(t, err)
			assert.Len(t, rowEnumCompares(f, consts), tt.want)
		})
	}
}

// rowEnumConsts は生成された oapi のモデルから、構造体の Id の欄の型になっている enum の定数名を集める。
// Id の欄を持つ構造体は raw の行なので、その型の値には必ず行がある。
func rowEnumConsts(f *ast.File) map[string]bool {
	rowTypes := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range st.Fields.List {
			typ, ok := field.Type.(*ast.Ident)
			if !ok {
				continue
			}
			for _, name := range field.Names {
				if name.Name == "Id" {
					rowTypes[typ.Name] = true
				}
			}
		}
		return true
	})

	consts := map[string]bool{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		// 型も値も省いた定数は、ブロックの直前の型と式を引き継ぐ
		var typeName string
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			switch typ := vs.Type.(type) {
			case *ast.Ident:
				typeName = typ.Name
			case nil:
				if len(vs.Values) > 0 {
					typeName = ""
				}
			default:
				typeName = ""
			}
			if !rowTypes[typeName] {
				continue
			}
			for _, name := range vs.Names {
				consts[name.Name] = true
			}
		}
	}
	return consts
}

// rowEnumCompares は consts に含まれる oapi の定数を == か != の片側、または case に置いた位置を返す
func rowEnumCompares(f *ast.File, consts map[string]bool) []token.Pos {
	isRowConst := func(e ast.Expr) bool {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		pkg, ok := sel.X.(*ast.Ident)
		return ok && pkg.Name == "oapi" && consts[sel.Sel.Name]
	}
	var found []token.Pos
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if (n.Op == token.EQL || n.Op == token.NEQ) && (isRowConst(n.X) || isRowConst(n.Y)) {
				found = append(found, n.Pos())
			}
		case *ast.CaseClause:
			for _, e := range n.List {
				if isRowConst(e) {
					found = append(found, e.Pos())
				}
			}
		}
		return true
	})
	return found
}
