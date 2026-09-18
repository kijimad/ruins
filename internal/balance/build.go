package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// BuildProfile は評価するプレイヤーのビルド。能力と武器に加え、スキル成長ぶんの熟練度倍率を持つ。
// 倍率は実ゲームと同じく base 全体へ切り捨てで掛かる。静的下限は倍率 PercentBase の1プロファイル。
type BuildProfile struct {
	Name      string
	Player    CombatantStats
	Weapon    WeaponStats
	SkillMult consts.Percent
}

// buildStages は代表ビルドの段階と、その段階までに想定する累積攻撃回数。攻撃回数は scenario 入力で、
// スキル成長 SkillLevelAfterAttacks と熟練度 SkillDamageMultiplier を経由してダメージ倍率になる。
var buildStages = []struct {
	name    string
	attacks int
}{
	{"下限(0攻撃)", 0},
	{"中盤(500攻撃)", 500},
	{"後半(2000攻撃)", 2000},
}

// RepresentativeBuilds は基準プレイヤーを土台に、想定攻撃回数から導いた強化度合いで段階を表した代表ビルドを返す。
// 倍率はスキル成長と熟練度の実システムから導き、スキル由来のダメージ倍率だけを反映する。攻撃回数は設計仮説。
func RepresentativeBuilds(master oapi.Raws) ([]BuildProfile, error) {
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return nil, err
	}
	weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
	if err != nil {
		return nil, err
	}
	builds := make([]BuildProfile, 0, len(buildStages))
	for _, st := range buildStages {
		mult := SkillDamagePercent(SkillLevelAfterAttacks(0, st.attacks))
		builds = append(builds, BuildProfile{Name: st.name, Player: player, Weapon: weapon, SkillMult: mult})
	}
	return builds, nil
}

// DaySpreadRow は代表ビルド間の、ある日の戦力比の幅。強武器ほど戦力比が上がるので、最大は最強ビルド、
// 最小は下限ビルドに対応する。幅が広いほどビルド次第で難易度が大きく振れる。
type DaySpreadRow struct {
	Day      int
	MinRatio float64
	MaxRatio float64
}

// Spread は最大と最小の差。後半のブレの大きさを表す。
func (r DaySpreadRow) Spread() float64 {
	return r.MaxRatio - r.MinRatio
}

// BuildSpread は代表ビルドで敵テーブルのカーブを評価し、各日の戦力比の最小と最大を返す。後半で幅が急に開くなら
// バフが強すぎて終盤が崩れやすいことを示す。
func BuildSpread(master oapi.Raws, profiles []BuildProfile, enemyTableName string, days int) ([]DaySpreadRow, error) {
	curves := make([][]DayMetric, 0, len(profiles))
	for _, p := range profiles {
		c, err := DifficultyCurveWithSkill(master, p.Player, p.Weapon, enemyTableName, days, p.SkillMult)
		if err != nil {
			return nil, err
		}
		curves = append(curves, c)
	}
	out := make([]DaySpreadRow, 0, days)
	for i := range days {
		mn, mx := math.Inf(1), math.Inf(-1)
		for _, c := range curves {
			r := c[i].PowerRatio
			mn = math.Min(mn, r)
			mx = math.Max(mx, r)
		}
		out = append(out, DaySpreadRow{Day: i + 1, MinRatio: mn, MaxRatio: mx})
	}
	return out, nil
}

// StageSpread は1つのゲーム段階での、代表ビルド間の戦力比の幅の最大と、その許容幅。
type StageSpread struct {
	Stage     string
	FromDay   int
	ToDay     int
	MaxSpread float64 // 段階内の日で最大の戦力比の幅
	Tolerance float64 // 許容する幅
}

// InRange は段階内の最大の幅が許容幅に収まるかを返す。
func (s StageSpread) InRange() bool {
	return s.MaxSpread <= s.Tolerance
}

// stageBands はゲーム段階の日範囲とブレの許容幅。後半ほどバフで開きやすいので許容を絞る。設計仮説で見直す。
var stageBands = []struct {
	name     string
	from, to int
	tol      float64
}{
	{"序盤", 1, 7, 2.0},
	{"中盤", 8, 14, 1.6},
	{"終盤", 15, 21, 1.2},
}

// StageSpreads は代表ビルドの戦力比の幅を段階ごとに集計し、各段階での最大の幅と許容帯を返す。
// 幅が許容を超える段階はビルド次第で難易度が振れすぎる。特に終盤の超過はバフが強すぎる兆候。
func StageSpreads(master oapi.Raws, profiles []BuildProfile, enemyTableName string) ([]StageSpread, error) {
	spread, err := BuildSpread(master, profiles, enemyTableName, 21)
	if err != nil {
		return nil, err
	}
	byDay := make(map[int]float64, len(spread))
	for _, r := range spread {
		byDay[r.Day] = r.Spread()
	}
	out := make([]StageSpread, 0, len(stageBands))
	for _, st := range stageBands {
		mx := 0.0
		for d := st.from; d <= st.to; d++ {
			mx = math.Max(mx, byDay[d])
		}
		out = append(out, StageSpread{Stage: st.name, FromDay: st.from, ToDay: st.to, MaxSpread: mx, Tolerance: st.tol})
	}
	return out, nil
}
