package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolCombatRisk(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	require.NoError(t, err)
	weapon, err := LoadWeaponFromBodyAttack(master, raw.BareHands)
	require.NoError(t, err)

	t.Run("存在しない敵テーブルはエラーを返す", func(t *testing.T) {
		t.Parallel()
		_, _, ok, err := PoolCombatRisk(master, player, weapon, "no_such_table", 1, consts.PercentBase)
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
		assert.Equal(t, raw.KeyNotFoundError{Key: "no_such_table", Collection: "EnemyTables"}, notFound)
		assert.False(t, ok, "エラー時はokもfalse")
	})

	t.Run("危険度帯に敵がいなければokはfalse", func(t *testing.T) {
		t.Parallel()
		// master はフィールドを丸ごと差し替えるだけなので、値の写しを使っても共有の master は変わらない
		m := master
		m.EnemyTables = &[]rawschema.EnemyTable{{
			Id:      "test_gap",
			Entries: []rawschema.EnemyTableEntry{{Id: "unreachable", MinDanger: 100, MaxDanger: 200, Pack: "1", Weight: 1}},
		}}
		deathProb, expTurns, ok, err := PoolCombatRisk(m, player, weapon, "test_gap", 1, consts.PercentBase)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Zero(t, deathProb)
		assert.Zero(t, expTurns)
	})

	t.Run("entryが存在しないメンバーを指すとエラーを返す", func(t *testing.T) {
		t.Parallel()
		m := master
		m.EnemyTables = &[]rawschema.EnemyTable{{
			Id:      "test_broken_member",
			Entries: []rawschema.EnemyTableEntry{{Id: "no_such_member", MinDanger: 1, MaxDanger: 20, Pack: "1", Weight: 1}},
		}}
		_, _, ok, err := PoolCombatRisk(m, player, weapon, "test_broken_member", 1, consts.PercentBase)
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
		assert.Equal(t, raw.KeyNotFoundError{Key: "no_such_member", Collection: "Members"}, notFound)
		assert.False(t, ok)
	})
}

func TestCombatRiskCurve(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	require.NoError(t, err)
	weapon, err := LoadWeaponFromBodyAttack(master, raw.BareHands)
	require.NoError(t, err)

	t.Run("危険度帯に敵がいない日は空のDayRiskを置く", func(t *testing.T) {
		t.Parallel()
		// day1の危険度はどのentryのminDangerにも届かないので、1日目はentryが無い帯になる
		m := master
		m.EnemyTables = &[]rawschema.EnemyTable{{
			Id:      "test_gap",
			Entries: []rawschema.EnemyTableEntry{{Id: "unreachable", MinDanger: 100, MaxDanger: 200, Pack: "1", Weight: 1}},
		}}
		curve, err := CombatRiskCurve(m, player, weapon, "test_gap", 1)
		require.NoError(t, err)
		assert.Equal(t, []DayRisk{{Day: 1, Danger: int(query.DangerLevelForDay(1))}}, curve)
	})

	t.Run("存在しない敵テーブルはエラーを返す", func(t *testing.T) {
		t.Parallel()
		_, err := CombatRiskCurve(master, player, weapon, "no_such_table", BaselineDays)
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
		assert.Equal(t, raw.KeyNotFoundError{Key: "no_such_table", Collection: "EnemyTables"}, notFound)
	})
}

func TestAttackDamagePMF_命中は0ダメージにならない(t *testing.T) {
	t.Parallel()
	// 防御が高くダメージが下限に張り付く組でも、命中は MinDamage 以上で 0 にならない。dmg=0 は miss
	// だけを表し、その確率は不命中率に一致する。この不変条件が崩れると 0 キーに miss と命中が混ざる。
	attacker := CombatantStats{Strength: 1, Sensation: 1, Dexterity: 5, Agility: 5}
	defender := CombatantStats{HP: 30, Agility: 5, Defense: 50}
	weapon := WeaponStats{Damage: 1, Accuracy: 80}

	pmf := attackDamagePMF(attacker, defender, weapon, consts.PercentBase)

	hitRate := formula.CalcHitRate(attacker.Dexterity, defender.Agility, weapon.Accuracy)
	wantMiss := float64(formula.DiceMax-hitRate) / float64(formula.DiceMax)

	assert.GreaterOrEqual(t, formula.MinDamage, 1, "MinDamage が1以上でないと 0 キーが miss と命中で混ざる")
	var sum, missP float64
	zeros := 0
	for _, dp := range pmf {
		sum += dp.p
		if dp.dmg == 0 {
			zeros++
			missP = dp.p
		} else {
			assert.GreaterOrEqual(t, dp.dmg, formula.MinDamage, "命中ダメージは MinDamage 以上")
		}
	}
	assert.Equal(t, 1, zeros, "dmg=0 は miss だけの単一エントリ")
	assert.InDelta(t, wantMiss, missP, 1e-12, "dmg=0 の確率は不命中率に一致")
	assert.InDelta(t, 1.0, sum, 1e-12, "確率の総和は1")
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
	weapon, err := LoadWeaponFromBodyAttack(master, raw.BareHands)
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

func TestCombatDistribution_HPが1どうしなら先に当てた側が勝つ式に一致する(t *testing.T) {
	t.Parallel()
	player := CombatantStats{HP: 1, Strength: 5, Sensation: 5, Dexterity: 6, Agility: 6}
	enemy := CombatantStats{HP: 1, Strength: 5, Sensation: 5, Dexterity: 9, Agility: 3}
	pw := WeaponStats{Damage: 3, Accuracy: 60}
	ew := WeaponStats{Damage: 3, Accuracy: 80}
	pp := float64(formula.CalcHitRate(player.Dexterity, enemy.Agility, pw.Accuracy)) / float64(formula.DiceMax)
	pe := float64(formula.CalcHitRate(enemy.Dexterity, player.Agility, ew.Accuracy)) / float64(formula.DiceMax)

	got := CombatDistribution(player, enemy, pw, ew)

	// プレイヤーが先に攻撃するので、プレイヤーが外し続けるあいだに敵が先に当てると倒される
	want := (1 - pp) * pe / (1 - (1-pp)*(1-pe))
	assert.InDelta(t, want, got.PlayerDeathProb, 1e-9)
	// 毎ターンどちらかが当てれば決着するので、決着ターンは成功確率 1-(1-pp)(1-pe) の幾何分布になる
	assert.InDelta(t, 1/(1-(1-pp)*(1-pe)), got.ExpectedTurns, 1e-9)
}
