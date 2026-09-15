package balance

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombatDistribution_モンテカルロと一致(t *testing.T) {
	t.Parallel()
	// 両者が倒し倒される拮抗した組。閉形式のマルコフ連鎖が実戦闘 SimulateBattle と一致することを見る。
	player := CombatantStats{HP: 40, Strength: 5, Sensation: 5, Dexterity: 5, Agility: 5, Defense: 2}
	enemy := CombatantStats{HP: 30, Strength: 8, Sensation: 8, Dexterity: 5, Agility: 5, Defense: 2}
	pw := WeaponStats{Damage: 5, Accuracy: 80}
	ew := WeaponStats{Damage: 6, Accuracy: 80}

	got := CombatDistribution(player, enemy, pw, ew)

	const trials = 300000
	rng := rand.New(rand.NewPCG(1, 2))
	deaths, turnSum := 0, 0
	for range trials {
		r := SimulateBattle(player, enemy, pw, ew, rng)
		if !r.PlayerWon {
			deaths++
		}
		turnSum += r.Turns
	}
	mcDeath := float64(deaths) / trials
	mcTurns := float64(turnSum) / trials

	assert.InDelta(t, mcDeath, got.PlayerDeathProb, 0.01, "死亡確率がMCと一致")
	assert.InDelta(t, mcTurns, got.ExpectedTurns, 0.1, "期待決着ターンがMCと一致")
}

func TestCombatDistribution_圧勝は死亡確率ほぼ0で裾も短い(t *testing.T) {
	t.Parallel()
	// プレイヤーが厚く敵が薄い一方的な組。死亡確率はほぼ0で、決着は数ターンに収まる。
	player := CombatantStats{HP: 200, Strength: 20, Sensation: 20, Dexterity: 20, Agility: 20, Defense: 10}
	enemy := CombatantStats{HP: 10, Strength: 1, Sensation: 1, Dexterity: 1, Agility: 20, Defense: 0}
	pw := WeaponStats{Damage: 20, Accuracy: 90}
	ew := WeaponStats{Damage: 1, Accuracy: 50}

	got := CombatDistribution(player, enemy, pw, ew)
	assert.Less(t, got.PlayerDeathProb, 0.001, "圧勝は死亡確率ほぼ0")
	assert.Positive(t, got.TTKMedian, "中央値が求まる")
	assert.GreaterOrEqual(t, got.TTKp95, got.TTKMedian, "p95は中央値以上")
}

func TestCombatRiskCurve_危険度上昇で死亡確率が非減少傾向(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	require.NoError(t, err)
	weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
	require.NoError(t, err)

	curve, err := CombatRiskCurve(master, player, weapon, BaselineAreaTable, BaselineDays)
	require.NoError(t, err)
	require.Len(t, curve, BaselineDays)
	for _, d := range curve {
		assert.GreaterOrEqual(t, d.DeathProb, 0.0)
		assert.LessOrEqual(t, d.DeathProb, 1.0)
	}
	// 序盤より終盤の方が死にやすい。day1 と day20 を比べて危険度上昇が効いていることを確認する。
	assert.Greater(t, curve[BaselineDays-1].DeathProb, curve[0].DeathProb, "終盤は序盤より死亡確率が高い")
}
