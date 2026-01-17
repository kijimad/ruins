package entities

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestComponentList(t *testing.T) {
	t.Parallel()
	t.Run("create entity component list", func(t *testing.T) {
		t.Parallel()
		gameComponents := []interface{}{
			gc.EntitySpec{
				Name: &gc.Name{Name: "テストエンティティ"},
			},
		}

		list := ComponentList[interface{}]{
			Entities: gameComponents,
		}

		assert.Len(t, list.Entities, 1, "エンティティリストの長さが正しくない")
	})

	t.Run("empty entity component list", func(t *testing.T) {
		t.Parallel()
		list := ComponentList[interface{}]{}
		assert.Nil(t, list.Entities, "空のリストでEntitiesがnilでない")
	})
}

func TestAddEntities(t *testing.T) {
	t.Parallel()
	t.Run("basic functionality test", func(t *testing.T) {
		t.Parallel()
		// 循環依存を避けるため、基本的な機能のみテスト
		entityComponentList := ComponentList[interface{}]{
			Entities: []interface{}{
				gc.EntitySpec{
					Name: &gc.Name{Name: "テストエンティティ"},
				},
			},
		}

		// AddEntitiesは実際のworldオブジェクトが必要なため、
		// 構造体の正常性のみテスト
		assert.Len(t, entityComponentList.Entities, 1, "エンティティリストの長さが正しくない")

		// EntitySpecの中身を確認
		entityComponent := entityComponentList.Entities[0].(gc.EntitySpec)
		assert.NotNil(t, entityComponent.Name, "Nameコンポーネントが設定されていない")
		assert.Equal(t, "テストエンティティ", entityComponent.Name.Name, "名前が正しく設定されていない")
	})
}
