package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/skill"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ReadActivity は読書アクティビティの実装
type ReadActivity struct {
	Target   ecs.Entity
	Duration int
}

// Info はBehaviorの実装
func (ra *ReadActivity) Info() Info {
	return Info{
		Name:            "読書",
		Description:     "本を読んでスキルやレシピを習得する",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (ra *ReadActivity) Name() gc.BehaviorName {
	return gc.BehaviorRead
}

// BuildActivity はBehaviorの実装
func (ra *ReadActivity) BuildActivity(actor ecs.Entity, world w.World) (*gc.Activity, error) {
	duration := ra.Duration
	if duration <= 0 {
		characterAP, err := getEntityMaxAP(actor, world)
		if err != nil {
			return nil, err
		}
		duration = CalculateRequiredTurns(ra, characterAP)
	}
	comp, err := NewActivity(ra, duration)
	if err != nil {
		return nil, err
	}
	comp.Target = &ra.Target
	return comp, nil
}

// Validate は読書アクティビティの検証を行う
func (ra *ReadActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("本が指定されていません")
	}

	book := ra.getBook(*comp.Target, world)
	if book == nil {
		return fmt.Errorf("対象はBookコンポーネントを持っていません")
	}

	var skills *gc.Skills
	if world.Components.Skills.Has(actor) {
		skillsComp := world.Components.Skills.Get(actor)
		skills = skillsComp
	}
	if err := book.CanRead(skills); err != nil {
		return err
	}

	if !isAreaSafe(actor, world) {
		return fmt.Errorf("周囲に敵がいるため読書できません")
	}

	return nil
}

// Start は読書開始時の処理を実行する
func (ra *ReadActivity) Start(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return ErrReadTargetNotSet
	}

	book := ra.getBook(*comp.Target, world)
	if book == nil {
		return fmt.Errorf("Bookコンポーネントが見つかりません")
	}

	name := query.GetEntityName(*comp.Target, world)
	gamelog.New(query.GetGameLog(world)).
		Append(fmt.Sprintf("「%s」を読み始めた", name)).
		Log()

	log.Debug("読書開始", "actor", actor, "book", name, "effort", book.Effort.Max)
	return nil
}

// DoTurn は読書アクティビティの1ターン分の処理を実行する
func (ra *ReadActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 安全性チェック
	if !isAreaSafe(actor, world) {
		Cancel(comp, "周囲に敵がいるため読書を中断")
		return nil
	}

	book := ra.getBook(*comp.Target, world)
	if book == nil {
		Cancel(comp, "本が見つかりません")
		return nil
	}

	// 対応する能力値を1回だけ取得して工数と経験値の両方に使う
	abilityValue := ra.getSkillAbilityValue(book, actor, world)

	// 工数を進める。対応する能力値が高いほど速く読める
	book.Effort.Current += ra.calcEffortPerTurn(book, abilityValue)

	// 空腹進行
	progressHunger(actor, world)

	// 効果の適用（毎ターン）
	ra.applyPerTurnEffect(book, actor, world, abilityValue)

	// ターン進行
	comp.TurnsLeft--

	// 読了チェック
	if book.IsCompleted() {
		Complete(comp)
		return nil
	}

	if comp.TurnsLeft <= 0 {
		Complete(comp)
	}

	return nil
}

// Finish は読書完了時の処理を実行する
func (ra *ReadActivity) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	book := ra.getBook(*comp.Target, world)
	name := query.GetEntityName(*comp.Target, world)

	if book != nil && book.IsCompleted() {
		gamelog.New(query.GetGameLog(world)).
			Append(fmt.Sprintf("「%s」を読了した", name)).
			Log()

		// 読了した本を消費する
		if err := lifecycle.ChangeItemCount(world, *comp.Target, -1); err != nil {
			return fmt.Errorf("本の消費に失敗: %w", err)
		}
	}

	log.Debug("読書完了", "actor", actor, "book", name)
	return nil
}

// Canceled は読書キャンセル時の処理を実行する
func (ra *ReadActivity) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	name := query.GetEntityName(*comp.Target, world)

	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Append(fmt.Sprintf("「%s」の読書を中断した", name)).
			Log()
	}

	log.Debug("読書中断", "reason", comp.CancelReason, "book", name)
	return nil
}

// applyPerTurnEffect は毎ターンの効果を適用する
func (ra *ReadActivity) applyPerTurnEffect(book *gc.Book, actor ecs.Entity, world w.World, abilityValue int) {
	if book.Skill == nil {
		return
	}
	effect := book.Skill

	// プレイヤーのSkillsコンポーネントを取得
	if !world.Components.Skills.Has(actor) {
		return
	}
	skillsComp := world.Components.Skills.Get(actor)
	skills := skillsComp

	s := skills.Get(effect.TargetSkill)

	// 経験値効率を計算
	efficiency := gc.ReadingEfficiency(s.Value, effect.MaxLevel)
	if efficiency <= 0 {
		return
	}

	leveledUp := skill.GainExpScaled(s, abilityValue, efficiency)

	// スキルアップした場合はCharModifiers再計算
	if leveledUp {
		world.Components.StatsChanged.Add(actor, &gc.StatsChanged{})

		name := gc.SkillName(effect.TargetSkill)
		gamelog.New(query.GetGameLog(world)).
			Append(fmt.Sprintf("%sスキルが %d に上がった", name, s.Value)).
			Log()
	}
}

// calcEffortPerTurn は1ターンあたりの読書工数を計算する
// 基本工数10に、本のスキルに対応する能力値を加算する
func (ra *ReadActivity) calcEffortPerTurn(book *gc.Book, abilityValue int) int {
	const baseEffort = 10
	if book.Skill == nil {
		return baseEffort
	}
	return baseEffort + abilityValue
}

// getSkillAbilityValue は本のスキルに対応する能力値を取得する
func (ra *ReadActivity) getSkillAbilityValue(book *gc.Book, actor ecs.Entity, world w.World) int {
	if book.Skill == nil {
		return 0
	}
	if !world.Components.Abilities.Has(actor) {
		return 0
	}
	abilsComp := world.Components.Abilities.Get(actor)
	abils := abilsComp
	ablID := gc.SkillAbilityID(book.Skill.TargetSkill)
	return abils.ValueOf(ablID)
}

// getBook は対象エンティティのBookコンポーネントを取得する
func (ra *ReadActivity) getBook(entity ecs.Entity, world w.World) *gc.Book {
	if !world.Components.Book.Has(entity) {
		return nil
	}
	comp := world.Components.Book.Get(entity)
	return comp
}
