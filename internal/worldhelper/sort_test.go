package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestSortEntities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entities func(w w.World) []ecs.Entity
		expected []string
	}{
		{
			name: "アイテムのソート",
			entities: func(world w.World) []ecs.Entity {
				item1 := world.Manager.NewEntity()
				item1.AddComponent(world.Components.Name, &gc.Name{Name: "Zebra Item"})

				item2 := world.Manager.NewEntity()
				item2.AddComponent(world.Components.Name, &gc.Name{Name: "Alpha Item"})

				item3 := world.Manager.NewEntity()
				item3.AddComponent(world.Components.Name, &gc.Name{Name: "Beta Item"})

				return []ecs.Entity{item1, item2, item3}
			},
			expected: []string{"Alpha Item", "Beta Item", "Zebra Item"},
		},
		{
			name: "空のリスト",
			entities: func(_ w.World) []ecs.Entity {
				return []ecs.Entity{}
			},
			expected: []string{},
		},
		{
			name: "日本語名のソート",
			entities: func(world w.World) []ecs.Entity {
				item1 := world.Manager.NewEntity()
				item1.AddComponent(world.Components.Name, &gc.Name{Name: "剣"})

				item2 := world.Manager.NewEntity()
				item2.AddComponent(world.Components.Name, &gc.Name{Name: "盾"})

				item3 := world.Manager.NewEntity()
				item3.AddComponent(world.Components.Name, &gc.Name{Name: "鎧"})

				return []ecs.Entity{item1, item2, item3}
			},
			expected: []string{"剣", "盾", "鎧"}, // UTF-8コード順
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// 各テストケースで新しいworldを作成
			world := testutil.InitTestWorld(t)

			entities := tt.entities(world)
			sorted := SortEntities(world, entities)

			// ソート結果の検証
			assert.Len(t, sorted, len(tt.expected))
			for i, entity := range sorted {
				if len(tt.expected) > 0 {
					if entity.HasComponent(world.Components.Name) {
						name := world.Components.Name.Get(entity).(*gc.Name)
						assert.Equal(t, tt.expected[i], name.Name)
					}
				}
			}
		})
	}
}

func TestSortEntitiesWithMixedComponents(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// Nameコンポーネントを持つエンティティと持たないエンティティの混在
	entity1 := world.Manager.NewEntity()
	entity1.AddComponent(world.Components.Name, &gc.Name{Name: "Charlie"})

	entity2 := world.Manager.NewEntity()
	// Nameコンポーネントなし

	entity3 := world.Manager.NewEntity()
	entity3.AddComponent(world.Components.Name, &gc.Name{Name: "Alice"})

	entity4 := world.Manager.NewEntity()
	// Nameコンポーネントなし

	entity5 := world.Manager.NewEntity()
	entity5.AddComponent(world.Components.Name, &gc.Name{Name: "Bob"})

	entities := []ecs.Entity{entity1, entity2, entity3, entity4, entity5}

	// ソート実行
	sorted := SortEntities(world, entities)

	// Nameコンポーネントを持つエンティティのみがソートされる
	require.Len(t, sorted, 3, "Nameコンポーネントを持つエンティティのみが返されるべき")

	// ソート順の確認
	name1 := world.Components.Name.Get(sorted[0]).(*gc.Name)
	name2 := world.Components.Name.Get(sorted[1]).(*gc.Name)
	name3 := world.Components.Name.Get(sorted[2]).(*gc.Name)

	assert.Equal(t, "Alice", name1.Name)
	assert.Equal(t, "Bob", name2.Name)
	assert.Equal(t, "Charlie", name3.Name)
}

func TestSortEntitiesEmptyAndNilCases(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 空のリストのテスト
	emptyList := []ecs.Entity{}
	sortedEmpty := SortEntities(world, emptyList)
	assert.Empty(t, sortedEmpty, "空のリストは空のまま返されるべき")

	// nilリストのテスト（もし実装で対応する場合）
	var nilList []ecs.Entity
	sortedNil := SortEntities(world, nilList)
	assert.Empty(t, sortedNil, "nilリストは空のリストとして返されるべき")
}
