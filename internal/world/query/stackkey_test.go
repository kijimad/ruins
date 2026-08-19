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

func TestStackKeyOf_鮮度と品種でキーが決まる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 非腐敗品。段階は空文字
	apple := world.ECS.NewEntity()
	world.Components.RawID.Add(apple, &gc.RawID{ID: "apple"})
	key, ok := StackKeyOf(world, apple)
	require.True(t, ok)
	assert.Equal(t, "apple", key.RawID)
	assert.Equal(t, gc.FreshnessStage(""), key.FreshnessStage, "非腐敗品の段階は空文字")

	// 腐敗品。生成直後は fresh 段階
	meat := world.ECS.NewEntity()
	world.Components.RawID.Add(meat, &gc.RawID{ID: "meat"})
	world.Components.Perishable.Add(meat, &gc.Perishable{StageLength: consts.Turn(100)})
	mkey, mok := StackKeyOf(world, meat)
	require.True(t, mok)
	assert.Equal(t, "meat", mkey.RawID)
	assert.Equal(t, gc.FreshnessFresh, mkey.FreshnessStage, "腐敗品は現在の鮮度段階を持つ")
}

func TestSameStack_同一判定(t *testing.T) {
	t.Parallel()

	// 並列サブテストは world を共有しない。Ark の world は NewEntity/Add が構造変更なので、
	// 共有すると並行生成でデータ競合になる。各サブテストが自分の world を持つ
	newNonPerish := func(world w.World, id string) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		return e
	}
	newPerish := func(world w.World, id string, rot consts.Turn) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		world.Components.Perishable.Add(e, &gc.Perishable{StageLength: consts.Turn(100), RotAccrued: rot})
		return e
	}

	t.Run("非腐敗で同じ品種は束ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.True(t, SameStack(world, newNonPerish(world, "bolt"), newNonPerish(world, "bolt")))
	})
	t.Run("品種が違えば束ねない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.False(t, SameStack(world, newNonPerish(world, "bolt"), newNonPerish(world, "nut")))
	})
	t.Run("腐敗で同じ段階なら束ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.True(t, SameStack(world, newPerish(world, "apple", 0), newPerish(world, "apple", 50)), "どちらも fresh 段階")
	})
	t.Run("腐敗で段階が違えば束ねない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.False(t, SameStack(world, newPerish(world, "apple", 0), newPerish(world, "apple", 150)), "fresh と stale")
	})
	t.Run("腐敗と非腐敗は同じ品種でも束ねない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.False(t, SameStack(world, newNonPerish(world, "apple"), newPerish(world, "apple", 0)))
	})
}

// TestSameStack_装備の個体差 は、クラフトの乱数化などで性能が違う同名装備が束ねられないこと、
// 性能が完全一致なら束ねられることを固定する。個体差の指紋が StackKey に効く根拠
func TestSameStack_装備の個体差(t *testing.T) {
	t.Parallel()

	newSword := func(world w.World, damage int) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: "wooden_sword"})
		world.Components.Melee.Add(e, &gc.Melee{Accuracy: 80, Damage: damage})
		return e
	}

	t.Run("性能が同じ装備は束ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.True(t, SameStack(world, newSword(world, 10), newSword(world, 10)))
	})
	t.Run("性能が違う装備は同じ品種でも束ねない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.False(t, SameStack(world, newSword(world, 10), newSword(world, 15)))
	})
	t.Run("防具は防御力の違いで束ねない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		mk := func(defense int) ecs.Entity {
			e := world.ECS.NewEntity()
			world.Components.RawID.Add(e, &gc.RawID{ID: "leather_armor"})
			world.Components.Wearable.Add(e, &gc.Wearable{Defense: defense})
			return e
		}
		assert.False(t, SameStack(world, mk(5), mk(9)))
		assert.True(t, SameStack(world, mk(5), mk(5)))
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

func TestStackMembers_床は同じタイルの同種だけ束ねる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	onField := func(id string, x, y consts.Tile) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		world.Components.LocationOnField.Add(e, &gc.LocationOnField{})
		world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		return e
	}

	// タイル(2,3)に apple 3個
	apple := onField("apple", 2, 3)
	onField("apple", 2, 3)
	onField("apple", 2, 3)
	// 別タイルの同種、同タイルの別品種は束ねない
	onField("apple", 4, 3)
	onField("bolt", 2, 3)

	assert.Len(t, StackMembers(world, apple), 3, "同タイルの apple だけ束ねる")
	assert.Equal(t, 3, GetEntityCount(world, apple), "床でも同種の個数を導出する")
}

// TestStackMembers_所有者で分かれる は、同種でも所有者が違えば束ねないことを固定する。
// この分離が壊れると他人の在庫が個数に混入し、重量・表示・丸ごと移動の範囲が silent に壊れる
func TestStackMembers_所有者で分かれる(t *testing.T) {
	t.Parallel()

	t.Run("バックパックは所有者ごとに束ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		ownerA := world.ECS.NewEntity()
		ownerB := world.ECS.NewEntity()
		mk := func(owner ecs.Entity) ecs.Entity {
			e := world.ECS.NewEntity()
			world.Components.RawID.Add(e, &gc.RawID{ID: "iron"})
			world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: owner})
			return e
		}
		a1 := mk(ownerA)
		mk(ownerA)
		b1 := mk(ownerB)

		assert.Len(t, StackMembers(world, a1), 2, "A の束は A の2個だけ")
		assert.Equal(t, 2, GetEntityCount(world, a1))
		assert.Equal(t, 1, GetEntityCount(world, b1), "B の同種は混ざらない")
	})

	t.Run("収納は所有者ごとに束ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storageA := world.ECS.NewEntity()
		storageB := world.ECS.NewEntity()
		mk := func(storage ecs.Entity) ecs.Entity {
			e := world.ECS.NewEntity()
			world.Components.RawID.Add(e, &gc.RawID{ID: "iron"})
			world.Components.LocationInStorage.Add(e, &gc.LocationInStorage{Owner: storage})
			return e
		}
		a1 := mk(storageA)
		mk(storageA)
		mk(storageA)
		b1 := mk(storageB)

		assert.Len(t, StackMembers(world, a1), 3, "A の束は A の3個だけ")
		assert.Equal(t, 1, GetEntityCount(world, b1), "別収納の同種は混ざらない")
	})
}

// TestSameStack_同定キーの無い実体は束ねない は、RawID を持たない実体がスタックに
// なりえないことを固定する。raw 定義を経ない実行時生成の実体が混ざった任意の実体列を
// GroupStacks へ渡しても安全である根拠
func TestSameStack_同定キーの無い実体は束ねない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 死亡フェードアウトのエフェクトのような、raw 定義を経ない実行時生成の実体を模す
	effectA := world.ECS.NewEntity()
	effectB := world.ECS.NewEntity()
	_, ok := StackKeyOf(world, effectA)
	assert.False(t, ok, "同定キーが無い実体はスタックになりえない")
	assert.False(t, SameStack(world, effectA, effectB), "同定キーが無い別実体は束ねない")
	assert.False(t, SameStack(world, effectA, effectA), "スタックになりえない実体は自分自身とも束ねない")

	stacks := GroupStacks(world, []ecs.Entity{effectA, effectB})
	require.Len(t, stacks, 2, "それぞれ1個だけの束として並ぶ")
	assert.Equal(t, 1, stacks[0].Count)
	assert.Equal(t, 1, stacks[1].Count)
}

// TestBackpackStacks_収集と整列と束ねを1口で行う は、一覧口が所有者で絞り、
// 名前順に並べ、同一スタックを束ねた結果を返すことを固定する。一覧はこの口だけを使う
func TestBackpackStacks_収集と整列と束ねを1口で行う(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	owner := world.ECS.NewEntity()
	other := world.ECS.NewEntity()
	mk := func(o ecs.Entity, id string, name string) {
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: id})
		world.Components.Name.Add(e, &gc.Name{Name: name})
		world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: o})
	}
	// わざと名前の逆順で作る。Zebra 2個、Alpha 1個、他人の Alpha 1個
	mk(owner, "zebra", "Zebra")
	mk(owner, "zebra", "Zebra")
	mk(owner, "alpha", "Alpha")
	mk(other, "alpha", "Alpha")

	stacks := BackpackStacks(world, owner)

	require.Len(t, stacks, 2, "所有者の2品種だけが束になる")
	assert.Equal(t, "Alpha", world.Components.Name.Get(stacks[0].Rep).Name, "名前順で並ぶ")
	assert.Equal(t, 1, stacks[0].Count)
	assert.Equal(t, "Zebra", world.Components.Name.Get(stacks[1].Rep).Name)
	assert.Equal(t, 2, stacks[1].Count, "同一スタックは束ねられる")
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
