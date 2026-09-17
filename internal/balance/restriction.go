package balance

import (
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ElementValue は1つの武器を持たせたときと、素手へ制限したときの死亡確率の差。Restricted Play の
// 発想を対戦でないサバイバルへ翻案したもの。要素を封じたときの体験メトリクスの劣化量で、その要素が
// どれだけ効いているかを測る。劣化が大きいほど必須で、ゼロに近いほど死にコンテンツ。
type ElementValue struct {
	Element       string  // 武器 id
	DeathProbWith float64 // その武器を持たせたときの死亡確率
	Degradation   float64 // 素手へ制限したときの死亡確率の上昇量。BaselineDeathProb - DeathProbWith
	ExpTurnsWith  float64 // その武器での期待決着ターン
}

// WeaponRestrictionValues は経過日 day の廃墟プールに対し、装備可能な武器すなわち近接と遠距離の
// それぞれを持たせたときの死亡確率と、素手へ制限したときの劣化量を返す。劣化量の降順、すなわち必須な武器から並べる。
// 基準プレイヤーは強化なしの BaselinePlayer。素手 BaselineWeapon が制限時のフォールバックになる。
func WeaponRestrictionValues(master oapi.Raws, enemyTableName string, day int) ([]ElementValue, float64, error) {
	danger := query.DangerLevelForDay(day)
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return nil, 0, err
	}
	bare, err := LoadWeaponFromItem(master, BaselineWeapon)
	if err != nil {
		return nil, 0, err
	}
	baseDeath, _, ok, err := PoolCombatRisk(master, player, bare, enemyTableName, danger, consts.PercentBase)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, nil
	}

	items := raw.PtrSlice(master.Items)
	values := make([]ElementValue, 0, len(items))
	for i := range items {
		// 近接と遠距離の武器を対象にする。素手は基準そのものなので除く。遠距離は Sensation で撃つが、
		// 弾薬の消費と費用はこの指標に含めないので、遠距離の劣化量は弾薬コストを無視した上限になる。
		if (items[i].Melee == nil && items[i].Fire == nil) || items[i].Id == BaselineWeapon {
			continue
		}
		weapon, err := LoadWeaponFromItem(master, items[i].Id)
		if err != nil {
			return nil, 0, err
		}
		death, turns, ok, err := PoolCombatRisk(master, player, weapon, enemyTableName, danger, consts.PercentBase)
		if err != nil {
			return nil, 0, err
		}
		if !ok {
			continue
		}
		values = append(values, ElementValue{
			Element:       items[i].Id,
			DeathProbWith: death,
			Degradation:   baseDeath - death,
			ExpTurnsWith:  turns,
		})
	}
	// 劣化量の降順。同値は id で安定化する
	sort.Slice(values, func(a, b int) bool {
		if values[a].Degradation != values[b].Degradation {
			return values[a].Degradation > values[b].Degradation
		}
		return values[a].Element < values[b].Element
	})
	return values, baseDeath, nil
}
