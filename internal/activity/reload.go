package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// リロードシステムの定数
const (
	BaseReloadEffort = 10 // 1ターンあたりの基本装填工数
)

// ReloadActivity はリロードアクティビティの実装
type ReloadActivity struct {
	effortAccum int // 蓄積した装填工数
}

// Info はBehaviorの実装
func (ra *ReloadActivity) Info() Info {
	return Info{
		Name:            "装填",
		Description:     "武器に弾薬を装填する",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (ra *ReloadActivity) Name() gc.BehaviorName {
	return gc.BehaviorReload
}

// BuildActivity はBehaviorの実装
func (ra *ReloadActivity) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	comp, err := NewActivity(ra, 1)
	if err != nil {
		return nil, err
	}
	return comp, nil
}

// Validate はリロードの検証を行う
func (ra *ReloadActivity) Validate(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		return err
	}

	if fire.Magazine >= fire.MagazineSize {
		return ErrReloadNotNeeded
	}

	// 弾薬の在庫チェック
	if _, found := query.FindAmmoInInventory(world, fire.AmmoTag); !found {
		return ErrReloadNoAmmo
	}

	return nil
}

// Start はリロード開始時の処理
func (ra *ReloadActivity) Start(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		return err
	}

	// 最大ターン数の見積もり（最低能力の場合）
	maxTurns := max((fire.ReloadEffort+BaseReloadEffort-1)/BaseReloadEffort, 1)
	comp.TurnsTotal = maxTurns
	comp.TurnsLeft = maxTurns

	ra.effortAccum = 0

	gamelog.New(query.GetGameLog(world)).
		Append("装填を開始した").
		Log()

	return nil
}

// DoTurn はリロードの1ターン分の処理
func (ra *ReloadActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		Cancel(comp, "遠距離武器が装備されていません")
		return err
	}

	// 1ターンあたりの工数を計算
	effortPerTurn := ra.calcEffortPerTurn(actor, fire, world)
	ra.effortAccum += effortPerTurn

	// 空腹進行
	progressHunger(actor, world)

	comp.TurnsLeft--

	// 工数が目標に達したら装填完了
	if ra.effortAccum >= fire.ReloadEffort {
		// 装填数を計算（マガジン容量と弾薬在庫の小さい方）
		needed := fire.MagazineSize - fire.Magazine
		ammoEntity, found := query.FindAmmoInInventory(world, fire.AmmoTag)
		if !found {
			Cancel(comp, "弾薬がなくなった")
			return nil
		}
		ammoCount := query.GetEntityCount(world, ammoEntity)

		loaded := min(ammoCount, needed)

		// 装填した弾薬の修正値を記録する
		ammoComp := world.Components.Ammo.Get(ammoEntity)
		fire.LoadedDamageBonus = ammoComp.DamageBonus
		fire.LoadedAccuracyBonus = ammoComp.AccuracyBonus

		fire.Magazine += loaded
		if err := lifecycle.ChangeItemCount(world, ammoEntity, -loaded); err != nil {
			return fmt.Errorf("弾薬の消費に失敗: %w", err)
		}

		gamelog.New(query.GetGameLog(world)).
			Append(fmt.Sprintf("装填完了（%d/%d）", fire.Magazine, fire.MagazineSize)).
			Log()

		Complete(comp)
		return nil
	}

	if comp.TurnsLeft <= 0 {
		Complete(comp)
	}

	return nil
}

// Finish はリロード完了時の処理
func (ra *ReloadActivity) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("リロード完了", "actor", actor)
	return nil
}

// Canceled はリロードキャンセル時の処理
func (ra *ReloadActivity) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	gamelog.New(query.GetGameLog(world)).
		Append("装填を中断した").
		Log()
	log.Debug("リロードキャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// calcEffortPerTurn は1ターンあたりの装填工数を計算する
func (ra *ReloadActivity) calcEffortPerTurn(actor ecs.Entity, fire *gc.Fire, world w.World) int {
	effort := BaseReloadEffort

	// DEXを加算
	abilsComp := world.Components.Abilities.Get(actor)
	if abilsComp != nil {
		abils := abilsComp.(*gc.Abilities)
		effort += abils.Dexterity.Total
	}

	// 武器スキルレベルを加算
	skillID, ok := gc.WeaponSkillID(fire.AttackCategory)
	if ok {
		skillsComp := world.Components.Skills.Get(actor)
		if skillsComp != nil {
			skills := skillsComp.(*gc.Skills)
			effort += skills.Get(skillID).Value
		}
	}

	return effort
}

// ExecuteReloadAction はリロードアクションを実行する
func ExecuteReloadAction(actor ecs.Entity, world w.World) error {
	_, err := Execute(&ReloadActivity{}, actor, world)
	if err != nil {
		return err
	}
	return nil
}
