package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ThrowBehavior はバックパックの物を投げて敵に当てるアクティビティの実装
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

// Validate は投げられるかを検証する。物が actor のバックパックにあり、狙う敵が射程と射線の内にあること
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

// DoTurn は物を投げて当てる。投げた物は当たっても外れても無くなる
func (tb *ThrowBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ThrowParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	attack, ok := query.ThrowAttackOf(world, p.Item)
	if !ok {
		return errThrowTooHeavy
	}
	var name i18n.Msgid
	if world.Components.Name.Has(p.Item) {
		name = world.Components.Name.Get(p.Item).Name
	}

	switch {
	case attack.Blast != nil:
		if err := explode(world, actor, p.Target, attack, name); err != nil {
			return err
		}
	case attack.Damage > 0:
		hitModifier := calculateRangedHitModifier(actor, p.Target, attack, world)
		if err := applyAttackDamage(actor, p.Target, world, gc.InvalidEntity, attack, name, hitModifier, 0); err != nil {
			return err
		}
	default:
		// 威力 0 の物は、効果を持つまで攻撃として扱わない。ダメージの式は威力 0 でも能力値と出目で傷を負わせる
		logThrow(world, actor, p.Item)
	}
	// 投げた物は、当たっても外れても無くなる
	if err := lifecycle.ChangeItemCount(world, p.Item, -1); err != nil {
		return fmt.Errorf("failed to consume thrown item: %w", err)
	}

	Complete(comp)
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

// explode は target のマスで爆ぜさせ、そこから半径のマスにいる者すべてに効かせる。命中の判定も会心もしない。
// 怪我は抽選しない。範囲の全員に抽選すると強すぎるので、破片の傷は爆風のダメージに含める。
// 手に構えた道具は爆風に関わらないので、置物へのダメージと廃材に効かせない。
// 投げた本人も範囲にいれば巻き込む。投擲のスキルは、本人と随伴するキューブ以外を傷つけたときに1回だけ伸ばす
func explode(world w.World, actor, target ecs.Entity, attack gc.ThrowAttack, name i18n.Msgid) error {
	center := world.Components.GridElement.Get(target).Coord
	hurtOther := false
	for _, victim := range blastVictims(world, center, attack.Blast.Radius) {
		if world.Components.Dead.Has(victim) {
			continue
		}
		if attack.Damage > 0 {
			damage, err := applyHit(actor, victim, world, attack, name, hitOptions{})
			if err != nil {
				return err
			}
			if damage > 0 && victim != actor && !world.Components.Cube.Has(victim) {
				hurtOther = true
			}
		}
		// 爆風で倒れた者には不調を付けない
		if attack.Blast.Condition != "" && !world.Components.Dead.Has(victim) {
			infect(world, victim, attack.Blast.Condition)
		}
	}
	// 自分の爆風で倒れた本人のスキルは伸ばさない
	if hurtOther && !world.Components.Dead.Has(actor) {
		growWeaponSkill(actor, world, attack)
	}
	return nil
}

// blastVictims は center から縦横斜めに radius マス以内にいる、HP を持ち死んでいない者を集める。壁は爆風を遮らない
func blastVictims(world w.World, center consts.Coord[consts.Tile], radius int) []ecs.Entity {
	var victims []ecs.Entity
	q := query.ActiveFilter2[gc.GridElement, gc.HP](world).Without(ecs.C[gc.Dead]()).Query()
	for q.Next() {
		grid, _ := q.Get()
		if geometry.ChebyshevDistance(grid.Coord, center) <= radius {
			victims = append(victims, q.Entity())
		}
	}
	return victims
}

// infect は victim に不調を付け、プレイヤー以外がかかったらログに出す。プレイヤーには ContractIllness がログを出す
func infect(world w.World, victim ecs.Entity, ct gc.ConditionType) {
	if !gameaction.ContractIllness(world, victim, ct) || world.Components.Player.Has(victim) {
		return
	}
	victimMarkup := query.NameMarkup(victim, query.GetEntityName(victim, world), world)
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s contracted %s.", victimMarkup, query.ConditionName(world, ct))).
		Log()
}

// logThrow は攻撃にならない物を投げたことをログに出す。プレイヤーが投げたときだけ出す
func logThrow(world w.World, actor, item ecs.Entity) {
	if !world.Components.Player.Has(actor) {
		return
	}
	actorMarkup := query.NameMarkup(actor, query.GetEntityName(actor, world), world)
	itemMarkup := gamelog.Tag("item", query.GetEntityName(item, world))
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s threw %s.", actorMarkup, itemMarkup)).
		Log()
}

// throwItemError は actor が item を投げられなければ理由を返す。物が actor のバックパックにあり、重すぎないこと
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

// CalculateThrowHitRate は投げの値 attack で target を狙ったときの命中率を返す。情報パネル表示用。
// 爆ぜる物は命中の判定をせず必ず狙ったマスで爆ぜるので 100 を返す
func CalculateThrowHitRate(actor, target ecs.Entity, attack gc.ThrowAttack, world w.World) int {
	if attack.Blast != nil {
		return 100
	}
	return calculateHitRate(actor, target, world, attack, calculateRangedHitModifier(actor, target, attack, world))
}
