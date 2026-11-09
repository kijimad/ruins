package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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
	// maxInjuriesPerPartType は1部位に付く同種の怪我の上限。種類が違えば共存する
	maxInjuriesPerPartType = 1
)

// applyInjury は命中時に確率で命中部位へ怪我を1つ付ける。
// 怪我の種類は攻撃種別の行が持ち、命中部位はその怪我が起こりうる部位から部位サイズの重みで抽選する。HealthStatus を持つ対象だけが対象。
// 同じ部位に同種の怪我が既にあれば付けない。種類や部位が違う傷は独立で、身体機能と失血は全傷が合算される
func applyInjury(actor, target ecs.Entity, world w.World, attack gc.Attacker) {
	if !world.Components.HealthStatus.Has(target) {
		return
	}
	if world.Resources.Config.RNG.IntN(100) >= injuryChancePercent {
		return
	}

	defs := world.Resources.ConditionDefs
	injuryType, ok := defs.Injury(attack.GetAttackCategory())
	if !ok {
		return
	}
	part := hitPart(world.Resources.Config.RNG, defs, injuryType)

	hs := world.Components.HealthStatus.Get(target)
	if hs.Parts[part].CountConditions(injuryType) >= maxInjuriesPerPartType {
		return
	}
	hs.Parts[part].AddCondition(gc.HealthCondition{
		Type:     injuryType,
		Timer:    injuryInitialTimer,
		Severity: gc.TimerToSeverity(injuryInitialTimer),
	})

	logInjury(actor, target, world, part, injuryType)
}

// hitPart は怪我 ct が起こりうる部位から、命中の重みで部位を1つ引く。
func hitPart(r consts.Roller, defs gc.ConditionDefs, ct gc.ConditionType) gc.BodyPart {
	parts := make([]gc.BodyPart, gc.BodyPartCount)
	for p := range gc.BodyPartCount {
		parts[p] = p
	}
	i, ok := consts.Pick(r, parts, func(p gc.BodyPart) float64 {
		if !defs.CanAffect(ct, p) {
			return 0
		}
		return defs.HitWeight(p)
	})
	if !ok {
		panic(fmt.Sprintf("no body part can take %s", ct))
	}
	return parts[i]
}

// logInjury は怪我を負ったことをログに出す。攻撃者か対象が味方に関わるときだけ出す
func logInjury(actor, target ecs.Entity, world w.World, part gc.BodyPart, injuryType gc.ConditionType) {
	if !query.IsAlly(world, actor) && !query.IsAlly(world, target) {
		return
	}
	targetName := query.GetEntityName(target, world)
	targetMarkup := query.NameMarkup(target, targetName, world)
	partName := query.T(world, part.String())
	injuryName := query.ConditionName(world, injuryType)
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s suffered %s on the %s.", targetMarkup, injuryName, partName)).
		Log()
}
