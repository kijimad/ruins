package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 戦闘での怪我のパラメータ。値は実プレイで調整する
const (
	// injuryChancePercent は命中1回で怪我が付く確率。怪我はたまに起きる脇役なので低く保つ
	injuryChancePercent = 15
	// injuryInitialTimer は付いた怪我の初期進行度。発症する 25 以上にして即座に効かせる
	injuryInitialTimer = 40
	// maxInjuriesPerPartType は1部位に積める同種の怪我の上限。独立傷が溢れないようにする
	maxInjuriesPerPartType = 3
)

// injuryTypeFor は武器種から怪我の種類を導く。鈍器は骨折、刃と弾は切り傷。
// 大砲は射撃だが重い衝撃で骨を砕くので骨折側に置く
func injuryTypeFor(attackType gc.AttackType) gc.ConditionType {
	switch attackType {
	case gc.AttackFist, gc.AttackCanon:
		return gc.ConditionFracture
	default:
		return gc.ConditionLaceration
	}
}

// applyInjury は命中時に確率で命中部位へ独立した怪我を1つ付ける。
// 怪我の種類は武器種から導き、命中部位は部位サイズの重みで抽選する。HealthStatus を持つ対象だけが対象。
// 同じ部位に同種の怪我がソフト上限に達していれば付けない。独立に積むので capacity と失血は全傷が合算される
func applyInjury(actor, target ecs.Entity, world w.World, attack gc.Attacker) {
	if !world.Components.HealthStatus.Has(target) {
		return
	}
	if world.Resources.Config.RNG.IntN(100) >= injuryChancePercent {
		return
	}

	part := randomHitPart(world)
	injuryType := injuryTypeFor(attack.GetAttackCategory())

	hs := world.Components.HealthStatus.Get(target)
	if hs.Parts[part].CountConditions(injuryType) >= maxInjuriesPerPartType {
		return
	}
	hs.Parts[part].AddCondition(gc.HealthCondition{
		Type:     injuryType,
		Timer:    injuryInitialTimer,
		Severity: gc.TimerToSeverity(injuryInitialTimer),
	})

	// 怪我は身体機能を下げるので CharModifiers の再計算を促す
	if !world.Components.StatsChanged.Has(target) {
		world.Components.StatsChanged.Add(target, &gc.StatsChanged{})
	}

	logInjury(actor, target, world, part, injuryType)
}

// randomHitPart は命中部位を重みで抽選する
func randomHitPart(world w.World) gc.BodyPart {
	return hitPartForRoll(world.Resources.Config.RNG.IntN(totalHitWeight()))
}

// totalHitWeight は全部位の命中重みの合計を返す
func totalHitWeight() int {
	total := 0
	for p := range gc.BodyPartCount {
		total += gc.BodyPartHitWeight(p)
	}
	return total
}

// hitPartForRoll は重み合計内の roll を命中部位へ写す。roll は [0, totalHitWeight) を想定する。
// 部位順に走査し、重み 0 の部位は飛ばすので、命中先は必ず重みのある部位になる
func hitPartForRoll(roll int) gc.BodyPart {
	for p := range gc.BodyPartCount {
		weight := gc.BodyPartHitWeight(p)
		if weight == 0 {
			continue
		}
		if roll < weight {
			return p
		}
		roll -= weight
	}
	return gc.BodyPartTorso // 合計重みの内で必ず返るのでここには来ない
}

// logInjury は怪我を負ったことをログに出す。攻撃者か対象が味方に関わるときだけ出す
func logInjury(actor, target ecs.Entity, world w.World, part gc.BodyPart, injuryType gc.ConditionType) {
	if !query.IsAlly(world, actor) && !query.IsAlly(world, target) {
		return
	}
	targetName := query.GetEntityName(target, world)
	targetMarkup := query.NameMarkup(target, targetName, world)
	partName := query.T(world, part.String())
	injuryName := query.T(world, gc.ConditionTypeDisplayName(injuryType))
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s suffered %s on the %s.", targetMarkup, injuryName, partName)).
		Log()
}
