package activity

import (
	"fmt"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 射撃システムの定数
const (
	CoverPenaltyPerObject = 15 // 射線上の遮蔽物1つにつきの命中率ペナルティ(%)
)

// ShootBehavior は射撃アクティビティの実装
type ShootBehavior struct{}

// Info はBehaviorの実装
func (sb *ShootBehavior) Info() Info {
	return Info{
		Name:            "Shoot",
		Description:     "Attack an enemy from range",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (sb *ShootBehavior) Name() gc.BehaviorName {
	return gc.BehaviorShoot
}

// NewShootActivity は射撃対象を指定して射撃アクティビティを組む。
func NewShootActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorShoot, 0)
	comp.Params = &gc.ShootParams{Target: target}
	return comp
}

// Validate は射撃の検証を行う
func (sb *ShootBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ShootParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if world.Components.Dead.Has(actor) {
		return ErrAttackerDead
	}
	// 対象は CanShootTarget が事前に絞る。存在しない・死亡はここでは選べないので、
	// 発火するのは選択後の消失など不変条件違反。システムエラーとして伝播させる
	if !world.Components.GridElement.Has(p.Target) {
		return fmt.Errorf("target has no position")
	}
	if world.Components.Dead.Has(p.Target) {
		return fmt.Errorf("target is already dead")
	}

	// 遠距離武器が装備されているか。CanShootTarget が事前に絞るので、武器スロット不正も
	// 遠距離武器なしもここでは起こらない。発火したら不変条件違反なのでそのまま伝播させる
	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		return err
	}

	// 残弾チェック
	if fire.Magazine <= 0 {
		return fmt.Errorf("out of ammo")
	}

	// 射程・射線チェック
	distance := EntityDistance(actor, p.Target, world)
	rangeParams, rangeOK := gc.GetRangeParams(fire.AttackCategory)
	if !rangeOK {
		return fmt.Errorf("no ranged weapon equipped")
	}
	if distance > float64(rangeParams.MaxRange) {
		return fmt.Errorf("target is out of range")
	}

	// 射線上に壁がないか
	if blocked, _ := checkLineOfSight(actor, p.Target, world); blocked {
		return fmt.Errorf("line of sight is blocked")
	}

	return nil
}

// Start はBehaviorの実装
func (sb *ShootBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	if p, ok := comp.Params.(*gc.ShootParams); ok {
		log.Debug("shoot started", "actor", actor, "target", p.Target)
	}
	return nil
}

// DoTurn は射撃の実行処理
func (sb *ShootBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ShootParams)
	if !ok {
		Cancel(comp, "shoot target is not set")
		return ErrParamsTypeMismatch
	}

	target := p.Target

	// 装備武器を取得
	fire, weaponName, err := getEquippedFire(actor, world)
	if err != nil {
		Cancel(comp, "no ranged weapon equipped")
		return err
	}

	// 弾薬消費
	fire.Magazine--

	// 命中率修正を計算（距離ペナルティ + 遮蔽ペナルティ + 弾薬修正）
	hitModifier := calculateRangedHitModifier(actor, target, fire, world)
	hitModifier += fire.LoadedAccuracyBonus

	// ダメージ適用（共通関数を使用）
	if err := applyAttackDamage(actor, target, world, fire, weaponName, hitModifier, fire.LoadedDamageBonus); err != nil {
		return err
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (sb *ShootBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("shoot finished", "actor", actor)
	return nil
}

// Canceled はBehaviorの実装
func (sb *ShootBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("shoot canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// getEquippedFire はプレイヤーの装備中の遠距離武器のFireと武器名を取得する
func getEquippedFire(actor ecs.Entity, world w.World) (*gc.Fire, string, error) {
	selectedSlot := query.GetWeaponSelection(world).Slot
	weaponIndex := selectedSlot - 1
	if weaponIndex < 0 || weaponIndex >= 5 {
		return nil, "", fmt.Errorf("invalid weapon slot number: %d", selectedSlot)
	}

	weapons := query.GetWeapons(world, actor)
	weaponEntity := weapons[weaponIndex]
	if weaponEntity == nil {
		return nil, "", ErrShootNoFireWeapon
	}

	fire, name, err := query.GetFireFromWeapon(world, *weaponEntity)
	if err != nil {
		return nil, "", ErrShootNoFireWeapon
	}
	return fire, name, nil
}

// calculateRangedHitModifier は距離と遮蔽による命中率修正を計算する
func calculateRangedHitModifier(actor, target ecs.Entity, attack gc.Attacker, world w.World) int {
	modifier := 0

	// 距離ペナルティ
	distance := EntityDistance(actor, target, world)
	rangeParams, ok := gc.GetRangeParams(attack.GetAttackCategory())
	if ok && distance > float64(rangeParams.OptimalRange) {
		excess := int(distance) - rangeParams.OptimalRange
		modifier -= excess * rangeParams.PenaltyPerTile
	}

	// 遮蔽ペナルティ
	_, coverCount := checkLineOfSight(actor, target, world)
	modifier -= coverCount * CoverPenaltyPerObject

	return modifier
}

// EntityDistance は2エンティティ間の距離を返す
func EntityDistance(a, b ecs.Entity, world w.World) float64 {
	if !world.Components.GridElement.Has(a) || !world.Components.GridElement.Has(b) {
		return math.MaxFloat64
	}
	aPos := world.Components.GridElement.Get(a)
	bPos := world.Components.GridElement.Get(b)
	return geometry.Distance(float64(aPos.X), float64(aPos.Y), float64(bPos.X), float64(bPos.Y))
}

// checkLineOfSight は射線上の壁と遮蔽物を1パスでチェックする。
// 壁（BlockView=true）があればblocked=true、遮蔽物（BlockPass=true, BlockView=false）の数をcoverCountで返す
func checkLineOfSight(actor, target ecs.Entity, world w.World) (blocked bool, coverCount int) {
	if !world.Components.GridElement.Has(actor) || !world.Components.GridElement.Has(target) {
		return true, 0
	}
	aPos := world.Components.GridElement.Get(actor)
	tPos := world.Components.GridElement.Get(target)

	points := geometry.BresenhamLine(consts.Coord[int]{X: int(aPos.X), Y: int(aPos.Y)}, consts.Coord[int]{X: int(tPos.X), Y: int(tPos.Y)})
	for _, p := range points {
		entities := query.GetEntitiesAt(world, consts.Tile(p.X), consts.Tile(p.Y))
		for _, e := range entities {
			if world.Components.BlockView.Has(e) {
				return true, coverCount
			}
			if world.Components.BlockPass.Has(e) {
				coverCount++
			}
		}
	}
	return false, coverCount
}

// CanShootTarget はactorからtargetに射撃可能かを判定する。
// 射撃対象選択UIでのフィルタリング用
func CanShootTarget(actor, target ecs.Entity, world w.World) bool {
	comp := NewShootActivity(target)
	return (&ShootBehavior{}).Validate(comp, actor, world) == nil
}

// CalculateShootHitRate は射撃の命中率を計算して返す。情報パネル表示用
func CalculateShootHitRate(actor, target ecs.Entity, world w.World) int {
	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		return 0
	}

	modifier := calculateRangedHitModifier(actor, target, fire, world)
	modifier += fire.LoadedAccuracyBonus

	return calculateHitRate(actor, target, world, fire, modifier)
}
