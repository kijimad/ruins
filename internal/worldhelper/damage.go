package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ApplyDamage は共通のダメージ処理を実行する
// source から target へダメージを与え、死亡判定とログ出力を行う
func ApplyDamage(world w.World, target ecs.Entity, damage int, source ecs.Entity) {
	pools := world.Components.Pools.Get(target).(*gc.Pools)

	beforeHP := pools.HP.Current
	pools.HP.Current -= damage
	if pools.HP.Current < 0 {
		pools.HP.Current = 0
	}

	// ダメージログ出力（プレイヤー関連の場合のみ）
	if IsPlayerEntity(source, world) || IsPlayerEntity(target, world) {
		logDamageDealt(world, source, target, damage)
	}

	// 死亡チェック
	if pools.HP.Current <= 0 && beforeHP > 0 {
		target.AddComponent(world.Components.Dead, &gc.Dead{})
		logDeath(world, target, source)
	}
}

// logDamageDealt はダメージログを出力する
func logDamageDealt(world w.World, source ecs.Entity, target ecs.Entity, damage int) {
	sourceName := GetEntityName(source, world)
	targetName := GetEntityName(target, world)

	logger := gamelog.New(gamelog.FieldLog)
	logger.Build(func(l *gamelog.Logger) {
		AppendNameWithColor(l, source, sourceName, world)
	}).Append(" は ").Build(func(l *gamelog.Logger) {
		AppendNameWithColor(l, target, targetName, world)
	}).Append(fmt.Sprintf(" に %d のダメージを与えた。", damage)).Log()
}

// logDeath は死亡ログを出力する
func logDeath(world w.World, target ecs.Entity, source ecs.Entity) {
	// プレイヤー関連の場合のみログ出力
	if !IsPlayerEntity(source, world) && !IsPlayerEntity(target, world) {
		return
	}

	targetName := GetEntityName(target, world)

	gamelog.New(gamelog.FieldLog).
		Build(func(l *gamelog.Logger) {
			AppendNameWithColor(l, target, targetName, world)
		}).
		Append(" は倒れた。").
		Log()
}

// GetEntityName はエンティティの名前を取得する
func GetEntityName(entity ecs.Entity, world w.World) string {
	name := world.Components.Name.Get(entity)
	if name != nil {
		return name.(*gc.Name).Name
	}
	return "Unknown"
}

// AppendNameWithColor はエンティティの種類に応じて色付きで名前を追加する
func AppendNameWithColor(logger *gamelog.Logger, entity ecs.Entity, name string, world w.World) {
	if entity.HasComponent(world.Components.Player) {
		logger.PlayerName(name)
	} else if entity.HasComponent(world.Components.AIMoveFSM) {
		logger.NPCName(name)
	} else {
		logger.Append(name)
	}
}

// IsPlayerEntity はエンティティがプレイヤーかを判定する
func IsPlayerEntity(entity ecs.Entity, world w.World) bool {
	return entity.HasComponent(world.Components.Player)
}

// ApplyHealing は共通の回復処理を実行する
// target に amount 分のHPを回復させる
// 実際の回復量を返す
func ApplyHealing(world w.World, target ecs.Entity, amount int) int {
	pools := world.Components.Pools.Get(target).(*gc.Pools)

	beforeHP := pools.HP.Current
	pools.HP.Current += amount
	if pools.HP.Current > pools.HP.Max {
		pools.HP.Current = pools.HP.Max
	}
	actualHealing := pools.HP.Current - beforeHP

	return actualHealing
}
