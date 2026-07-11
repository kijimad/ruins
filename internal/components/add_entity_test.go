package components

import (
	"reflect"
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddEntity_AllFields は EntitySpec の全ポインタフィールドが AddEntity で
// コンポーネントとして付与されることを検証する。
// フィールドを追加して AddEntity への追記を忘れると、そのフィールドだけを設定した
// エンティティが空になり、このテストが該当フィールドで落ちる（網羅性の保証）。
func TestAddEntity_AllFields(t *testing.T) {
	t.Parallel()
	specType := reflect.TypeFor[EntitySpec]()
	for i := range specType.NumField() {
		field := specType.Field(i)
		if field.Type.Kind() != reflect.Pointer {
			continue
		}
		t.Run(field.Name, func(t *testing.T) {
			t.Parallel()
			world := ecs.NewWorld()
			c := &Components{}
			require.NoError(t, c.InitializeComponents(world))

			// 対象フィールドだけを非nilにする
			spec := EntitySpec{}
			fv := reflect.ValueOf(&spec).Elem().Field(i)
			if elem := field.Type.Elem(); elem.Kind() == reflect.Interface {
				// *LocationType は具体型を割り当てる
				fv.Set(reflect.ValueOf(concreteForInterfaceField(field.Name)))
			} else {
				fv.Set(reflect.New(elem))
			}

			entity := c.AddEntity(world, &spec)
			assert.Positive(t, countComponents(world, entity),
				"%s が AddEntity で処理されていない（コンポーネントが付与されていない）", field.Name)
		})
	}
}

// countComponents はエンティティが保持する登録済みコンポーネントの数を返す
func countComponents(world *ecs.World, e ecs.Entity) int {
	u := world.Unsafe()
	n := 0
	for _, id := range ecs.ComponentIDs(world) {
		if u.Has(e, id) {
			n++
		}
	}
	return n
}

// concreteForInterfaceField は interfaceフィールド（*LocationType）に
// 割り当てる具体値のポインタを返す
func concreteForInterfaceField(fieldName string) any {
	if fieldName == "LocationType" {
		l := LocationType(LocationOnField{})
		return &l
	}
	panic("未知のinterfaceフィールド: " + fieldName)
}
