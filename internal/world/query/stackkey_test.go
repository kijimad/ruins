package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackKeyOf_鮮度と品種でキーが決まる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 非腐敗品。段階は空文字
	apple := world.ECS.NewEntity()
	world.Components.RawID.Add(apple, &gc.RawID{ID: "apple"})
	key := StackKeyOf(world, apple)
	assert.Equal(t, "apple", key.RawID)
	assert.Equal(t, gc.FreshnessStage(""), key.FreshnessStage, "非腐敗品の段階は空文字")

	// 腐敗品。生成直後は fresh 段階
	meat := world.ECS.NewEntity()
	world.Components.RawID.Add(meat, &gc.RawID{ID: "meat"})
	world.Components.Perishable.Add(meat, &gc.Perishable{StageLength: consts.Turn(100)})
	mkey := StackKeyOf(world, meat)
	assert.Equal(t, "meat", mkey.RawID)
	assert.Equal(t, gc.FreshnessFresh, mkey.FreshnessStage, "腐敗品は現在の鮮度段階を持つ")
}

func TestSameStack_同一判定(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	newNonPerish := func(id string) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		return e
	}
	newPerish := func(id string, rot consts.Turn) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		world.Components.Perishable.Add(e, &gc.Perishable{StageLength: consts.Turn(100), RotAccrued: rot})
		return e
	}

	t.Run("非腐敗で同じ品種は束ねる", func(t *testing.T) {
		t.Parallel()
		assert.True(t, SameStack(world, newNonPerish("bolt"), newNonPerish("bolt")))
	})
	t.Run("品種が違えば束ねない", func(t *testing.T) {
		t.Parallel()
		assert.False(t, SameStack(world, newNonPerish("bolt"), newNonPerish("nut")))
	})
	t.Run("腐敗で同じ段階なら束ねる", func(t *testing.T) {
		t.Parallel()
		assert.True(t, SameStack(world, newPerish("apple", 0), newPerish("apple", 50)), "どちらも fresh 段階")
	})
	t.Run("腐敗で段階が違えば束ねない", func(t *testing.T) {
		t.Parallel()
		assert.False(t, SameStack(world, newPerish("apple", 0), newPerish("apple", 150)), "fresh と stale")
	})
	t.Run("腐敗と非腐敗は同じ品種でも束ねない", func(t *testing.T) {
		t.Parallel()
		assert.False(t, SameStack(world, newNonPerish("apple"), newPerish("apple", 0)))
	})
}

func TestStackMembers_装備品や未配置は単独スタックになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 装備スロットのアイテムは束ねず単独を返す。個数は常に1になる
	equipped := world.ECS.NewEntity()
	world.Components.RawID.Add(equipped, &gc.RawID{ID: "sword"})
	world.Components.LocationEquipped.Add(equipped, &gc.LocationEquipped{Owner: world.ECS.NewEntity(), EquipmentSlot: gc.SlotWeapon1})
	assert.Equal(t, []ecs.Entity{equipped}, StackMembers(world, equipped))
	assert.Equal(t, 1, GetEntityCount(world, equipped))

	// 位置未設定のアイテムも単独スタック
	bare := world.ECS.NewEntity()
	world.Components.RawID.Add(bare, &gc.RawID{ID: "gem"})
	assert.Equal(t, []ecs.Entity{bare}, StackMembers(world, bare))
	assert.Equal(t, 1, GetEntityCount(world, bare))
}

func TestGroupStacks_同一性キーで束ね初出順を保つ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mk := func(id string) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		return e
	}
	bolt1, nut, bolt2, bolt3 := mk("bolt"), mk("nut"), mk("bolt"), mk("bolt")
	entities := []ecs.Entity{bolt1, nut, bolt2, bolt3}

	stacks := GroupStacks(world, entities)

	require.Len(t, stacks, 2, "bolt と nut の2束")
	assert.Equal(t, bolt1, stacks[0].Rep, "初出の bolt が代表")
	assert.Equal(t, 3, stacks[0].Count, "bolt は3個")
	assert.Equal(t, []ecs.Entity{bolt1, bolt2, bolt3}, stacks[0].Members)
	assert.Equal(t, nut, stacks[1].Rep, "2束目は nut")
	assert.Equal(t, 1, stacks[1].Count)

	assert.Equal(t, 3, StackCountOf(world, bolt2, entities), "候補の中で bolt と同一スタックは3個")
	assert.Equal(t, 1, StackCountOf(world, nut, entities))
}
