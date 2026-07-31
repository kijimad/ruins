package activity

import (
	"fmt"
	"math"
	"slices"

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

// DisassembleActivity は工具でpropやアイテムを分解して素材を得るアクティビティの実装。
// 工具は開始時に固定せず、毎回actorの所持品から分類に適合する最良の1つを解決する。
// エンティティ参照を持ち越さないのでセーブ互換の考慮が不要になり、
// 途中で工具を失った場合も次のターン検査で自然に中断へ落ちる
type DisassembleActivity struct {
	Target ecs.Entity
}

// Info はBehaviorの実装
func (da *DisassembleActivity) Info() Info {
	return Info{
		Name:            "分解",
		Description:     "工具で対象を分解して素材を得る",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (da *DisassembleActivity) Name() gc.BehaviorName {
	return gc.BehaviorDisassemble
}

// BuildActivity はBehaviorの実装。
// 必要ターン数は固定値でなく、対象のbaseAPへ機械スキルと工具グレードの短縮を
// 掛けた総APから毎回計算する
func (da *DisassembleActivity) BuildActivity(actor ecs.Entity, world w.World) (*gc.Activity, error) {
	def, ok := findDisassemblyDef(da.Target, world)
	if !ok {
		return nil, fmt.Errorf("対象は分解定義を持っていません")
	}
	grade, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory)
	if !ok {
		return nil, fmt.Errorf("分解に必要な工具を持っていません")
	}

	requiredAP := RequiredDisassemblyAP(int(def.BaseAP), mechanicSkillValue(actor, world), grade)
	characterAP, err := getEntityMaxAP(actor, world)
	if err != nil {
		return nil, err
	}
	duration := consts.Turn(1)
	if characterAP > 0 {
		duration = consts.Turn((requiredAP + characterAP - 1) / characterAP)
	}

	comp, err := NewActivity(da, duration)
	if err != nil {
		return nil, err
	}
	comp.Target = &da.Target
	return comp, nil
}

// Validate は分解アクティビティの検証を行う
func (da *DisassembleActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("分解対象が指定されていません")
	}
	if !world.ECS.Alive(*comp.Target) {
		return fmt.Errorf("分解対象が存在しません")
	}
	def, ok := findDisassemblyDef(*comp.Target, world)
	if !ok {
		return fmt.Errorf("対象は分解定義を持っていません")
	}
	if _, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory); !ok {
		return fmt.Errorf("分解に必要な工具を持っていません")
	}
	if !isAreaSafe(actor, world) {
		return fmt.Errorf("周囲に敵がいるため分解できません")
	}
	return nil
}

// Start は分解開始時の処理を実行する
func (da *DisassembleActivity) Start(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	def, ok := findDisassemblyDef(*comp.Target, world)
	if !ok {
		return fmt.Errorf("分解定義が見つかりません")
	}
	_, toolName, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory)
	if !ok {
		return fmt.Errorf("分解に必要な工具を持っていません")
	}

	name := query.GetEntityName(*comp.Target, world)
	gamelog.New(query.GetGameLog(world)).
		ItemName(toolName).
		Append("で").
		ItemName(name).
		Append("の分解を始めた").
		Log()

	log.Debug("分解開始", "actor", actor, "target", name, "tool", toolName)
	return nil
}

// DoTurn は分解アクティビティの1ターン分の処理を実行する。
// 対象が消えている可能性があるため、毎ターン先頭で生存を確認する
func (da *DisassembleActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if !world.ECS.Alive(*comp.Target) {
		Cancel(comp, "分解対象が消えたため中断")
		return nil
	}
	if !isAreaSafe(actor, world) {
		Cancel(comp, "周囲に敵がいるため分解を中断")
		return nil
	}
	def, ok := findDisassemblyDef(*comp.Target, world)
	if !ok {
		Cancel(comp, "分解定義が見つからないため中断")
		return nil
	}
	if _, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory); !ok {
		Cancel(comp, "工具を失ったため分解を中断")
		return nil
	}

	comp.TurnsLeft--
	if comp.TurnsLeft <= 0 {
		Complete(comp)
	}
	return nil
}

// Finish は分解完了時の処理を実行する。産出を抽選し、propは足元へ落として
// エンティティを除去、アイテムは1個消費して所持品へ加える
func (da *DisassembleActivity) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	target := *comp.Target
	if !world.ECS.Alive(target) {
		return nil
	}
	def, ok := findDisassemblyDef(target, world)
	if !ok {
		return fmt.Errorf("分解定義が見つかりません")
	}
	grade, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory)
	if !ok {
		return fmt.Errorf("分解に必要な工具を持っていません")
	}

	name := query.GetEntityName(target, world)
	stacks, err := lifecycle.RollDisassemblyYields(world.Config.RNG, def, mechanicSkillValue(actor, world), grade, true)
	if err != nil {
		return fmt.Errorf("分解産出の抽選に失敗: %w", err)
	}

	if world.Components.Prop.Has(target) && world.Components.GridElement.Has(target) {
		// 座標は除去前に値で控える
		coord := world.Components.GridElement.Get(target).Coord
		// 収納propの場合は中身を足元へ出してから取り壊す
		lifecycle.SpillStorageItems(world, target, coord.X, coord.Y)
		world.ECS.RemoveEntity(target)
		query.InvalidateSpatialIndex(world)
		if err := lifecycle.SpawnDisassemblyYields(world, stacks, coord.X, coord.Y); err != nil {
			return fmt.Errorf("分解産出の生成に失敗: %w", err)
		}
	} else {
		if err := lifecycle.ChangeItemCount(world, target, -1); err != nil {
			return fmt.Errorf("分解対象の消費に失敗: %w", err)
		}
		for _, s := range stacks {
			if _, err := lifecycle.SpawnBackpackItem(world, s.Name, s.Count); err != nil {
				return fmt.Errorf("分解産出の生成に失敗: %w", err)
			}
		}
	}

	logger := gamelog.New(query.GetGameLog(world)).
		ItemName(name).
		Append("を分解した。")
	appendYields(logger, stacks)
	logger.Log()

	da.gainMechanicExp(actor, world)

	log.Debug("分解完了", "actor", actor, "target", name, "yields", stacks)
	return nil
}

// Canceled は分解キャンセル時の処理を実行する。
// 対象消滅による中断もあるため、名前は対象が生きている場合だけ出す
func (da *DisassembleActivity) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if world.Components.Player.Has(actor) {
		logger := gamelog.New(query.GetGameLog(world))
		if comp.Target != nil && world.ECS.Alive(*comp.Target) {
			logger.ItemName(query.GetEntityName(*comp.Target, world)).Append("の分解を中断した")
		} else {
			logger.Append("分解を中断した")
		}
		logger.Log()
	}

	log.Debug("分解中断", "reason", comp.CancelReason)
	return nil
}

// gainMechanicExp は分解完了で機械スキルの経験値を与える
func (da *DisassembleActivity) gainMechanicExp(actor ecs.Entity, world w.World) {
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
			Append(fmt.Sprintf("%sスキルが %d に上がった", gc.SkillName(gc.SkillMechanic), s.Value)).
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
		itemName := query.GetEntityName(itemEntity, world)
		tool, ok := raw.FindDisassemblyTool(world.Resources.RawMaster, itemName)
		if !ok || !slices.Contains(tool.Categories, category) {
			continue
		}
		if int(tool.Grade) > bestGrade {
			bestGrade = int(tool.Grade)
			bestName = itemName
		}
	}

	return bestGrade, bestName, bestGrade > 0
}

// findDisassemblyDef は対象エンティティの名前で分解定義を引く
func findDisassemblyDef(entity ecs.Entity, world w.World) (*oapi.Disassembly, bool) {
	name := query.GetEntityName(entity, world)
	return raw.FindDisassembly(world.Resources.RawMaster, name)
}

// mechanicSkillValue はactorの機械スキル値を返す。スキルを持たなければ0
func mechanicSkillValue(actor ecs.Entity, world w.World) int {
	if !world.Components.Skills.Has(actor) {
		return 0
	}
	return world.Components.Skills.Get(actor).Get(gc.SkillMechanic).Value
}

// appendYields は産出一覧をアイテム名の色付きでログへ追記する
func appendYields(logger *gamelog.Logger, stacks []lifecycle.YieldStack) {
	if len(stacks) == 0 {
		logger.Append("何も得られなかった")
		return
	}
	for i, s := range stacks {
		if i > 0 {
			logger.Append("、")
		}
		logger.ItemName(s.Name).Append(fmt.Sprintf(" x%d", s.Count))
	}
	logger.Append(" を得た")
}
