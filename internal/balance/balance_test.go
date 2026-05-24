package balance

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestMaster(t *testing.T) *raw.Master {
	t.Helper()
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "raw.tomlの読み込みに失敗")
	return &master
}

func TestCalcHitRate(t *testing.T) {
	t.Parallel()

	t.Run("同じステータスでは基本命中率になる", func(t *testing.T) {
		t.Parallel()
		attacker := CombatantStats{Dexterity: 10}
		target := CombatantStats{Agility: 10}
		weapon := WeaponStats{Accuracy: baseHitRate}
		assert.Equal(t, baseHitRate, CalcHitRate(attacker, target, weapon))
	})

	t.Run("器用度が高いと命中率が上がる", func(t *testing.T) {
		t.Parallel()
		attacker := CombatantStats{Dexterity: 15}
		target := CombatantStats{Agility: 10}
		weapon := WeaponStats{Accuracy: baseHitRate}
		hitRate := CalcHitRate(attacker, target, weapon)
		assert.Equal(t, baseHitRate+5*hitRatePerStatPoint, hitRate)
	})

	t.Run("最大命中率を超えない", func(t *testing.T) {
		t.Parallel()
		attacker := CombatantStats{Dexterity: 100}
		target := CombatantStats{Agility: 1}
		weapon := WeaponStats{Accuracy: baseHitRate}
		assert.Equal(t, maxHitRate, CalcHitRate(attacker, target, weapon))
	})

	t.Run("最小命中率を下回らない", func(t *testing.T) {
		t.Parallel()
		attacker := CombatantStats{Dexterity: 1}
		target := CombatantStats{Agility: 100}
		weapon := WeaponStats{Accuracy: baseHitRate}
		assert.Equal(t, minHitRate, CalcHitRate(attacker, target, weapon))
	})
}

func TestCalcDamage(t *testing.T) {
	t.Parallel()

	t.Run("最低保証ダメージが1", func(t *testing.T) {
		t.Parallel()
		attacker := CombatantStats{Strength: 1}
		target := CombatantStats{Defense: 100}
		weapon := WeaponStats{Damage: 0}
		rng := rand.New(rand.NewPCG(0, 0))
		dmg := CalcDamage(attacker, target, weapon, rng)
		assert.Equal(t, minDamage, dmg)
	})

	t.Run("遠距離武器では感覚を参照する", func(t *testing.T) {
		t.Parallel()
		attacker := CombatantStats{Strength: 1, Sensation: 20}
		target := CombatantStats{Defense: 0}
		weapon := WeaponStats{Damage: 0, IsRanged: true}
		rng := rand.New(rand.NewPCG(0, 0))
		dmg := CalcDamage(attacker, target, weapon, rng)
		// 感覚20 + rand(1..6) + 0 - 0 >= 21
		assert.GreaterOrEqual(t, dmg, 21)
	})
}

func TestSimulateBattle(t *testing.T) {
	t.Parallel()

	t.Run("強いプレイヤーは必ず勝つ", func(t *testing.T) {
		t.Parallel()
		player := CombatantStats{HP: 1000, Strength: 50, Dexterity: 20, Defense: 50}
		enemy := CombatantStats{HP: 10, Strength: 1, Dexterity: 1, Agility: 1}
		rng := rand.New(rand.NewPCG(42, 0))
		br := SimulateBattle(player, enemy, WeaponStats{Accuracy: 80}, WeaponStats{Accuracy: 80}, rng)
		assert.True(t, br.PlayerWon)
		assert.LessOrEqual(t, br.Turns, 5)
	})
}

func TestLoadCombatantFromMember(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("プレイヤーのステータスを読み込める", func(t *testing.T) {
		t.Parallel()
		stats, err := LoadCombatantFromMember(master, "Ash")
		require.NoError(t, err)
		// HP = 30 + 5*8 + 5 + 5 = 80
		assert.Equal(t, 80, stats.HP)
		assert.Equal(t, 5, stats.Strength)
		assert.Equal(t, 5, stats.Agility)
		assert.Equal(t, 3, stats.Defense)
	})

	t.Run("敵のステータスを読み込める", func(t *testing.T) {
		t.Parallel()
		stats, err := LoadCombatantFromMember(master, "スライム")
		require.NoError(t, err)
		// HP = 30 + 1*8 + 2 + 2 = 42
		assert.Equal(t, 42, stats.HP)
		assert.Equal(t, 2, stats.Strength)
	})
}

func TestBattle_Depth1_Slime(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	player, err := LoadCombatantFromMember(master, "Ash")
	require.NoError(t, err)

	enemy, err := LoadCombatantFromMember(master, "スライム")
	require.NoError(t, err)

	// 素手で戦闘
	playerWeapon, err := LoadWeaponFromItem(master, "素手")
	require.NoError(t, err)

	enemyWeapon, err := LoadEnemyWeapon(master, "スライム")
	require.NoError(t, err)

	rng := rand.New(rand.NewPCG(42, 0))
	results := RunBattles(player, enemy, playerWeapon, enemyWeapon, 1000, rng)
	bs := BattleStats{Results: results}

	// スライムには高確率で勝てるはず
	assert.Greater(t, bs.WinRate(), 0.9, "スライムへの勝率は90%以上")
	// 撃破ターン数はそこそこ短い
	assert.Less(t, bs.AvgTTK(), 30.0, "スライムの平均撃破ターン数は30未満")
	// 被ダメージの平均は正の値
	assert.Greater(t, bs.AvgDamageTaken(), 0.0, "被ダメージの平均は正")
}

func TestRunSimulation_Basic(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	player, err := LoadCombatantFromMember(master, "Ash")
	require.NoError(t, err)

	playerWeapon, err := LoadWeaponFromItem(master, "素手")
	require.NoError(t, err)

	rng := rand.New(rand.NewPCG(42, 0))
	result := SimulateRun(master, "通常", player, playerWeapon, 5, rng)

	// 5階層分は少なくとも到達できる可能性が高い
	assert.GreaterOrEqual(t, result.ReachedDepth, 1)
	assert.NotEmpty(t, result.HPByDepth)
}

func TestRunSimulations_Stats(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	player, err := LoadCombatantFromMember(master, "Ash")
	require.NoError(t, err)

	playerWeapon, err := LoadWeaponFromItem(master, "素手")
	require.NoError(t, err)

	stats := RunSimulations(master, "通常", player, playerWeapon, 10, 100, 42)

	// 基本的な統計が取れることの確認
	assert.Greater(t, stats.MedianDepth(), 0)
	assert.GreaterOrEqual(t, stats.DeathRate(), 0.0)

	// 各深度の統計が取れる
	hps := stats.HPAtDepth(1)
	assert.NotEmpty(t, hps, "深度1のHP分布が取れる")
	assert.GreaterOrEqual(t, stats.MedianHP(1), 0, "深度1のHP中央値は非負")
	assert.GreaterOrEqual(t, stats.P5HP(1), 0, "深度1のP5HPは非負")
	assert.GreaterOrEqual(t, stats.P95HP(1), 0, "深度1のP95HPは非負")

}

func TestPercentile(t *testing.T) {
	t.Parallel()

	t.Run("空スライスでは0を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, Percentile(nil, 0.5))
	})

	t.Run("ソート済みスライスの中央値", func(t *testing.T) {
		t.Parallel()
		sorted := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		assert.Equal(t, 5, Percentile(sorted, 0.5))
	})

	t.Run("下位5%", func(t *testing.T) {
		t.Parallel()
		sorted := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		assert.Equal(t, 1, Percentile(sorted, 0.05))
	})
}

func TestMedian(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 3, Median([]int{5, 1, 3}))
	assert.Equal(t, 0, Median(nil))
}

func TestMean(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 3.0, Mean([]int{1, 2, 3, 4, 5}), 0.01)
	assert.InDelta(t, 0.0, Mean(nil), 0.01)
}
