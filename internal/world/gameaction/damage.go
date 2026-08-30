package gameaction

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// dealHP は HP を damage ぶん削り、致死なら Dead を付けて died を返す。
// ログや死因記録は呼び出し側が状況に応じて行う。
func dealHP(world w.World, target ecs.Entity, damage int) (died bool) {
	hp := world.Components.HP.Get(target)

	beforeHP := hp.Current
	hp.Current -= damage
	if hp.Current < 0 {
		hp.Current = 0
	}

	if hp.Current <= 0 && beforeHP > 0 {
		world.Components.Dead.Add(target, &gc.Dead{})
		return true
	}
	return false
}

// ApplyDamage は共通のダメージ処理を実行する
// source から target へダメージを与え、死亡判定とログ出力を行う
func ApplyDamage(world w.World, target ecs.Entity, damage int, source ecs.Entity) {
	died := dealHP(world, target, damage)

	// 被ダメージによる態度変化
	reactToHostileAction(world, target)

	if died {
		logDeath(world, target, source)
	}
}

// ApplyConditionDamage は低体温や病気など発生源のない不調から target の HP を削る。
// 死因ラベルを受け取り、プレイヤーが倒れたら RunStats へ記録する。
// 攻撃者がいないので AI の敵対反応は起こさない。
func ApplyConditionDamage(world w.World, target ecs.Entity, damage int, cause string) {
	died := dealHP(world, target, damage)

	if died {
		if isPlayerEntity(target, world) {
			if rs := query.GetRunStats(world); rs != nil {
				rs.Cause = cause
			}
		}
		// 発生源が無いので source は target 自身を渡す。プレイヤー関与のログ判定はこれで足りる
		logDeath(world, target, target)
	}
}

// reactToHostileAction は被ダメージ時にAIの戦闘方針を変化させる。
// CombatIgnore は反撃のため CombatAttack に遷移する
func reactToHostileAction(world w.World, target ecs.Entity) {
	if world.Components.SoloAI.Has(target) {
		world.Components.SoloAI.Get(target).ReactToHostile()
	}
}

// logDeath は死亡・破壊ログを出力する。
// Fixed は「壊れた」、それ以外は「倒れた」と表示する。
// プレイヤーが関与する場合のみログを出力する
func logDeath(world w.World, target ecs.Entity, source ecs.Entity) {
	isRelevant := isPlayerEntity(source, world) || isPlayerEntity(target, world)
	if !isRelevant {
		return
	}

	targetName := query.GetEntityName(target, world)

	suffix := query.T(world, " was defeated.")
	if world.Components.Fixed.Has(target) {
		suffix = query.T(world, " was destroyed.")
	}

	gamelog.New(query.GetGameLog(world)).
		Markup(query.NameMarkup(target, targetName, world) + suffix).
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
