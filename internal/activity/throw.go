package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ThrowBehavior は背負い袋の物を投げて敵に当てるアクティビティの実装
type ThrowBehavior struct{}

// Info はBehaviorの実装
func (tb *ThrowBehavior) Info() Info {
	return Info{
		Name:            "Throw",
		Description:     "Throw an item at an enemy",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (tb *ThrowBehavior) Name() gc.BehaviorName {
	return gc.BehaviorThrow
}

// NewThrowActivity は投げる物と狙う敵を指定して投げるアクティビティを組む
func NewThrowActivity(item, target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorThrow, 0)
	comp.Params = &gc.ThrowParams{Item: item, Target: target}
	return comp
}

// Validate は投げられるかを検証する。物が actor の背負い袋にあり、狙う敵が射程と射線の内にあること
func (tb *ThrowBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ThrowParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if world.Components.Dead.Has(actor) {
		return ErrAttackerDead
	}
	if err := throwItemError(world, actor, p.Item); err != nil {
		return err
	}
	if !world.ECS.Alive(p.Target) || !world.Components.GridElement.Has(p.Target) || world.Components.Dead.Has(p.Target) {
		return errThrowTargetInvalid
	}
	rangeParams, ok := world.Resources.SkillDefs.RangeParams(rawschema.AttackCategoryThrow)
	if !ok {
		return fmt.Errorf("throw attack type has no range")
	}
	if EntityDistance(actor, p.Target, world) > float64(rangeParams.MaxRange) {
		return errThrowOutOfRange
	}
	if blocked, _ := checkLineOfSight(actor, p.Target, world); blocked {
		return errThrowLineBlocked
	}
	return nil
}

// Start はBehaviorの実装
func (tb *ThrowBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	if p, ok := comp.Params.(*gc.ThrowParams); ok {
		log.Debug("throw started", "actor", actor, "item", p.Item, "target", p.Target)
	}
	return nil
}

// DoTurn は物を投げて当てる。当たった物と壊れる物は消え、外れた物は狙った敵の足元へ落ちる
func (tb *ThrowBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ThrowParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	attack, ok := query.ThrowAttackOf(world, p.Item)
	if !ok {
		return errThrowTooHeavy
	}
	// 倒した敵の死亡の処理に頼らないよう、落とす座標は当てる前に取る
	coord := world.Components.GridElement.Get(p.Target).Coord
	var name i18n.Msgid
	if world.Components.Name.Has(p.Item) {
		name = world.Components.Name.Get(p.Item).Name
	}

	// 威力 0 の物は、効果を持つまで攻撃として扱わない。ダメージの式は威力 0 でも能力値と出目で傷を負わせる
	hit := false
	if attack.Damage > 0 {
		hitModifier := calculateRangedHitModifier(actor, p.Target, attack, world)
		var err error
		hit, err = applyAttackDamage(actor, p.Target, world, attack, name, hitModifier, 0)
		if err != nil {
			return err
		}
	}
	if err := landThrown(world, actor, p.Item, coord, hit || attack.Breaks); err != nil {
		return err
	}

	Complete(comp)
	return nil
}

// landThrown は投げた物を手放す。consumed なら1つ消し、そうでなければ coord へ落とす。
// 当たった物と壊れる物は消える。外れた壊れない物は落ち、拾い直せる
func landThrown(world w.World, actor, item ecs.Entity, coord consts.Coord[consts.Tile], consumed bool) error {
	if consumed {
		if err := lifecycle.ChangeItemCount(world, item, -1); err != nil {
			return fmt.Errorf("failed to consume thrown item: %w", err)
		}
		return nil
	}
	lifecycle.MoveMembersToField(world, []ecs.Entity{item}, coord, actor)
	return nil
}

// Finish はBehaviorの実装
func (tb *ThrowBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("throw finished", "actor", actor)
	return nil
}

// Canceled はBehaviorの実装
func (tb *ThrowBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("throw canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// throwItemError は actor が item を投げられなければ理由を返す。物が actor の背負い袋にあり、重すぎないこと
func throwItemError(world w.World, actor, item ecs.Entity) error {
	if !world.ECS.Alive(item) || !world.Components.LocationInBackpack.Has(item) ||
		world.Components.LocationInBackpack.Get(item).Owner != actor {
		return errThrowItemNotInBackpack
	}
	if !query.CanThrow(world, item) {
		return errThrowTooHeavy
	}
	return nil
}

// CanThrowItem は actor が item を投げられるかを返す。狙う敵は問わない
func CanThrowItem(actor, item ecs.Entity, world w.World) bool {
	return throwItemError(world, actor, item) == nil
}

// CanThrowTarget は actor が item を target に投げられるかを判定する。狙いを選ぶ画面の絞り込みに使う
func CanThrowTarget(actor, item, target ecs.Entity, world w.World) bool {
	return (&ThrowBehavior{}).Validate(NewThrowActivity(item, target), actor, world) == nil
}

// CalculateThrowHitRate は投げの命中率を計算して返す。情報パネル表示用
func CalculateThrowHitRate(actor, item, target ecs.Entity, world w.World) int {
	attack, ok := query.ThrowAttackOf(world, item)
	if !ok {
		return 0
	}
	return calculateHitRate(actor, target, world, attack, calculateRangedHitModifier(actor, target, attack, world))
}
