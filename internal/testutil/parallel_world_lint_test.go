package testutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// worldTypePath は共有を禁止する world 型の完全修飾パッケージパス
const worldTypePath = "github.com/kijimaD/ruins/internal/world"

// TestNoSharedWorldInParallelSubtests は並列サブテストでの world 共有を禁止する。
// Ark の world は並行安全でないため、t.Parallel() するサブテストが外側で作った world に
// 触れるとデータ競合になる。world は各サブテストの中で作る。
// 型情報で判定するので、変数名やヘルパ経由の取得によらず w.World 型の共有を拾う
func TestNoSharedWorldInParallelSubtests(t *testing.T) {
	t.Parallel()
	if raceEnabled() {
		// packages.Load の並列型チェックが Go 1.27 の go/types 内部で競合し、
		// 上流由来の race 検出が出る。静的検査なので race なしの実行が担えば十分
		t.Skip("go/types の上流競合を避けるため -race では実行しない")
	}
	root := moduleRootForLint(t)
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: true,
		Dir:   root,
	}
	pkgs, err := packages.Load(cfg, "./internal/...")
	require.NoError(t, err)

	seen := map[string]bool{}
	var found []string
	for _, p := range pkgs {
		// 型解決に失敗したパッケージは検査漏れになるのでハードエラーにする
		require.Empty(t, p.Errors, "パッケージを読み込めない: %s", p.PkgPath)
		for _, f := range p.Syntax {
			filename := p.Fset.Position(f.Pos()).Filename
			if !strings.HasSuffix(filename, "_test.go") {
				continue
			}
			for _, pos := range sharedWorldViolations(f, p.TypesInfo, isRuinsWorld) {
				pt := p.Fset.Position(pos)
				rel, relErr := filepath.Rel(root, pt.Filename)
				if relErr != nil {
					rel = pt.Filename
				}
				loc := rel + ":" + strconv.Itoa(pt.Line)
				// テストバリアントで同じファイルが複数回現れるので重複を除く
				if !seen[loc] {
					seen[loc] = true
					found = append(found, loc)
				}
			}
		}
	}
	assert.Empty(t, found, "並列サブテストが外側の world を共有している。world はサブテストの中で作る")
}

// TestSharedWorldViolations は検出器そのものを両方向で検証する。
// 適合だけでは検出力の保証にならないので、違反を実際に検出できることも確認する
func TestSharedWorldViolations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		body      string
		wantFound bool
	}{
		{
			name: "外側のworldを並列サブテストで参照すると違反",
			body: `world := InitTestWorld()
				tt.Run("a", func(tt T) { tt.Parallel(); world.ECS.NewEntity() })`,
			wantFound: true,
		},
		{
			name: "変数名がworldでなくても型で拾う",
			body: `earth := InitTestWorld()
				tt.Run("a", func(tt T) { tt.Parallel(); earth.ECS.NewEntity() })`,
			wantFound: true,
		},
		{
			name:      "サブテスト内で作れば適合",
			body:      `tt.Run("a", func(tt T) { tt.Parallel(); world := InitTestWorld(); world.ECS.NewEntity() })`,
			wantFound: false,
		},
		{
			name: "並列でないサブテストの共有は許す",
			body: `world := InitTestWorld()
				tt.Run("a", func(tt T) { world.ECS.NewEntity() })`,
			wantFound: false,
		},
		{
			name:      "worldを引数に取る内側の関数リテラルは適合",
			body:      `tt.Run("a", func(tt T) { tt.Parallel(); use(func(world World) { world.ECS.NewEntity() }) })`,
			wantFound: false,
		},
		{
			name: "forループ内で作れば適合",
			body: `tt.Run("a", func(tt T) { tt.Parallel()
				for i := 0; i < 3; i++ { world := InitTestWorld(); world.ECS.NewEntity() } })`,
			wantFound: false,
		},
		{
			name:      "複合リテラルのフィールド名worldは適合",
			body:      `tt.Run("a", func(tt T) { tt.Parallel(); world := InitTestWorld(); _ = holder{world: world} })`,
			wantFound: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, info := parseWithStubTypes(t, tt.body)
			// スタブは自前の World 型なのでパッケージパスを見ない述語で判定する
			isWorld := func(obj types.Object) bool {
				named, ok := namedType(obj)
				return ok && named.Obj().Name() == "World"
			}
			got := sharedWorldViolations(f, info, isWorld)
			assert.Equal(t, tt.wantFound, len(got) > 0)
		})
	}
}

// parseWithStubTypes は検出器テスト用の自己完結ソースを型解決付きで組み立てる。
// 本物の testing や world を import せず、同名の形だけ持つスタブで代用する
func parseWithStubTypes(t *testing.T, body string) (*ast.File, *types.Info) {
	t.Helper()
	src := `package p
type ECS struct{}
func (ECS) NewEntity() {}
type World struct{ ECS ECS }
func InitTestWorld() World { return World{} }
type T struct{}
func (T) Run(string, func(T)) {}
func (T) Parallel() {}
func use(func(World)) {}
type holder struct{ world World }
func TestX(tt T) {
` + body + `
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x_test.go", src, 0)
	require.NoError(t, err)
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}}
	_, err = (&types.Config{}).Check("p", fset, []*ast.File{f}, info)
	require.NoError(t, err)
	return f, info
}

// sharedWorldViolations はファイルから「t.Parallel() する t.Run クロージャが、
// クロージャの外で定義された world 型の変数を参照している」位置を集める。
// クロージャ内で定義された変数や引数は定義位置がクロージャ範囲内なので除かれる
func sharedWorldViolations(f *ast.File, info *types.Info, isWorld func(types.Object) bool) []token.Pos {
	var violations []token.Pos
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := subtestFuncLit(n)
		if !ok || !callsParallel(lit) {
			return true
		}
		reported := map[types.Object]bool{}
		ast.Inspect(lit.Body, func(m ast.Node) bool {
			id, ok := m.(*ast.Ident)
			if !ok {
				return true
			}
			obj := info.Uses[id]
			if obj == nil || !isWorld(obj) || reported[obj] {
				return true
			}
			if obj.Pos() >= lit.Pos() && obj.Pos() <= lit.End() {
				return true
			}
			reported[obj] = true
			violations = append(violations, id.Pos())
			return true
		})
		return true
	})
	return violations
}

// isRuinsWorld は internal/world.World 型の変数かを返す
func isRuinsWorld(obj types.Object) bool {
	named, ok := namedType(obj)
	return ok && named.Obj().Name() == "World" &&
		named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == worldTypePath
}

// namedType は変数の型を、ポインタなら剥がして名前付き型として返す。
// 構造体フィールドは複合リテラルのフィールド名として Uses に載るが、
// 変数の共有ではないので対象にしない
func namedType(obj types.Object) (*types.Named, bool) {
	v, ok := obj.(*types.Var)
	if !ok || v.IsField() {
		return nil, false
	}
	typ := v.Type()
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}
	named, ok := typ.(*types.Named)
	return named, ok
}

// subtestFuncLit は t.Run(name, func(t *testing.T){...}) の関数リテラルを取り出す
func subtestFuncLit(n ast.Node) (*ast.FuncLit, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Run" {
		return nil, false
	}
	lit, ok := call.Args[1].(*ast.FuncLit)
	return lit, ok
}

// callsParallel はクロージャ直下に t.Parallel() 呼び出しがあるかを返す
func callsParallel(lit *ast.FuncLit) bool {
	for _, stmt := range lit.Body.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "Parallel" {
			return true
		}
	}
	return false
}

// raceEnabled は -race 付きでビルドされているかをビルド情報から返す。
// race 無効時は -race キー自体が載らない
func raceEnabled() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, s := range info.Settings {
		if s.Key == "-race" {
			return s.Value == "true"
		}
	}
	return false
}

// moduleRootForLint は go.mod のあるモジュールルートを返す
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
