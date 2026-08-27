package testutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoSharedWorldInParallelSubtests は並列サブテストでの world 共有を禁止する。
// Ark の world は並行安全でないため、t.Parallel() するサブテストが外側で作った world に
// 触れるとデータ競合になる。world は各サブテストの中で作る。
// world という変数名の慣習に依存した構文一致だが、InitTestWorld の戻り値は
// 一貫してこの名前で受けており実用上十分に拾える。
func TestNoSharedWorldInParallelSubtests(t *testing.T) {
	t.Parallel()
	root := moduleRootForLint(t)
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
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			for _, pos := range sharedWorldViolations(fn.Body) {
				p := fset.Position(pos)
				rel, relErr := filepath.Rel(root, p.Filename)
				if relErr != nil {
					rel = p.Filename
				}
				found = append(found, rel+":"+strconv.Itoa(p.Line))
			}
			return false
		})
		return nil
	})
	require.NoError(t, err)
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
			body: `world := testutil.InitTestWorld(t)
				t.Run("a", func(t *testing.T) { t.Parallel(); world.ECS.NewEntity() })`,
			wantFound: true,
		},
		{
			name:      "サブテスト内で作れば適合",
			body:      `t.Run("a", func(t *testing.T) { t.Parallel(); world := testutil.InitTestWorld(t); world.ECS.NewEntity() })`,
			wantFound: false,
		},
		{
			name: "並列でないサブテストの共有は許す",
			body: `world := testutil.InitTestWorld(t)
				t.Run("a", func(t *testing.T) { world.ECS.NewEntity() })`,
			wantFound: false,
		},
		{
			name:      "worldを引数に取る内側の関数リテラルは適合",
			body:      `t.Run("a", func(t *testing.T) { t.Parallel(); use(func(world w.World) { world.ECS.NewEntity() }) })`,
			wantFound: false,
		},
		{
			name: "forループ内で作れば適合",
			body: `t.Run("a", func(t *testing.T) { t.Parallel()
				for range 3 { world := testutil.InitTestWorld(t); world.ECS.NewEntity() } })`,
			wantFound: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := "package p\nfunc TestX(t *testing.T) {\n" + tt.body + "\n}\n"
			f, err := parser.ParseFile(token.NewFileSet(), "x_test.go", src, 0)
			require.NoError(t, err)
			fn, ok := f.Decls[0].(*ast.FuncDecl)
			require.True(t, ok)
			got := sharedWorldViolations(fn.Body)
			assert.Equal(t, tt.wantFound, len(got) > 0)
		})
	}
}

// sharedWorldViolations は関数本体から「t.Parallel() する t.Run クロージャが、
// クロージャの外で定義された world を参照している」位置を集める
func sharedWorldViolations(body *ast.BlockStmt) []token.Pos {
	var violations []token.Pos
	ast.Inspect(body, func(n ast.Node) bool {
		lit, ok := subtestFuncLit(n)
		if !ok {
			return true
		}
		if !callsParallel(lit) {
			return true
		}
		if pos, ok := usesOuterWorld(lit); ok {
			violations = append(violations, pos)
		}
		return true
	})
	return violations
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

// usesOuterWorld はクロージャが自ら定義していない world を参照している位置を返す。
// クロージャ内の world := 以降の参照と、world を引数に持つ内側の関数リテラルの中は
// ローカルなので外側の共有ではない
func usesOuterWorld(lit *ast.FuncLit) (token.Pos, bool) {
	return outerWorldUseInBlock(lit.Body, paramsDefineWorld(lit))
}

// paramsDefineWorld は関数リテラルの引数に world があるかを返す
func paramsDefineWorld(lit *ast.FuncLit) bool {
	if lit.Type.Params == nil {
		return false
	}
	for _, field := range lit.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == "world" {
				return true
			}
		}
	}
	return false
}

// outerWorldUseInBlock はブロックの文を順に見て、未定義のまま world を参照する位置を返す。
// world := の定義文から先はローカル定義済みなので見ない
func outerWorldUseInBlock(block *ast.BlockStmt, defined bool) (token.Pos, bool) {
	if defined {
		return token.NoPos, false
	}
	for _, stmt := range block.List {
		if pos, ok := outerWorldUseInNode(stmt); ok {
			return pos, true
		}
		if definesWorld(stmt) {
			return token.NoPos, false
		}
	}
	return token.NoPos, false
}

// definesWorld は world := を含む定義文かを返す
func definesWorld(stmt ast.Stmt) bool {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE {
		return false
	}
	for _, lhs := range assign.Lhs {
		if id, ok := lhs.(*ast.Ident); ok && id.Name == "world" {
			return true
		}
	}
	return false
}

// outerWorldUseInNode はノード内の world 参照位置を返す。定義文の左辺は参照に数えず、
// 内側の関数リテラルはそのスコープを考慮して再帰で見る
func outerWorldUseInNode(root ast.Node) (token.Pos, bool) {
	var pos token.Pos
	var found bool
	ast.Inspect(root, func(n ast.Node) bool {
		if found {
			return false
		}
		switch node := n.(type) {
		case *ast.FuncLit:
			if p, ok := outerWorldUseInBlock(node.Body, paramsDefineWorld(node)); ok {
				pos, found = p, true
			}
			return false
		case *ast.BlockStmt:
			// for や if の本体もブロックとして逐次に見る。中の world := 以降はローカル
			if p, ok := outerWorldUseInBlock(node, false); ok {
				pos, found = p, true
			}
			return false
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				for _, rhs := range node.Rhs {
					if p, ok := outerWorldUseInNode(rhs); ok {
						pos, found = p, true
						break
					}
				}
				return false
			}
		case *ast.Ident:
			if node.Name == "world" {
				pos, found = node.Pos(), true
				return false
			}
		}
		return true
	})
	return pos, found
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
