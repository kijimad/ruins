package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
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
