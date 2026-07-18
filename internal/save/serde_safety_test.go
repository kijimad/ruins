package save

import (
	"reflect"
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSavedComponentsSerdeSafe は保存対象コンポーネントが ark-serde で往復可能な
// 形状であることを保証する規約テスト。
// ark-serde は JSON マーシャルのため、interface / chan / func / struct キー map は
// 復元できない。保存されるコンポーネント（skipComponents に無いもの）がこれらを
// 露出していないことを、実際に登録された全コンポーネント型に対して検査する。
// 違反したら json:"-" で除外するか、一時状態なら skipComponents に追加すること。
func TestSavedComponentsSerdeSafe(t *testing.T) {
	t.Parallel()

	// 全コンポーネント型を登録する
	c := &gc.Components{}
	world := ecs.NewWorld()
	require.NoError(t, c.InitializeComponents(world))

	// 保存対象外（skipComponents）を除外集合にする
	skip := map[reflect.Type]bool{}
	for _, comp := range skipComponents() {
		skip[comp.Type()] = true
	}

	for _, id := range ecs.ComponentIDs(world) {
		info, ok := ecs.ComponentInfo(world, id)
		if !ok || skip[info.Type] {
			continue
		}
		assertSerdeSafe(t, info.Type, info.Type.String(), map[reflect.Type]bool{})
	}
}

// assertSerdeSafe は型を再帰的に走査し、serde 非対応の要素があればテストを失敗させる。
// json:"-" タグの付いたフィールドと非公開フィールドは serde 対象外なので検査しない。
func assertSerdeSafe(t *testing.T, typ reflect.Type, path string, visited map[reflect.Type]bool) {
	t.Helper()
	if visited[typ] {
		return // 再帰型の無限ループを防ぐ
	}
	visited[typ] = true

	switch typ.Kind() {
	case reflect.Interface:
		assert.Failf(t, "serde非対応", "%s は interface で ark-serde が復元できない。json:\"-\" 除外か skipComponents に追加せよ", path)
	case reflect.Chan, reflect.Func:
		assert.Failf(t, "serde非対応", "%s は %s で serde 非対応", path, typ.Kind())
	case reflect.Pointer, reflect.Slice, reflect.Array:
		assertSerdeSafe(t, typ.Elem(), path+"[]", visited)
	case reflect.Map:
		if typ.Key().Kind() == reflect.Struct {
			assert.Failf(t, "serde非対応", "%s は struct キーの map で ark-serde が復元できない", path)
		}
		assertSerdeSafe(t, typ.Elem(), path+"{}", visited)
	case reflect.Struct:
		for f := range typ.Fields() {
			if f.PkgPath != "" { // 非公開フィールドは JSON 化されない
				continue
			}
			if strings.Split(f.Tag.Get("json"), ",")[0] == "-" { // json:"-" 除外
				continue
			}
			assertSerdeSafe(t, f.Type, path+"."+f.Name, visited)
		}
	default:
		// Bool/数値/String などのスカラー種別は JSON で往復可能なので検査不要
	}
}
