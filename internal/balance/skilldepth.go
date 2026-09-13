package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/skill"
	"github.com/kijimaD/ruins/internal/world/query"
)

// SkillTier はスキル値1段の体験。熟練度倍率と丸め後の武器ダメージ、敵プールを倒す期待ターンを持つ。
// PlayerTTK は小さいほど速く倒せる。前ティアからの改善量 GapFromPrev がそのレベルアップの手応え。
type SkillTier struct {
	Level        int
	DamageMult   float64 // 熟練度倍率
	WeaponDamage int     // 丸め後の武器ダメージ
	PlayerTTK    float64 // プレイヤーが敵プールを倒す期待ターン。重み付き
	GapFromPrev  float64 // 前ティアからの PlayerTTK の改善量。0 なら手応えなし
}

// SkillDepthProfile はスキル進行の手応えを測る。NTBEA の「ティア間の最小ギャップを最大化して各段を
// 識別可能にする」発想を、対戦でなく体験メトリクスへ翻案したもの。丸めで武器ダメージが変わらない死んだ
// ティアが多いほど、レベルアップの多くが体験に響かない平坦な進行になる。
type SkillDepthProfile struct {
	Weapon          string
	Tiers           []SkillTier
	EffectiveSteps  int     // 武器ダメージが増えて体験が動いたレベル数
	DeadTiers       int     // 丸めで武器ダメージが前と変わらず、体験が動かないティア数
	MinEffectiveGap float64 // 体験が動いたステップの中で最小の PlayerTTK 改善量。手応えの最小粒
	MinGapAt        int     // 最小の実効ギャップが起きるレベル
}

// SkillDepthProfileFor は weaponName をスキルで強化していったときの進行の手応えを、危険度 danger の
// 敵プールに対して測る。スキルは build.go と同じく武器ダメージへ倍率として反映する。近似は同じ。
// レベルは0から skill.MaxLevel まで全段を見る。
func SkillDepthProfileFor(master oapi.Raws, weaponName, enemyTableName string, day int) (SkillDepthProfile, error) {
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return SkillDepthProfile{}, err
	}
	base, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return SkillDepthProfile{}, err
	}
	danger := query.DangerLevelForDay(day)

	prof := SkillDepthProfile{Weapon: weaponName, MinEffectiveGap: math.Inf(1)}
	prevTTK, prevDmg := math.NaN(), -1
	for level := 0; level <= skill.MaxLevel(); level++ {
		mult := SkillDamageMultiplier(level)
		w := base
		w.Damage = int(math.Round(float64(base.Damage) * mult))
		ttk, ok := poolPlayerTTK(master, player, w, enemyTableName, danger)
		if !ok {
			continue
		}
		tier := SkillTier{Level: level, DamageMult: mult, WeaponDamage: w.Damage, PlayerTTK: ttk}
		if prevDmg >= 0 {
			tier.GapFromPrev = math.Abs(prevTTK - ttk)
			if w.Damage == prevDmg {
				prof.DeadTiers++
			} else {
				prof.EffectiveSteps++
				if tier.GapFromPrev < prof.MinEffectiveGap {
					prof.MinEffectiveGap = tier.GapFromPrev
					prof.MinGapAt = level
				}
			}
		}
		prof.Tiers = append(prof.Tiers, tier)
		prevTTK, prevDmg = ttk, w.Damage
	}
	if prof.EffectiveSteps == 0 {
		prof.MinEffectiveGap = 0
	}
	return prof, nil
}

// poolPlayerTTK は危険度 danger の敵プールに対する、プレイヤーの重み付き期待撃破ターンを返す。
// PlayerTTK はプレイヤーの攻撃力側だけを見る指標で、敵の攻撃力に依らずスキルの手応えを測るのに向く。
func poolPlayerTTK(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, danger int) (float64, bool) {
	table, err := raw.GetEnemyTable(master, enemyTableName)
	if err != nil {
		return 0, false
	}
	var wSum, ttkSum float64
	for _, entry := range table.Entries {
		if danger < entry.MinDanger || danger > entry.MaxDanger {
			continue
		}
		enemy, err := LoadCombatantFromMember(master, entry.Id)
		if err != nil {
			return 0, false
		}
		wSum += entry.Weight
		ttkSum += entry.Weight * ExpectedTTK(player, enemy, playerWeapon)
	}
	if wSum == 0 {
		return 0, false
	}
	return ttkSum / wSum, true
}
