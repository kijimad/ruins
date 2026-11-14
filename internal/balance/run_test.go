package balance

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFloorLoot(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("回復アイテムで回復量が加算される", func(t *testing.T) {
		t.Parallel()
		result := &floorLoot{}
		addFloorLoot(result, master, raw.DrawnItem{Name: "healing_potion", Count: 2}, 80)
		assert.Equal(t, 80, result.healing)
	})

	t.Run("栄養アイテムで栄養値と収入が加算される", func(t *testing.T) {
		t.Parallel()
		result := &floorLoot{}
		addFloorLoot(result, master, raw.DrawnItem{Name: "ice_cream", Count: 3}, 80)
		assert.Equal(t, 225, result.nutrition)
		assert.Equal(t, 30, result.income)
	})

	t.Run("より強い武器に更新され、ダメージの低い武器では更新されない", func(t *testing.T) {
		t.Parallel()
		result := &floorLoot{}
		addFloorLoot(result, master, raw.DrawnItem{Name: "torch", Count: 1}, 80)
		require.NotNil(t, result.weapon)
		assert.Equal(t, "torch", result.weaponName)
		assert.Equal(t, 5, result.weapon.Damage)
		assert.Equal(t, 7, result.income)

		addFloorLoot(result, master, raw.DrawnItem{Name: "bare_hands", Count: 1}, 80)
		assert.Equal(t, "torch", result.weaponName)
		assert.Equal(t, 5, result.weapon.Damage)
	})

	t.Run("存在しないアイテムは無視される", func(t *testing.T) {
		t.Parallel()
		result := &floorLoot{}
		addFloorLoot(result, master, raw.DrawnItem{Name: "存在しないアイテム", Count: 1}, 80)
		assert.Equal(t, floorLoot{}, *result)
	})
}

func TestRollFloorLoot(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("存在しないテーブル名はゼロ値を返す", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(1, 0))
		got := rollFloorLoot(master, "存在しないテーブル", 1, 80, rng)
		assert.Equal(t, floorLoot{}, got)
	})

	t.Run("有効なテーブルからは負にならない loot が算出される", func(t *testing.T) {
		t.Parallel()
		for seed := range 20 {
			rng := rand.New(rand.NewPCG(uint64(seed), 0))
			got := rollFloorLoot(master, "ruins_area", 1, 80, rng)
			assert.GreaterOrEqual(t, got.healing, 0)
			assert.GreaterOrEqual(t, got.nutrition, 0)
			assert.GreaterOrEqual(t, got.income, 0)
			if got.weapon != nil {
				assert.NotEmpty(t, got.weaponName)
			}
		}
	})
}

func TestSimulateRun(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("存在しない敵テーブルは0階到達で終わる", func(t *testing.T) {
		t.Parallel()
		player := CombatantStats{HP: 80}
		weapon := WeaponStats{Damage: 3, Accuracy: 100}
		rng := rand.New(rand.NewPCG(1, 0))
		result := SimulateRun(master, "存在しないテーブル", player, weapon, 3, rng)
		assert.Equal(t, 0, result.ReachedDepth)
		assert.False(t, result.Died)
		assert.Empty(t, result.HPByDepth)
	})

	t.Run("圧倒的に強いプレイヤーは全フロアを生き残る", func(t *testing.T) {
		t.Parallel()
		player := CombatantStats{HP: 100000, Strength: 100, Sensation: 100, Dexterity: 100, Agility: 100, Defense: 100}
		weapon := WeaponStats{Damage: 10000, Accuracy: 100}
		rng := rand.New(rand.NewPCG(7, 0))
		const maxDepth = 3
		result := SimulateRun(master, "ruins_area", player, weapon, maxDepth, rng)

		assert.Equal(t, maxDepth, result.ReachedDepth)
		assert.False(t, result.Died)
		assert.Len(t, result.HPByDepth, maxDepth)
		assert.Len(t, result.HPBeforeHealByDepth, maxDepth)
		assert.Len(t, result.WeaponByDepth, maxDepth)
		assert.Len(t, result.WeaponDamageByDepth, maxDepth)
		assert.Len(t, result.AvgKillTurnsByDepth, maxDepth)
		assert.Len(t, result.HungerByDepth, maxDepth)
		assert.GreaterOrEqual(t, result.LootIncome, 0)
	})
}
