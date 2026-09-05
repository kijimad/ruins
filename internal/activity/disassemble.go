package activity

import (
	"fmt"
	"math"
	"slices"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/skill"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// DisassembleBehavior は工具でpropやアイテムを分解して素材を得るアクティビティの実装。
// 工具は開始時に固定せず、毎回actorの所持品から分類に適合する最良の1つを解決する。
// エンティティ参照を持ち越さないのでセーブ互換の考慮が不要になり、
// 途中で工具を失った場合も次のターン検査で自然に中断へ落ちる
type DisassembleBehavior struct{}

// Info はBehaviorの実装
func (db *DisassembleBehavior) Info() Info {
	return Info{
		Name:            "Disassemble",
		Description:     "Disassemble a target with a tool to obtain materials",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (db *DisassembleBehavior) Name() gc.BehaviorName {
	return gc.BehaviorDisassemble
}

// NewDisassembleActivity は分解対象を指定して分解アクティビティを組む。
// 必要APは対象のbaseAPに機械スキルと工具グレードの短縮を掛けて求める。
func NewDisassembleActivity(target, actor ecs.Entity, world w.World) *gc.Activity {
	requiredAP := 0
	if def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(target, world)); ok {
		if grade, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory); ok {
			requiredAP = RequiredDisassemblyAP(def.BaseAP, mechanicSkillValue(actor, world), grade)
		}
	}
	comp := NewActivity(gc.BehaviorDisassemble, requiredAP)
	comp.Params = &gc.DisassembleParams{Target: target}
	return comp
}

// Validate は分解アクティビティの検証を行う
func (db *DisassembleBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !world.ECS.Alive(p.Target) {
		return fmt.Errorf("target does not exist")
	}
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(p.Target, world))
	if !ok {
		return fmt.Errorf("target has no disassembly definition")
	}
	if _, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory); !ok {
		return &UserError{Msg: query.T(world, "Do not have a tool to disassemble %s", gamelog.Tag("item", query.GetEntityName(p.Target, world)))}
	}
	if !IsAreaSafe(actor, world) {
		return &UserError{Msg: query.T(world, "cannot disassemble because enemies are nearby")}
	}
	return nil
}

// Start は分解開始時の処理を実行する
func (db *DisassembleBehavior) Start(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(p.Target, world))
	if !ok {
		return fmt.Errorf("disassembly definition not found")
	}
	_, toolName, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory)
	if !ok {
		return fmt.Errorf("does not have the tool required for disassembly")
	}

	name := query.GetEntityName(p.Target, world)
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s began disassembling %s", gamelog.Tag("item", toolName), gamelog.Tag("item", name))).
		Log()

	log.Debug("disassemble started", "actor", actor, "target", name, "tool", toolName)
	return nil
}

// DoTurn は分解アクティビティの1ターン分の処理を実行する。
// 対象が消えている可能性があるため、毎ターン先頭で生存を確認する
func (db *DisassembleBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		Cancel(comp, "disassembly target is not set")
		return ErrParamsTypeMismatch
	}
	if !world.ECS.Alive(p.Target) {
		Cancel(comp, "interrupted because the disassembly target disappeared")
		return nil
	}
	if !IsAreaSafe(actor, world) {
		Cancel(comp, "disassembly interrupted because enemies are nearby")
		return nil
	}
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(p.Target, world))
	if !ok {
		Cancel(comp, "interrupted because the disassembly definition was not found")
		return nil
	}
	if _, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory); !ok {
		Cancel(comp, "disassembly interrupted because the tool was lost")
		return nil
	}

	// 今ターンのAPを注ぐ。APが高いほど速く分解が進む
	comp.Progress.Current += perTurnAP(actor, world)
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}
	return nil
}

// Finish は分解完了時の処理を実行する。産出を抽選し、propは足元へ落として
// エンティティを除去、アイテムは1個消費して所持品へ加える
func (db *DisassembleBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	target := p.Target
	if !world.ECS.Alive(target) {
		return nil
	}
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(target, world))
	if !ok {
		return fmt.Errorf("disassembly definition not found")
	}
	grade, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory)
	if !ok {
		return fmt.Errorf("does not have the tool required for disassembly")
	}

	name := query.GetEntityName(target, world)
	stacks, err := lifecycle.RollDisassemblyYields(world.Resources.Config.RNG, def, mechanicSkillValue(actor, world), grade, true)
	if err != nil {
		return fmt.Errorf("failed to roll disassembly yields: %w", err)
	}

	if world.Components.Fixed.Has(target) && world.Components.GridElement.Has(target) {
		// 座標は除去前に値で控える
		coord := world.Components.GridElement.Get(target).Coord
		// 収納propの場合は中身を足元へ出してから取り壊す
		lifecycle.SpillStorageItems(world, target, coord.X, coord.Y)
		world.ECS.RemoveEntity(target)
		query.InvalidateSpatialIndex(world)
		if err := lifecycle.SpawnDisassemblyYields(world, stacks, coord.X, coord.Y); err != nil {
			return fmt.Errorf("failed to spawn disassembly yields: %w", err)
		}
	} else {
		if err := lifecycle.ChangeItemCount(world, target, -1); err != nil {
			return fmt.Errorf("failed to consume disassembly target: %w", err)
		}
		for _, s := range stacks {
			if _, err := lifecycle.SpawnBackpackItem(world, s.Name, s.Count); err != nil {
				return fmt.Errorf("failed to spawn disassembly yields: %w", err)
			}
		}
	}

	targetMarkup := gamelog.Tag("item", name)
	logger := gamelog.New(query.GetGameLog(world))
	if len(stacks) == 0 {
		logger.Markup(query.T(world, "Disassembled %s but obtained nothing.", targetMarkup))
	} else {
		logger.Markup(query.T(world, "Disassembled %s and obtained %s.", targetMarkup, yieldsMarkup(stacks, world)))
	}
	logger.Log()

	db.gainMechanicExp(actor, world)

	log.Debug("disassemble finished", "actor", actor, "target", name, "yields", stacks)
	return nil
}

// Canceled は分解キャンセル時の処理を実行する。
// 対象消滅による中断もあるため、名前は対象が生きている場合だけ出す
func (db *DisassembleBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if world.Components.Player.Has(actor) {
		logger := gamelog.New(query.GetGameLog(world))
		if p, ok := comp.Params.(*gc.DisassembleParams); ok && world.ECS.Alive(p.Target) {
			logger.Markup(query.T(world, "Interrupted disassembling %s", gamelog.Tag("item", query.GetEntityName(p.Target, world))))
		} else {
			logger.Markup(query.T(world, "Interrupted disassembly"))
		}
		logger.Log()
	}

	log.Debug("disassemble interrupted", "reason", comp.CancelReason)
	return nil
}

// gainMechanicExp は分解完了で機械スキルの経験値を与える
func (db *DisassembleBehavior) gainMechanicExp(actor ecs.Entity, world w.World) {
	if !world.Components.Skills.Has(actor) {
		return
	}
	s := world.Components.Skills.Get(actor).Get(gc.SkillMechanic)

	abilityValue := 0
	if world.Components.Abilities.Has(actor) {
		abilityValue = world.Components.Abilities.Get(actor).ValueOf(gc.SkillAbilityID(gc.SkillMechanic))
	}

	if skill.GainExp(s, abilityValue) {
		// 同一ターン内の別処理が既にマーカーを付けていることがあるため、二重付与を避ける
		if !world.Components.StatsChanged.Has(actor) {
			world.Components.StatsChanged.Add(actor, &gc.StatsChanged{})
		}
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s skill rose to %d", gc.SkillName(gc.SkillMechanic), s.Value)).
			Log()
	}
}

// RequiredDisassemblyAP は分解に必要な総APを計算する。
// 機械スキル1につき1%短縮し下限50%。工具グレードは1で等倍、2で80%、3で60%
func RequiredDisassemblyAP(baseAP int, skillValue int, toolGrade int) int {
	skillFactor := 1.0 - float64(skillValue)*0.01
	if skillFactor < 0.5 {
		skillFactor = 0.5
	}
	gradeFactor := 1.0
	switch toolGrade {
	case 2:
		gradeFactor = 0.8
	case 3:
		gradeFactor = 0.6
	}
	return int(math.Ceil(float64(baseAP) * skillFactor * gradeFactor))
}

// FindBestDisassemblyTool はactorの所持品から分類に適合する最良の分解工具を探す。
// 見つかったらグレードと工具名を返す。同グレードは先に見つかったほうを保つ
func FindBestDisassemblyTool(world w.World, actor ecs.Entity, category oapi.ToolCategory) (int, string, bool) {
	bestGrade := 0
	bestName := ""

	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		itemEntity := q.Entity()
		if world.Components.LocationInBackpack.Get(itemEntity).Owner != actor {
			continue
		}
		tool, ok := raw.FindDisassemblyTool(world.Resources.RawMaster, query.GetEntityID(itemEntity, world))
		if !ok || !slices.Contains(tool.Categories, category) {
			continue
		}
		if tool.Grade > bestGrade {
			bestGrade = tool.Grade
			bestName = query.GetEntityName(itemEntity, world)
		}
	}

	return bestGrade, bestName, bestGrade > 0
}

// mechanicSkillValue はactorの機械スキル値を返す。スキルを持たなければ0
func mechanicSkillValue(actor ecs.Entity, world w.World) int {
	if !world.Components.Skills.Has(actor) {
		return 0
	}
	return world.Components.Skills.Get(actor).Get(gc.SkillMechanic).Value
}

// appendYields は産出一覧をアイテム名の色付きでログへ追記する。
// 産出は件数可変で各アイテム名に個別の色が付くため、単一の書式テンプレートには畳めない。
// 読点区切りで名前を色付き Segment として並べ、末尾に「を得た」の trailing clause を付ける
// yieldsMarkup は産出一覧を色付きマークアップ文字列にして返す。各アイテムを読点で連結する。
// 語尾や句読点は呼び出し側のテンプレートに委ね、ここは中身の並びだけを組む
func yieldsMarkup(stacks []lifecycle.YieldStack, world w.World) string {
	parts := make([]string, len(stacks))
	for i, s := range stacks {
		parts[i] = gamelog.Tag("item", s.Name) + fmt.Sprintf(" x%d", s.Count)
	}
	return strings.Join(parts, query.T(world, ", "))
}
