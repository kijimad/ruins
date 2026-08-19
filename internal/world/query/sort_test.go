package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				item1 := world.ECS.NewEntity()
				world.Components.Name.Add(item1, &gc.Name{Name: "Zebra Item"})

				item2 := world.ECS.NewEntity()
				world.Components.Name.Add(item2, &gc.Name{Name: "Alpha Item"})

				item3 := world.ECS.NewEntity()
				world.Components.Name.Add(item3, &gc.Name{Name: "Beta Item"})

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
				item1 := world.ECS.NewEntity()
				world.Components.Name.Add(item1, &gc.Name{Name: "剣"})

				item2 := world.ECS.NewEntity()
				world.Components.Name.Add(item2, &gc.Name{Name: "盾"})

				item3 := world.ECS.NewEntity()
				world.Components.Name.Add(item3, &gc.Name{Name: "鎧"})

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
					if world.Components.Name.Has(entity) {
						name := world.Components.Name.Get(entity)
						assert.Equal(t, tt.expected[i], name.Name)
					}
				}
			}
		})
	}
}

// TestSortEntities_同名は鮮度で決定的に並び削除後も入れ替わらない は、同名の別スタックが
// ECS の走査順でなく鮮度で決定的に並ぶこと、先頭スタックを消しても残りが入れ替わらないことを固定する。
// これがないとドロップの swap-remove で並びが崩れる。
func TestSortEntities_同名は鮮度で決定的に並び削除後も入れ替わらない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mk := func(rot consts.Turn) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.Name.Add(e, &gc.Name{Name: "Biscuit"})
		world.Components.RawID.Add(e, &gc.RawID{ID: "biscuit"})
		world.Components.Perishable.Add(e, &gc.Perishable{StageLength: 30000, RotAccrued: rot})
		return e
	}
	// わざと腐敗→新鮮→劣化の順で作る。走査順に依存しないことを見る
	mk(70000)
	fresh := mk(0)
	mk(40000)

	stages := func() []gc.FreshnessStage {
		var all []ecs.Entity
		q := ecs.NewFilter1[gc.Perishable](world.ECS).Query()
		for q.Next() {
			all = append(all, q.Entity())
		}
		sorted := SortEntities(world, all)
		out := make([]gc.FreshnessStage, 0, len(sorted))
		for _, e := range sorted {
			s, _ := FreshnessStageOf(world, e)
			out = append(out, s)
		}
		return out
	}

	assert.Equal(t, []gc.FreshnessStage{gc.FreshnessFresh, gc.FreshnessStale, gc.FreshnessRotten}, stages(),
		"作成順に依らず新鮮→劣化→腐敗の固定順")

	// 先頭(新鮮)を削除。Ark の swap-remove で格納順が変わる状況を作る
	world.ECS.RemoveEntity(fresh)
	assert.Equal(t, []gc.FreshnessStage{gc.FreshnessStale, gc.FreshnessRotten}, stages(),
		"先頭を捨てても残りは入れ替わらない")
}

// TestSortEntities_同名同鮮度は装備指紋で決定的に並ぶ は、性能違いの同名武器が
// 作成順に依らず同じ並びになることを固定する。指紋の副キーが無いと走査順に落ち、
// ドロップ等の swap-remove で並びが入れ替わる
func TestSortEntities_同名同鮮度は装備指紋で決定的に並ぶ(t *testing.T) {
	t.Parallel()

	damages := func(world w.World, order []int) []int {
		entities := make([]ecs.Entity, 0, len(order))
		for _, damage := range order {
			e := world.ECS.NewEntity()
			world.Components.Name.Add(e, &gc.Name{Name: "Sword"})
			world.Components.RawID.Add(e, &gc.RawID{ID: "sword"})
			world.Components.Melee.Add(e, &gc.Melee{Accuracy: 80, Damage: damage})
			entities = append(entities, e)
		}
		sorted := SortEntities(world, entities)
		out := make([]int, 0, len(sorted))
		for _, e := range sorted {
			out = append(out, world.Components.Melee.Get(e).Damage)
		}
		return out
	}

	worldA := testutil.InitTestWorld(t)
	worldB := testutil.InitTestWorld(t)
	// 作成順を逆にしても同じ並びになる
	assert.Equal(t, damages(worldA, []int{5, 12, 9}), damages(worldB, []int{9, 12, 5}),
		"性能違いの同名武器は作成順に依らず決定的に並ぶ")
}

func TestSortEntitiesWithMixedComponents(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// Nameコンポーネントを持つエンティティと持たないエンティティの混在
	entity1 := world.ECS.NewEntity()
	world.Components.Name.Add(entity1, &gc.Name{Name: "Charlie"})

	entity2 := world.ECS.NewEntity()
	// Nameコンポーネントなし

	entity3 := world.ECS.NewEntity()
	world.Components.Name.Add(entity3, &gc.Name{Name: "Alice"})

	entity4 := world.ECS.NewEntity()
	// Nameコンポーネントなし

	entity5 := world.ECS.NewEntity()
	world.Components.Name.Add(entity5, &gc.Name{Name: "Bob"})

	entities := []ecs.Entity{entity1, entity2, entity3, entity4, entity5}

	// ソート実行
	sorted := SortEntities(world, entities)

	// Nameコンポーネントを持つエンティティのみがソートされる
	require.Len(t, sorted, 3, "Nameコンポーネントを持つエンティティのみが返されるべき")

	// ソート順の確認
	name1 := world.Components.Name.Get(sorted[0])
	name2 := world.Components.Name.Get(sorted[1])
	name3 := world.Components.Name.Get(sorted[2])

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
