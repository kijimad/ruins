package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsChangedSystem_APClamp(t *testing.T) {
	t.Parallel()

	t.Run("現在APが最大APを超えている場合は切り詰められる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		// 現在APを非常に高い値に設定（通常ではありえない値）
		turnBased := world.Components.TurnBased.Get(player)
		turnBased.AP.Current = 9999
		turnBased.AP.Max = 9999

		// StatsChangedフラグを立てる
		// SpawnPlayer が初期装備でStatsChangedを立てるので、冪等に設定する
		require.NoError(t, gc.Upsert(world.ECS, world.Components.StatsChanged, player, &gc.StatsChanged{}))

		// システム実行
		sys := &StatsChangedSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		// APが正しく切り詰められていることを確認
		turnBased = world.Components.TurnBased.Get(player)
		assert.Equal(t, turnBased.AP.Max, turnBased.AP.Current, "現在APは最大APに切り詰められるべき")
		assert.Less(t, turnBased.AP.Current, 9999, "APが正しく再計算されるべき")
	})
}

// TestStatsChangedSystem_装備した光源をプレイヤーへ転写する は携行光源の中核を固定する。
// トーチを装備するとプレイヤーの LightSource が点灯し半径が写り、外すと消灯する。
func TestStatsChangedSystem_装備した光源をプレイヤーへ転写する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	sys := &StatsChangedSystem{}

	// SpawnPlayer は松明を初期装備し StatsChanged を立てる。処理すると光源が点灯する
	require.NoError(t, sys.Update(world))
	require.True(t, world.Components.LightSource.Has(player))
	ls := world.Components.LightSource.Get(player)
	assert.True(t, ls.Enabled, "松明を装備しているので点灯する")
	assert.Equal(t, 6, int(ls.Radius), "松明の半径がプレイヤーへ写る")

	// 松明を外すと光源が消灯する
	require.NoError(t, lifecycle.UnequipAll(world, player))
	require.NoError(t, sys.Update(world))
	assert.False(t, world.Components.LightSource.Get(player).Enabled, "外すと消灯する")
}

func TestMaxHP(t *testing.T) {
	t.Parallel()
	t.Run("calculate max HP with base stats", func(t *testing.T) {
		t.Parallel()
		abils := &gc.Abilities{
			Vitality: gc.Ability{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Strength: gc.Ability{
				Base:     5,
				Modifier: 0,
				Total:    5,
			},
			Sensation: gc.Ability{
				Base:     3,
				Modifier: 0,
				Total:    3,
			},
		}
		result := maxHP(abils)
		// 30 + (10*8 + 5 + 3) = 30 + 88 = 118
		expected := 118
		assert.Equal(t, expected, result, "maxHPの計算が正しくない")
	})

	t.Run("calculate max HP with level bonus", func(t *testing.T) {
		t.Parallel()
		abils := &gc.Abilities{
			Vitality: gc.Ability{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Strength: gc.Ability{
				Base:     5,
				Modifier: 0,
				Total:    5,
			},
			Sensation: gc.Ability{
				Base:     3,
				Modifier: 0,
				Total:    3,
			},
		}
		result := maxHP(abils)
		// 30 + (10*8 + 5 + 3) = 30 + 88 = 118
		expected := 118
		assert.Equal(t, expected, result, "レベルボーナス込みのmaxHPの計算が正しくない")
	})

	t.Run("calculate max HP with high stats", func(t *testing.T) {
		t.Parallel()
		abils := &gc.Abilities{
			Vitality: gc.Ability{
				Base:     20,
				Modifier: 5,
				Total:    25,
			},
			Strength: gc.Ability{
				Base:     15,
				Modifier: 3,
				Total:    18,
			},
			Sensation: gc.Ability{
				Base:     10,
				Modifier: 2,
				Total:    12,
			},
		}
		result := maxHP(abils)
		// 30 + (25*8 + 18 + 12) = 30 + 230 = 260
		expected := 260
		assert.Equal(t, expected, result, "高ステータスでのmaxHPの計算が正しくない")
	})
}

func TestStatsChangedAPRecalculation(t *testing.T) {
	t.Parallel()

	t.Run("装備変更でAPが再計算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.Abilities.Add(player, &gc.Abilities{
			Vitality:  gc.Ability{Base: 10, Total: 10},
			Strength:  gc.Ability{Base: 5, Total: 5},
			Sensation: gc.Ability{Base: 5, Total: 5},
			Dexterity: gc.Ability{Base: 5, Total: 5},
			Agility:   gc.Ability{Base: 5, Total: 5},
			Defense:   gc.Ability{Base: 0, Total: 0},
		})

		// 初期APを計算
		// AP = 100 + (素早さ * 3) + (器用さ * 1) = 100 + (5*3) + 5 = 120
		initialAP, err := query.CalculateMaxActionPoints(world, player)
		require.NoError(t, err)

		world.Components.HP.Add(player, &gc.HP{Current: 100, Max: 100})
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{
			AP: gc.IntPool{Current: initialAP, Max: initialAP},
		})

		// 素早さを上げる装備を作成
		equipment := world.ECS.NewEntity()
		world.Components.Name.Add(equipment, &gc.Name{Name: "素早さの指輪"})
		world.Components.Wearable.Add(equipment, &gc.Wearable{
			EquipmentCategory: gc.EquipmentJewelry,
			EquipBonus: gc.EquipBonus{
				Agility: 10, // 素早さ+10
			},
		})

		// 装備を装着（StatsChangedフラグが立つ）
		lifecycle.MoveToEquip(world, equipment, player, gc.SlotJewelry)

		// StatsChangedSystemを実行
		sys := &StatsChangedSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		// APが再計算されていることを確認
		turnBased := world.Components.TurnBased.Get(player)
		assert.Greater(t, turnBased.AP.Max, initialAP, "装備追加でAP.Maxが増加するべき")

		// 装備を外す（StatsChangedフラグが再度立つ）
		require.NoError(t, lifecycle.MoveToBackpack(world, equipment, player))

		// StatsChangedフラグが立っているか確認
		require.True(t, world.Components.StatsChanged.Has(player), "装備を外した後、StatsChangedフラグが立っているべき")

		// StatsChangedSystemを実行
		err = sys.Update(world)
		require.NoError(t, err)

		// StatsChangedフラグが削除されたか確認
		assert.False(t, world.Components.StatsChanged.Has(player), "StatsChangedSystemの実行後、フラグが削除されるべき")

		// APが元に戻っていることを確認
		turnBased = world.Components.TurnBased.Get(player)
		abils := world.Components.Abilities.Get(player)
		t.Logf("装備削除後: Agility.Total=%d, AP.Max=%d, 期待AP=%d", abils.Agility.Total, turnBased.AP.Max, initialAP)
		assert.Equal(t, initialAP, turnBased.AP.Max, "装備削除でAP.Maxが元に戻るべき")
	})

	t.Run("装備変更でHP/SPも再計算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		abils := &gc.Abilities{
			Vitality:  gc.Ability{Base: 10, Total: 10},
			Strength:  gc.Ability{Base: 5, Total: 5},
			Sensation: gc.Ability{Base: 5, Total: 5},
			Dexterity: gc.Ability{Base: 5, Total: 5},
			Agility:   gc.Ability{Base: 5, Total: 5},
			Defense:   gc.Ability{Base: 0, Total: 0},
		}
		world.Components.Abilities.Add(player, abils)

		// 初期HPを計算式から算出
		// maxHP: 30 + (体力*8 + 力 + 感覚) = 30 + (10*8 + 5 + 5) = 30 + 90 = 120
		initialHP := maxHP(abils)

		world.Components.HP.Add(player, &gc.HP{Current: initialHP, Max: initialHP})
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{
			AP: gc.IntPool{Current: 100, Max: 100},
		})

		// 体力を上げる装備を作成
		equipment := world.ECS.NewEntity()
		world.Components.Name.Add(equipment, &gc.Name{Name: "体力の鎧"})
		world.Components.Wearable.Add(equipment, &gc.Wearable{
			EquipmentCategory: gc.EquipmentTorso,
			EquipBonus: gc.EquipBonus{
				Vitality: 10, // 体力+10
			},
		})

		// 装備を装着
		lifecycle.MoveToEquip(world, equipment, player, gc.SlotTorso)

		// StatsChangedSystemを実行
		sys := &StatsChangedSystem{}
		err := sys.Update(world)
		require.NoError(t, err)

		// HPが再計算されていることを確認
		// 体力10→20で: HP = 30 + (20*8 + 5 + 5) = 200
		hp := world.Components.HP.Get(player)
		assert.Greater(t, hp.Max, initialHP, "装備追加でHP.Maxが増加するべき")
	})

}

func TestStatsChangedSystem_同アーキタイプ複数体でHPを取り違えない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 同一コンポーネント構成のエンティティを並べる。Skills を持つので Update が
	// CharModifiers を Add し、アーキタイプ移動の構造変更が起きる。構造変更後に
	// abils を読むと別エンティティの値を拾いうるため、各自の Vitality で HP が
	// 決まることを固定する
	mk := func(vit int) ecs.Entity {
		e := world.ECS.NewEntity()
		world.Components.Abilities.Add(e, &gc.Abilities{Vitality: gc.Ability{Base: vit, Total: vit}})
		world.Components.Skills.Add(e, gc.NewSkills())
		world.Components.HP.Add(e, &gc.HP{Current: 1, Max: 1})
		world.Components.StatsChanged.Add(e, &gc.StatsChanged{})
		return e
	}
	e1 := mk(10)
	e2 := mk(20)

	sys := &StatsChangedSystem{}
	require.NoError(t, sys.Update(world))

	assert.Equal(t, 30+10*8, world.Components.HP.Get(e1).Max, "e1 は自分の Vitality で HP が決まる")
	assert.Equal(t, 30+20*8, world.Components.HP.Get(e2).Max, "e2 は自分の Vitality で HP が決まる")
}
