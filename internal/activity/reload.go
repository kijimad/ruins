package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
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

// Validate はリロードの検証を行う
func (ra *ReloadActivity) Validate(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	weapon, err := getEquippedRangedWeapon(actor, world)
	if err != nil {
		return err
	}

	if weapon.Magazine >= weapon.MagazineSize {
		return ErrReloadNotNeeded
	}

	// 弾薬の在庫チェック
	_, found := worldhelper.FindAmmoInInventory(world, weapon.AmmoTag)
	if !found {
		return ErrReloadNoAmmo
	}

	return nil
}

// Start はリロード開始時の処理
func (ra *ReloadActivity) Start(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	weapon, err := getEquippedRangedWeapon(actor, world)
	if err != nil {
		return err
	}

	// 最大ターン数の見積もり（最低能力の場合）
	maxTurns := (weapon.ReloadEffort + BaseReloadEffort - 1) / BaseReloadEffort
	if maxTurns < 1 {
		maxTurns = 1
	}
	comp.TurnsTotal = maxTurns
	comp.TurnsLeft = maxTurns

	ra.effortAccum = 0

	gamelog.New(gamelog.FieldLog).
		Append("装填を開始した").
		Log()

	return nil
}

// DoTurn はリロードの1ターン分の処理
func (ra *ReloadActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	weapon, err := getEquippedRangedWeapon(actor, world)
	if err != nil {
		Cancel(comp, "遠距離武器が装備されていません")
		return err
	}

	// 1ターンあたりの工数を計算
	effortPerTurn := ra.calcEffortPerTurn(actor, world)
	ra.effortAccum += effortPerTurn

	// 空腹進行
	progressHunger(actor, world)

	comp.TurnsLeft--

	// 工数が目標に達したら装填完了
	if ra.effortAccum >= weapon.ReloadEffort {
		// 装填数を計算（マガジン容量と弾薬在庫の小さい方）
		needed := weapon.MagazineSize - weapon.Magazine
		ammoEntity, found := worldhelper.FindAmmoInInventory(world, weapon.AmmoTag)
		if !found {
			Cancel(comp, "弾薬がなくなった")
			return nil
		}
		ammoItem := world.Components.Item.Get(ammoEntity).(*gc.Item)

		loaded := needed
		if ammoItem.Count < loaded {
			loaded = ammoItem.Count
		}

		// 装填した弾薬の修正値を記録する
		ammoComp := world.Components.Ammo.Get(ammoEntity).(*gc.Ammo)
		weapon.LoadedDamageBonus = ammoComp.DamageBonus
		weapon.LoadedAccuracyBonus = ammoComp.AccuracyBonus

		weapon.Magazine += loaded
		if err := worldhelper.ChangeItemCount(world, ammoEntity, -loaded); err != nil {
			return fmt.Errorf("弾薬の消費に失敗: %w", err)
		}

		gamelog.New(gamelog.FieldLog).
			Append(fmt.Sprintf("装填完了（%d/%d）", weapon.Magazine, weapon.MagazineSize)).
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
func (ra *ReloadActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	gamelog.New(gamelog.FieldLog).
		Append("装填を中断した").
		Log()
	log.Debug("リロードキャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// calcEffortPerTurn は1ターンあたりの装填工数を計算する
func (ra *ReloadActivity) calcEffortPerTurn(actor ecs.Entity, world w.World) int {
	effort := BaseReloadEffort

	// DEXを加算
	abilsComp := world.Components.Abilities.Get(actor)
	if abilsComp != nil {
		abils := abilsComp.(*gc.Abilities)
		effort += abils.Dexterity.Total
	}

	// 武器スキルレベルを加算
	attack, _, err := getAttackParams(actor, world)
	if err == nil && attack != nil {
		skillID, ok := gc.WeaponSkillID(attack.AttackCategory)
		if ok {
			skillsComp := world.Components.Skills.Get(actor)
			if skillsComp != nil {
				skills := skillsComp.(*gc.Skills)
				effort += skills.Get(skillID).Value
			}
		}
	}

	return effort
}

// ExecuteReloadAction はリロードアクションを実行する
func ExecuteReloadAction(actor ecs.Entity, world w.World) error {
	behavior := &ReloadActivity{}
	activity, err := NewActivity(behavior, 1) // 初期値。Startで更新される
	if err != nil {
		return fmt.Errorf("リロードアクティビティの作成に失敗: %w", err)
	}

	if err := behavior.Validate(activity, actor, world); err != nil {
		gamelog.New(gamelog.FieldLog).
			Append(err.Error()).
			Log()
		return nil
	}

	actor.AddComponent(world.Components.Activity, activity)
	return nil
}
