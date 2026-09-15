package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/skill"
	"github.com/kijimaD/ruins/internal/world/query"
)

// SkillTier はスキル値1段の体験。熟練度倍率と、敵プールを倒す期待ターンを持つ。PlayerTTK は小さいほど
// 速く倒せる。前ティアからの改善量 GapFromPrev がそのレベルアップの手応え。
type SkillTier struct {
	Level       int
	DamageMult  float64 // 熟練度倍率
	PlayerTTK   float64 // プレイヤーが敵プールを倒す期待ターン。重み付き
	GapFromPrev float64 // 前ティアからの PlayerTTK の改善量。0 なら手応えなし
}

// SkillDepthProfile はスキル進行の手応えを測る。NTBEA の「ティア間の最小ギャップを最大化して各段を
// 識別可能にする」発想を、対戦でなく体験メトリクスへ翻案したもの。撃破ターンが動かない死んだティアが
// 多いほど、レベルアップの多くが体験に響かない平坦な進行になる。
type SkillDepthProfile struct {
	Weapon          string
	Tiers           []SkillTier
	EffectiveSteps  int     // 撃破ターンが縮んで体験が動いたレベル数
	DeadTiers       int     // 撃破ターンが前と変わらず、体験が動かないティア数
	MinEffectiveGap float64 // 体験が動いたステップの中で最小の PlayerTTK 改善量。手応えの最小粒
	MinGapAt        int     // 最小の実効ギャップが起きるレベル
}

// SkillDepthProfileFor は weaponName をスキルで強化していったときの進行の手応えを、危険度 danger の
// 敵プールに対して測る。熟練度倍率は実ゲームと同じく base 全体へ切り捨てで掛ける。
// レベルは0から skill.MaxLevel まで全段を見る。丸めの粒を実ゲームに合わせるため、武器ダメージだけを
// 丸める近似はしない。
func SkillDepthProfileFor(master oapi.Raws, weaponName, enemyTableName string, day int) (SkillDepthProfile, error) {
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return SkillDepthProfile{}, err
	}
	weapon, err := LoadWeaponFromItem(master, weaponName)
	if err != nil {
		return SkillDepthProfile{}, err
	}
	danger := query.DangerLevelForDay(day)

	prof := SkillDepthProfile{Weapon: weaponName, MinEffectiveGap: math.Inf(1)}
	prevTTK := math.NaN()
	for level := 0; level <= skill.MaxLevel(); level++ {
		mult := SkillDamagePercent(level)
		ttk, ok := poolPlayerTTK(master, player, weapon, enemyTableName, danger, mult)
		if !ok {
			continue
		}
		tier := SkillTier{Level: level, DamageMult: float64(mult) / float64(consts.PercentBase), PlayerTTK: ttk}
		if !math.IsNaN(prevTTK) {
			tier.GapFromPrev = math.Abs(prevTTK - ttk)
			if tier.GapFromPrev == 0 {
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
		prevTTK = ttk
	}
	if prof.EffectiveSteps == 0 {
		prof.MinEffectiveGap = 0
	}
	return prof, nil
}

// poolPlayerTTK は危険度 danger の敵プールに対する、熟練度倍率 skillMult 込みのプレイヤーの重み付き
// 期待撃破ターンを返す。PlayerTTK はプレイヤーの攻撃力側だけを見る指標で、敵の攻撃力に依らずスキルの
// 手応えを測るのに向く。
func poolPlayerTTK(master oapi.Raws, player CombatantStats, playerWeapon WeaponStats, enemyTableName string, danger int, skillMult consts.Percent) (float64, bool) {
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
		ttkSum += entry.Weight * ExpectedTTKWithSkill(player, enemy, playerWeapon, skillMult)
	}
	if wSum == 0 {
		return 0, false
	}
	return ttkSum / wSum, true
}
