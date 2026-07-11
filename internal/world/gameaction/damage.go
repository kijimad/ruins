package gameaction

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ApplyDamage は共通のダメージ処理を実行する
// source から target へダメージを与え、死亡判定とログ出力を行う
func ApplyDamage(world w.World, target ecs.Entity, damage int, source ecs.Entity) {
	hp := world.Components.HP.Get(target)

	beforeHP := hp.Current
	hp.Current -= damage
	if hp.Current < 0 {
		hp.Current = 0
	}

	// 被ダメージによる態度変化
	reactToHostileAction(world, target)

	// 死亡チェック
	if hp.Current <= 0 && beforeHP > 0 {
		world.Components.Dead.Add(target, &gc.Dead{})
		logDeath(world, target, source)
	}
}

// reactToHostileAction は被ダメージ時にAIの戦闘方針を変化させる。
// CombatIgnore は反撃のため CombatAttack に遷移する
func reactToHostileAction(world w.World, target ecs.Entity) {
	if world.Components.SoloAI.Has(target) {
		world.Components.SoloAI.Get(target).ReactToHostile()
	}
	if world.Components.SquadAI.Has(target) {
		world.Components.SquadAI.Get(target).ReactToHostile()
	}
}

// logDeath は死亡・破壊ログを出力する。
// Propは「壊れた」、それ以外は「倒れた」と表示する。
// プレイヤーまたは隊員が関与する場合のみログを出力する
func logDeath(world w.World, target ecs.Entity, source ecs.Entity) {
	isRelevant := isPlayerEntity(source, world) || isPlayerEntity(target, world) ||
		world.Components.SquadMember.Has(target) || world.Components.SquadMember.Has(source)
	if !isRelevant {
		return
	}

	targetName := query.GetEntityName(target, world)

	suffix := " は倒れた。"
	if world.Components.Prop.Has(target) {
		suffix = " は壊れた。"
	}

	gamelog.New(query.GetGameLog(world)).
		Build(func(l *gamelog.Logger) {
			query.AppendNameWithColor(l, target, targetName, world)
		}).
		Append(suffix).
		Log()
}

// isPlayerEntity はエンティティがプレイヤーかを判定する
func isPlayerEntity(entity ecs.Entity, world w.World) bool {
	return world.Components.Player.Has(entity)
}

// ApplyHealing は共通の回復処理を実行する
// target に amount 分のHPを回復させる
// 実際の回復量を返す
func ApplyHealing(world w.World, target ecs.Entity, amount int) int {
	hp := world.Components.HP.Get(target)

	beforeHP := hp.Current
	hp.Current += amount
	if hp.Current > hp.Max {
		hp.Current = hp.Max
	}
	actualHealing := hp.Current - beforeHP

	// 回復エフェクトを生成
	if actualHealing > 0 {
		lifecycle.SpawnVisualEffect(target, gc.NewHealEffect(actualHealing), world)
	}

	return actualHealing
}
