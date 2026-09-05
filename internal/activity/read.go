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

// ReadBehavior は読書アクティビティの実装
type ReadBehavior struct{}

// Info はBehaviorの実装
func (rb *ReadBehavior) Info() Info {
	return Info{
		Name:            "Read",
		Description:     "Read a book to learn skills or recipes",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (rb *ReadBehavior) Name() gc.BehaviorName {
	return gc.BehaviorRead
}

// NewReadActivity は読む本を指定して読書アクティビティを組む。進捗は本の Effort に
// 永続するので、Progress.Max も本の総工数に据えて表示を揃える。
func NewReadActivity(target ecs.Entity, world w.World) *gc.Activity {
	effort := 0
	if book := getBook(target, world); book != nil {
		effort = book.Effort.Max
	}
	comp := NewActivity(gc.BehaviorRead, effort)
	comp.Params = &gc.ReadParams{Target: target}
	return comp
}

// Validate は読書アクティビティの検証を行う
func (rb *ReadBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ReadParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	book := getBook(p.Target, world)
	if book == nil {
		return fmt.Errorf("target has no Book component")
	}

	var skills *gc.Skills
	if world.Components.Skills.Has(actor) {
		skillsComp := world.Components.Skills.Get(actor)
		skills = skillsComp
	}
	if err := book.CanRead(skills); err != nil {
		// 読めない理由ごとに翻訳した文言を組む。CanRead の英語 err はプレイヤーに出さない
		if book.IsCompleted() {
			return &UserError{Msg: query.T(world, "this book is already read")}
		}
		if book.Skill != nil && book.Skill.RequiredLevel > 0 {
			current := 0
			if skills != nil {
				current = skills.Get(book.Skill.TargetSkill).Value
			}
			return &UserError{Msg: query.T(world, "You need %s Lv%d or higher to read this. Current Lv%d",
				query.T(world, gc.SkillName(book.Skill.TargetSkill)), book.Skill.RequiredLevel, current)}
		}
		return fmt.Errorf("read validation failed: %w", err)
	}

	if !isAreaSafe(actor, world) {
		return &UserError{Msg: query.T(world, "cannot read because enemies are nearby")}
	}

	return nil
}

// Start は読書開始時の処理を実行する
func (rb *ReadBehavior) Start(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ReadParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	book := getBook(p.Target, world)
	if book == nil {
		return fmt.Errorf("book component not found")
	}

	name := query.GetEntityName(p.Target, world)
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "started reading \"%s\"", name)).
		Log()

	log.Debug("reading started", "actor", actor, "book", name, "effort", book.Effort.Max)
	return nil
}

// DoTurn は読書アクティビティの1ターン分の処理を実行する。
// スタック統合などで本エンティティが消えている可能性があるため、毎ターン先頭で生存を確認する
func (rb *ReadBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ReadParams)
	if !ok {
		Cancel(comp, "book is not set")
		return ErrParamsTypeMismatch
	}
	if !world.ECS.Alive(p.Target) {
		Cancel(comp, "interrupted because the book disappeared")
		return nil
	}

	// 安全性チェック
	if !isAreaSafe(actor, world) {
		Cancel(comp, "reading interrupted because enemies are nearby")
		return nil
	}

	book := getBook(p.Target, world)
	if book == nil {
		Cancel(comp, "book not found")
		return nil
	}

	// 対応する能力値を1回だけ取得して工数と経験値の両方に使う
	abilityValue := rb.getSkillAbilityValue(book, actor, world)

	// 工数を進める。対応する能力値が高いほど速く読める
	book.Effort.Current += rb.calcEffortPerTurn(book, abilityValue)

	// 効果の適用（毎ターン）
	rb.applyPerTurnEffect(book, actor, world, abilityValue)

	// 進捗を activity 側にも反映して表示を揃える
	comp.Progress.Current = book.Effort.Current

	// 読了チェック
	if book.IsCompleted() {
		Complete(comp)
	}

	return nil
}

// Finish は読書完了時の処理を実行する
func (rb *ReadBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ReadParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !world.ECS.Alive(p.Target) {
		return nil
	}

	book := getBook(p.Target, world)
	name := query.GetEntityName(p.Target, world)

	if book != nil && book.IsCompleted() {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "finished reading \"%s\"", name)).
			Log()

		// 読了した本を消費する
		if err := lifecycle.ChangeItemCount(world, p.Target, -1); err != nil {
			return fmt.Errorf("failed to consume book: %w", err)
		}
	}

	log.Debug("reading finished", "actor", actor, "book", name)
	return nil
}

// Canceled は読書キャンセル時の処理を実行する。
// 本が消えたことによる中断もあるため、名前は本が残っている場合だけ出す
func (rb *ReadBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if world.Components.Player.Has(actor) {
		message := query.T(world, "interrupted reading")
		if p, ok := comp.Params.(*gc.ReadParams); ok && world.ECS.Alive(p.Target) {
			message = query.T(world, "interrupted reading \"%s\"", query.GetEntityName(p.Target, world))
		}
		gamelog.New(query.GetGameLog(world)).
			Markup(message).
			Log()
	}

	log.Debug("reading interrupted", "reason", comp.CancelReason)
	return nil
}

// applyPerTurnEffect は毎ターンの効果を適用する
func (rb *ReadBehavior) applyPerTurnEffect(book *gc.Book, actor ecs.Entity, world w.World, abilityValue int) {
	if book.Skill == nil {
		return
	}
	effect := book.Skill

	// プレイヤーのSkillsコンポーネントを取得
	if !world.Components.Skills.Has(actor) {
		return
	}
	skills := world.Components.Skills.Get(actor)

	s := skills.Get(effect.TargetSkill)

	// 経験値効率を計算
	efficiency := gc.ReadingEfficiency(s.Value, effect.MaxLevel)
	if efficiency <= 0 {
		return
	}

	leveledUp := skill.GainExpScaled(s, abilityValue, efficiency)

	if leveledUp {
		name := gc.SkillName(effect.TargetSkill)
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s skill rose to %d", name, s.Value)).
			Log()
	}
}

// calcEffortPerTurn は1ターンあたりの読書工数を計算する
// 基本工数10に、本のスキルに対応する能力値を加算する
func (rb *ReadBehavior) calcEffortPerTurn(book *gc.Book, abilityValue int) int {
	const baseEffort = 10
	if book.Skill == nil {
		return baseEffort
	}
	return baseEffort + abilityValue
}

// getSkillAbilityValue は本のスキルに対応する能力値を取得する
func (rb *ReadBehavior) getSkillAbilityValue(book *gc.Book, actor ecs.Entity, world w.World) int {
	if book.Skill == nil {
		return 0
	}
	if !world.Components.Abilities.Has(actor) {
		return 0
	}
	abils := world.Components.Abilities.Get(actor)
	ablID := gc.SkillAbilityID(book.Skill.TargetSkill)
	return abils.ValueOf(ablID)
}

// getBook は対象エンティティのBookコンポーネントを取得する
func getBook(entity ecs.Entity, world w.World) *gc.Book {
	if !world.Components.Book.Has(entity) {
		return nil
	}
	return world.Components.Book.Get(entity)
}
