package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// useEffect は使用時に発動する効果1種。アイテムがその効果 component を持つときだけ働く。
// 効果を1つ足すときは useEffect を実装して useEffects へ1行足す。DoTurn/Validate/hasEffect の
// 本体は触らない。効果の一覧はこのスライスが単一の真実で、使用可否・提示・適用がここから導かれる
type useEffect interface {
	// applies はアイテムがこの効果を持つかを返す
	applies(world w.World, item ecs.Entity) bool
	// check は使う前の検証。使えない理由があれば *UserError を返す。既定は問題なし
	check(u *UseItemBehavior, actor, item ecs.Entity, world w.World) error
	// apply は効果を適用する
	apply(u *UseItemBehavior, comp *gc.Activity, actor, item ecs.Entity, world w.World) error
}

// useEffects は使用時に働く効果の一覧。順に applies を見て、持つ効果だけ適用する
var useEffects = []useEffect{healEffect{}, nutritionEffect{}, damageEffect{}, remedyEffect{}}

// healEffect は HP を回復する
type healEffect struct{}

func (healEffect) applies(world w.World, item ecs.Entity) bool {
	return world.Components.ProvidesHealing.Has(item)
}
func (healEffect) check(_ *UseItemBehavior, _, _ ecs.Entity, _ w.World) error { return nil }
func (healEffect) apply(u *UseItemBehavior, comp *gc.Activity, actor, item ecs.Entity, world w.World) error {
	return u.applyHealing(comp, actor, world, world.Components.ProvidesHealing.Get(item), item)
}

// nutritionEffect は空腹度を回復する
type nutritionEffect struct{}

func (nutritionEffect) applies(world w.World, item ecs.Entity) bool {
	return world.Components.ProvidesNutrition.Has(item)
}
func (nutritionEffect) check(_ *UseItemBehavior, _, _ ecs.Entity, _ w.World) error { return nil }
func (nutritionEffect) apply(u *UseItemBehavior, comp *gc.Activity, actor, item ecs.Entity, world w.World) error {
	return u.applyNutrition(comp, actor, world, world.Components.ProvidesNutrition.Get(item).Amount, item)
}

// damageEffect は使用者に自傷ダメージを与える
type damageEffect struct{}

func (damageEffect) applies(world w.World, item ecs.Entity) bool {
	return world.Components.InflictsDamage.Has(item)
}
func (damageEffect) check(_ *UseItemBehavior, _, _ ecs.Entity, _ w.World) error { return nil }
func (damageEffect) apply(_ *UseItemBehavior, _ *gc.Activity, actor, item ecs.Entity, world w.World) error {
	gameaction.ApplyDamage(world, actor, world.Components.InflictsDamage.Get(item).Amount, actor)
	return nil
}

// remedyEffect は不調を治療する
type remedyEffect struct{}

func (remedyEffect) applies(world w.World, item ecs.Entity) bool {
	return world.Components.Remedy.Has(item)
}

// check は治療専用アイテムが治せる不調を今持たないなら使用を弾く。空振りを使う前に止める。
// 併用効果を持つアイテムは他の効果で使えるので弾かない
func (remedyEffect) check(u *UseItemBehavior, actor, item ecs.Entity, world w.World) error {
	if !u.isRemedyOnly(world, item) {
		return nil
	}
	remedy := world.Components.Remedy.Get(item)
	if !world.Components.HealthStatus.Has(actor) || mostSevereTreatable(world.Components.HealthStatus.Get(actor), remedy) == nil {
		return &UserError{Msg: query.T(world, "Nothing to treat.")}
	}
	return nil
}
func (remedyEffect) apply(u *UseItemBehavior, _ *gc.Activity, actor, item ecs.Entity, world w.World) error {
	u.applyRemedy(actor, world, world.Components.Remedy.Get(item), item)
	return nil
}
