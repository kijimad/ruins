package balance

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestMaster(t *testing.T) oapi.Raws {
	t.Helper()
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "raw.tomlの読み込みに失敗")
	return master
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

	// DPSは正の値
	assert.Greater(t, bs.DPS(), 0.0, "スライムへのDPSは正")
}

func TestRunSimulation_Basic(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	player, err := LoadCombatantFromMember(master, "Ash")
	require.NoError(t, err)

	playerWeapon, err := LoadWeaponFromItem(master, "素手")
	require.NoError(t, err)

	rng := rand.New(rand.NewPCG(42, 0))
	result := SimulateRun(master, "廃墟", player, playerWeapon, 5, rng)

	// シード固定のため結果は決定論的。素手でも最低1階層には到達する
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

	stats := RunSimulations(master, "廃墟", player, playerWeapon, 10, 100, 42)

	// 基本的な統計が取れることの確認
	assert.Positive(t, stats.MedianDepth())
	assert.GreaterOrEqual(t, stats.DeathRate(), 0.0)

	// 各深度の統計が取れる
	hps := stats.HPAtDepth(1)
	assert.NotEmpty(t, hps, "深度1のHP分布が取れる")
	assert.GreaterOrEqual(t, stats.MedianHP(1), 0, "深度1のHP中央値は非負")
	assert.GreaterOrEqual(t, stats.P5HP(1), 0, "深度1のP5HPは非負")
	assert.GreaterOrEqual(t, stats.P95HP(1), 0, "深度1のP95HPは非負")

}

func TestLoadWeaponFromItem(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("近接武器を読み込める", func(t *testing.T) {
		t.Parallel()
		ws, err := LoadWeaponFromItem(master, "素手")
		require.NoError(t, err)
		assert.False(t, ws.IsRanged)
	})

	t.Run("存在しないアイテムはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadWeaponFromItem(master, "存在しない武器")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ロードに失敗")
	})
}

func TestLoadEnemyWeapon(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("スライムの武器を読み込める", func(t *testing.T) {
		t.Parallel()
		ws, err := LoadEnemyWeapon(master, "スライム")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, ws.Damage, 0)
	})

	t.Run("存在しない敵はエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadEnemyWeapon(master, "存在しない敵")
		require.Error(t, err)
	})
}

func TestLoadCombatantFromMember_Errors(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)

	t.Run("存在しないメンバーはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadCombatantFromMember(master, "存在しないメンバー")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ロードに失敗")
	})
}

func TestRollAttack(t *testing.T) {
	t.Parallel()

	attacker := CombatantStats{Strength: 10, Dexterity: 10}
	defender := CombatantStats{Agility: 5, Defense: 3}
	weapon := WeaponStats{Damage: 5, Accuracy: 80}

	// 固定シードで複数回実行して、ダメージが非負であることを確認
	rng := rand.New(rand.NewPCG(42, 0))
	for range 100 {
		dmg := rollAttack(attacker, defender, weapon, rng)
		assert.GreaterOrEqual(t, dmg, 0)
	}
}

func TestRollAttack_Ranged(t *testing.T) {
	t.Parallel()

	attacker := CombatantStats{Sensation: 15, Dexterity: 10}
	defender := CombatantStats{Agility: 5, Defense: 3}
	weapon := WeaponStats{Damage: 5, Accuracy: 80, IsRanged: true}

	rng := rand.New(rand.NewPCG(42, 0))
	for range 100 {
		dmg := rollAttack(attacker, defender, weapon, rng)
		assert.GreaterOrEqual(t, dmg, 0)
	}
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
