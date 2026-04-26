package activity

import (
	"fmt"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// 射撃システムの定数
const (
	CoverPenaltyPerObject = 15 // 射線上の遮蔽物1つにつきの命中率ペナルティ(%)
)

// ShootActivity は射撃アクティビティの実装
type ShootActivity struct{}

// Info はBehaviorの実装
func (sa *ShootActivity) Info() Info {
	return Info{
		Name:            "射撃",
		Description:     "遠距離から敵を攻撃する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (sa *ShootActivity) Name() gc.BehaviorName {
	return gc.BehaviorShoot
}

// Validate は射撃の検証を行う
func (sa *ShootActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return ErrAttackTargetNotSet
	}
	if actor.HasComponent(world.Components.Dead) {
		return ErrAttackerDead
	}
	if !comp.Target.HasComponent(world.Components.GridElement) {
		return ErrAttackTargetNotExists
	}
	if comp.Target.HasComponent(world.Components.Dead) {
		return ErrAttackTargetDead
	}

	// 遠距離武器が装備されているか
	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		return err
	}

	// 残弾チェック
	if fire.Magazine <= 0 {
		return ErrShootNoAmmo
	}

	// 射程・射線チェック
	distance := EntityDistance(actor, *comp.Target, world)
	rangeParams, ok := gc.GetRangeParams(fire.AttackCategory)
	if !ok {
		return ErrShootNoFireWeapon
	}
	if distance > float64(rangeParams.MaxRange) {
		return ErrAttackOutOfRange
	}

	// 射線上に壁がないか
	if blocked, _ := checkLineOfSight(actor, *comp.Target, world); blocked {
		return ErrShootLineOfSightBlocked
	}

	return nil
}

// Start はBehaviorの実装
func (sa *ShootActivity) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("射撃開始", "actor", actor, "target", *comp.Target)
	return nil
}

// DoTurn は射撃の実行処理
func (sa *ShootActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		Cancel(comp, "射撃対象が設定されていません")
		return ErrAttackTargetNotSet
	}

	target := *comp.Target

	// 装備武器を取得
	fire, weaponName, err := getEquippedFire(actor, world)
	if err != nil {
		Cancel(comp, "遠距離武器が装備されていません")
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

	// 空腹進行
	progressHunger(actor, world)

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (sa *ShootActivity) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("射撃完了", "actor", actor)
	return nil
}

// Canceled はBehaviorの実装
func (sa *ShootActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("射撃キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// getEquippedFire はプレイヤーの装備中の遠距離武器のFireと武器名を取得する
func getEquippedFire(actor ecs.Entity, world w.World) (*gc.Fire, string, error) {
	selectedSlot := world.Resources.Dungeon.SelectedWeaponSlot
	weaponIndex := selectedSlot - 1
	if weaponIndex < 0 || weaponIndex >= 5 {
		return nil, "", fmt.Errorf("無効な武器スロット番号: %d", selectedSlot)
	}

	weapons := worldhelper.GetWeapons(world, actor)
	weaponEntity := weapons[weaponIndex]
	if weaponEntity == nil {
		return nil, "", ErrShootNoFireWeapon
	}

	fire, name, err := worldhelper.GetFireFromWeapon(world, *weaponEntity)
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
	aGrid := world.Components.GridElement.Get(a)
	bGrid := world.Components.GridElement.Get(b)
	if aGrid == nil || bGrid == nil {
		return math.MaxFloat64
	}
	aPos := aGrid.(*gc.GridElement)
	bPos := bGrid.(*gc.GridElement)
	dx := float64(aPos.X - bPos.X)
	dy := float64(aPos.Y - bPos.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// bresenhamLine はBresenhamアルゴリズムで2点間の全座標を返す。始点と終点は含まない
func bresenhamLine(x0, y0, x1, y1 int) []consts.Coord[int] {
	var points []consts.Coord[int]

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	x, y := x0, y0
	for {
		// 始点と終点は除外
		if (x != x0 || y != y0) && (x != x1 || y != y1) {
			points = append(points, consts.Coord[int]{X: x, Y: y})
		}
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}

	return points
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// checkLineOfSight は射線上の壁と遮蔽物を1パスでチェックする。
// 壁（BlockView=true）があればblocked=true、遮蔽物（BlockPass=true, BlockView=false）の数をcoverCountで返す
func checkLineOfSight(actor, target ecs.Entity, world w.World) (blocked bool, coverCount int) {
	aPos := world.Components.GridElement.Get(actor).(*gc.GridElement)
	tPos := world.Components.GridElement.Get(target).(*gc.GridElement)

	points := bresenhamLine(int(aPos.X), int(aPos.Y), int(tPos.X), int(tPos.Y))
	for _, p := range points {
		entities := worldhelper.GetEntitiesAt(world, consts.Tile(p.X), consts.Tile(p.Y))
		for _, e := range entities {
			if e.HasComponent(world.Components.BlockView) {
				return true, coverCount
			}
			if e.HasComponent(world.Components.BlockPass) {
				coverCount++
			}
		}
	}
	return false, coverCount
}

// CanShootTarget はactorからtargetに射撃可能かを判定する。
// 射撃対象選択UIでのフィルタリング用
func CanShootTarget(actor, target ecs.Entity, world w.World) bool {
	sa := &ShootActivity{}
	comp, err := NewActivity(sa, 1)
	if err != nil {
		return false
	}
	comp.Target = &target
	return sa.Validate(comp, actor, world) == nil
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

// ExecuteShootAction は射撃アクションを実行する
func ExecuteShootAction(actor ecs.Entity, target ecs.Entity, world w.World) error {
	params := ActionParams{
		Actor:  actor,
		Target: &target,
	}
	_, err := Execute(&ShootActivity{}, params, world)
	if err != nil {
		gamelog.New(gamelog.FieldLog).
			Append(err.Error()).
			Log()
		return nil
	}
	return nil
}
