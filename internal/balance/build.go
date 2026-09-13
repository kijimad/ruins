package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/oapi"
)

// BuildProfile は評価するプレイヤーのビルド。能力と武器の組で、スキル成長や装備・バフを織り込んだ
// 想定像を表す。静的下限は強化なしの1プロファイルとして扱う。
type BuildProfile struct {
	Name   string
	Player CombatantStats
	Weapon WeaponStats
}

// RepresentativeBuilds は基準プレイヤーを土台に、武器ダメージ倍率で段階を表した代表ビルドを返す。
// 下限は強化なし、中盤・後半はスキル成長と装備・バフでダメージが伸びた想定。倍率は設計仮説で見直す。
func RepresentativeBuilds(master oapi.Raws) ([]BuildProfile, error) {
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return nil, err
	}
	weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
	if err != nil {
		return nil, err
	}
	scaled := func(mult float64) WeaponStats {
		w := weapon
		w.Damage = int(math.Round(float64(weapon.Damage) * mult))
		return w
	}
	return []BuildProfile{
		{Name: "下限(強化なし)", Player: player, Weapon: scaled(1.0)},
		{Name: "中盤想定(×2)", Player: player, Weapon: scaled(2.0)},
		{Name: "後半想定(×3.5)", Player: player, Weapon: scaled(3.5)},
	}, nil
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

// BuildSpread は代表ビルドで敵テーブルのカーブを評価し、各日の戦力比の最小と最大を返す。
// 静的下限だけでなく、スキル・装備・バフで戦力が振れる幅を可視化する。後半で幅が急に開くなら、
// バフが強すぎて終盤が崩れやすいことを示す。
func BuildSpread(master oapi.Raws, profiles []BuildProfile, enemyTableName string, days int) ([]DaySpreadRow, error) {
	curves := make([][]DayMetric, 0, len(profiles))
	for _, p := range profiles {
		c, err := DifficultyCurve(master, p.Player, p.Weapon, enemyTableName, days)
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
