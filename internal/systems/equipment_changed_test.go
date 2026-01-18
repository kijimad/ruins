package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxHP(t *testing.T) {
	t.Parallel()
	t.Run("calculate max HP with base stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Strength: gc.Attribute{
				Base:     5,
				Modifier: 0,
				Total:    5,
			},
			Sensation: gc.Attribute{
				Base:     3,
				Modifier: 0,
				Total:    3,
			},
		}
		result := maxHP(attrs)
		// 30 + (10*8 + 5 + 3) = 30 + 88 = 118
		expected := 118
		assert.Equal(t, expected, result, "maxHPの計算が正しくない")
	})

	t.Run("calculate max HP with level bonus", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Strength: gc.Attribute{
				Base:     5,
				Modifier: 0,
				Total:    5,
			},
			Sensation: gc.Attribute{
				Base:     3,
				Modifier: 0,
				Total:    3,
			},
		}
		result := maxHP(attrs)
		// 30 + (10*8 + 5 + 3) = 30 + 88 = 118
		expected := 118
		assert.Equal(t, expected, result, "レベルボーナス込みのmaxHPの計算が正しくない")
	})

	t.Run("calculate max HP with high stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     20,
				Modifier: 5,
				Total:    25,
			},
			Strength: gc.Attribute{
				Base:     15,
				Modifier: 3,
				Total:    18,
			},
			Sensation: gc.Attribute{
				Base:     10,
				Modifier: 2,
				Total:    12,
			},
		}
		result := maxHP(attrs)
		// 30 + (25*8 + 18 + 12) = 30 + 230 = 260
		expected := 260
		assert.Equal(t, expected, result, "高ステータスでのmaxHPの計算が正しくない")
	})
}

func TestMaxSP(t *testing.T) {
	t.Parallel()
	t.Run("calculate max SP with base stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Dexterity: gc.Attribute{
				Base:     8,
				Modifier: 0,
				Total:    8,
			},
			Agility: gc.Attribute{
				Base:     6,
				Modifier: 0,
				Total:    6,
			},
		}
		result := maxSP(attrs)
		// 10*2 + 8 + 6 = 20 + 8 + 6 = 34
		expected := 34
		assert.Equal(t, expected, result, "maxSPの計算が正しくない")
	})

	t.Run("calculate max SP with level bonus", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Dexterity: gc.Attribute{
				Base:     8,
				Modifier: 0,
				Total:    8,
			},
			Agility: gc.Attribute{
				Base:     6,
				Modifier: 0,
				Total:    6,
			},
		}
		result := maxSP(attrs)
		// 10*2 + 8 + 6 = 20 + 8 + 6 = 34
		expected := 34
		assert.Equal(t, expected, result, "maxSPの計算が正しくない")
	})

	t.Run("calculate max SP with high stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     20,
				Modifier: 5,
				Total:    25,
			},
			Dexterity: gc.Attribute{
				Base:     15,
				Modifier: 3,
				Total:    18,
			},
			Agility: gc.Attribute{
				Base:     12,
				Modifier: 2,
				Total:    14,
			},
		}
		result := maxSP(attrs)
		// 25*2 + 18 + 14 = 50 + 18 + 14 = 82
		expected := 82
		assert.Equal(t, expected, result, "高ステータスでのmaxSPの計算が正しくない")
	})
}

func TestEquipmentChangedAPRecalculation(t *testing.T) {
	t.Parallel()

	t.Run("装備変更でAPが再計算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.Attributes, &gc.Attributes{
			Vitality:  gc.Attribute{Base: 10, Total: 10},
			Strength:  gc.Attribute{Base: 5, Total: 5},
			Sensation: gc.Attribute{Base: 5, Total: 5},
			Dexterity: gc.Attribute{Base: 5, Total: 5},
			Agility:   gc.Attribute{Base: 5, Total: 5},
			Defense:   gc.Attribute{Base: 0, Total: 0},
		})

		// 初期APを計算
		// AP = 100 + (素早さ * 3) + (器用さ * 1) = 100 + (5*3) + 5 = 120
		initialAP, err := worldhelper.CalculateMaxActionPoints(world, player)
		require.NoError(t, err)

		player.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 100, Max: 100},
			SP: gc.Pool{Current: 50, Max: 50},
		})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.Pool{Current: initialAP, Max: initialAP},
		})

		// 素早さを上げる装備を作成
		equipment := world.Manager.NewEntity()
		equipment.AddComponent(world.Components.Item, &gc.Item{})
		equipment.AddComponent(world.Components.Name, &gc.Name{Name: "素早さの指輪"})
		equipment.AddComponent(world.Components.Wearable, &gc.Wearable{
			EquipmentCategory: gc.EquipmentJewelry,
			EquipBonus: gc.EquipBonus{
				Agility: 10, // 素早さ+10
			},
		})

		// 装備を装着（EquipmentChangedフラグが立つ）
		worldhelper.MoveToEquip(world, equipment, player, gc.SlotJewelry)

		// EquipmentChangedSystemを実行
		sys := &EquipmentChangedSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		// APが再計算されていることを確認
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		assert.Greater(t, turnBased.AP.Max, initialAP, "装備追加でAP.Maxが増加するべき")

		// 装備を外す（EquipmentChangedフラグが再度立つ）
		worldhelper.MoveToBackpack(world, equipment, player)

		// EquipmentChangedフラグが立っているか確認
		require.True(t, player.HasComponent(world.Components.EquipmentChanged), "装備を外した後、EquipmentChangedフラグが立っているべき")

		// EquipmentChangedSystemを実行
		err = sys.Update(world)
		require.NoError(t, err)

		// EquipmentChangedフラグが削除されたか確認
		assert.False(t, player.HasComponent(world.Components.EquipmentChanged), "EquipmentChangedSystemの実行後、フラグが削除されるべき")

		// APが元に戻っていることを確認
		turnBased = world.Components.TurnBased.Get(player).(*gc.TurnBased)
		attrs := world.Components.Attributes.Get(player).(*gc.Attributes)
		t.Logf("装備削除後: Agility.Total=%d, AP.Max=%d, 期待AP=%d", attrs.Agility.Total, turnBased.AP.Max, initialAP)
		assert.Equal(t, initialAP, turnBased.AP.Max, "装備削除でAP.Maxが元に戻るべき")
	})

	t.Run("装備変更でHP/SPも再計算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		attrs := &gc.Attributes{
			Vitality:  gc.Attribute{Base: 10, Total: 10},
			Strength:  gc.Attribute{Base: 5, Total: 5},
			Sensation: gc.Attribute{Base: 5, Total: 5},
			Dexterity: gc.Attribute{Base: 5, Total: 5},
			Agility:   gc.Attribute{Base: 5, Total: 5},
			Defense:   gc.Attribute{Base: 0, Total: 0},
		}
		player.AddComponent(world.Components.Attributes, attrs)

		// 初期HP/SPを計算式から算出
		// maxHP: 30 + (体力*8 + 力 + 感覚) = 30 + (10*8 + 5 + 5) = 30 + 90 = 120
		// maxSP: 体力*2 + 器用さ + 素早さ = 10*2 + 5 + 5 = 30
		initialHP := maxHP(attrs)
		initialSP := maxSP(attrs)

		player.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: initialHP, Max: initialHP},
			SP: gc.Pool{Current: initialSP, Max: initialSP},
		})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.Pool{Current: 100, Max: 100},
		})

		// 体力を上げる装備を作成
		equipment := world.Manager.NewEntity()
		equipment.AddComponent(world.Components.Item, &gc.Item{})
		equipment.AddComponent(world.Components.Name, &gc.Name{Name: "体力の鎧"})
		equipment.AddComponent(world.Components.Wearable, &gc.Wearable{
			EquipmentCategory: gc.EquipmentTorso,
			EquipBonus: gc.EquipBonus{
				Vitality: 10, // 体力+10
			},
		})

		// 装備を装着
		worldhelper.MoveToEquip(world, equipment, player, gc.SlotTorso)

		// EquipmentChangedSystemを実行
		sys := &EquipmentChangedSystem{}
		err := sys.Update(world)
		require.NoError(t, err)

		// HP/SPが再計算されていることを確認
		// 体力10→20で: HP = 30 + (20*8 + 5 + 5) = 200、SP = 20*2 + 5 + 5 = 50
		pools := world.Components.Pools.Get(player).(*gc.Pools)
		assert.Greater(t, pools.HP.Max, initialHP, "装備追加でHP.Maxが増加するべき")
		assert.Greater(t, pools.SP.Max, initialSP, "装備追加でSP.Maxが増加するべき")
	})
}
