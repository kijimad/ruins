package activity

import (
	"errors"
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

// ReloadBehavior はリロードアクティビティの実装。
// 継続処理は GetBehavior が毎回作るゼロ値インスタンスで回るため、
// 蓄積した装填工数などの進捗はフィールドに持たず gc.Activity 側に持たせる。
type ReloadBehavior struct{}

// Info はBehaviorの実装
func (rb *ReloadBehavior) Info() Info {
	return Info{
		Name:            "Reload",
		Description:     "Load ammunition into a weapon",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (rb *ReloadBehavior) Name() gc.BehaviorName {
	return gc.BehaviorReload
}

// NewReloadActivity は装填アクティビティを組む。必要総工数は装備中の遠距離武器から求める。
// 遠距離武器が無ければ工数0で組み、遠距離武器なしの判定は Validate に委ねる
func NewReloadActivity(actor ecs.Entity, world w.World) *gc.Activity {
	effort := 0
	if fire, _, err := getEquippedFire(actor, world); err == nil {
		effort = fire.ReloadEffort
	}
	return NewActivity(gc.BehaviorReload, effort)
}

// Validate はリロードの検証を行う
func (rb *ReloadBehavior) Validate(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	// 継続アクティビティ共通の周囲安全チェック。敵が近ければ装填を始めさせない
	if !query.IsAreaSafe(world, actor) {
		return &UserError{Msg: query.T(world, "cannot reload because enemies are nearby")}
	}

	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		if errors.Is(err, ErrShootNoFireWeapon) {
			return &UserError{Msg: query.T(world, "no ranged weapon equipped")}
		}
		return err
	}

	if fire.Magazine >= fire.MagazineSize {
		return &UserError{Msg: query.T(world, "reload is not needed")}
	}

	// 弾薬の在庫チェック
	if _, found := query.FindAmmoInInventory(world, fire.AmmoTag); !found {
		return &UserError{Msg: query.T(world, "no ammo")}
	}

	return nil
}

// Start はリロード開始時の処理
func (rb *ReloadBehavior) Start(_ *gc.Activity, _ ecs.Entity, world w.World) error {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "started reloading")).
		Log()
	return nil
}

// DoTurn はリロードの1ターン分の処理
func (rb *ReloadBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 装填中に敵が接近したら中断する。他の継続系と同じ契約
	if !query.IsAreaSafe(world, actor) {
		Cancel(comp, "reload interrupted because enemies are nearby")
		return nil
	}

	fire, _, err := getEquippedFire(actor, world)
	if err != nil {
		Cancel(comp, "no ranged weapon equipped")
		return err
	}

	// 1ターンあたりの工数を計算。能力が高いほど速く装填する
	comp.Progress.Current += rb.calcEffortPerTurn(actor, fire, world)

	// 工数が目標に達したら装填完了
	if comp.Progress.Current >= comp.Progress.Max {
		// 装填数を計算（マガジン容量と弾薬在庫の小さい方）
		needed := fire.MagazineSize - fire.Magazine
		ammoEntity, found := query.FindAmmoInInventory(world, fire.AmmoTag)
		if !found {
			Cancel(comp, "ran out of ammo")
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
			return fmt.Errorf("failed to consume ammo: %w", err)
		}

		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "reload complete (%d/%d)", fire.Magazine, fire.MagazineSize)).
			Log()

		Complete(comp)
		return nil
	}

	return nil
}

// Finish はリロード完了時の処理
func (rb *ReloadBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("reload finished", "actor", actor)
	return nil
}

// Canceled はリロードキャンセル時の処理
func (rb *ReloadBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "interrupted reloading")).
		Log()
	log.Debug("reload canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// calcEffortPerTurn は1ターンあたりの装填工数を計算する
func (rb *ReloadBehavior) calcEffortPerTurn(actor ecs.Entity, fire *gc.Fire, world w.World) int {
	effort := BaseReloadEffort

	// DEXを加算
	if world.Components.Abilities.Has(actor) {
		effort += world.Components.Abilities.Get(actor).Dexterity.Total
	}

	// 武器スキルレベルを加算
	skillID, ok := gc.WeaponSkillID(fire.AttackCategory)
	if ok {
		if world.Components.Skills.Has(actor) {
			effort += world.Components.Skills.Get(actor).Get(skillID).Value
		}
	}

	return effort
}
