package activity

import (
	"fmt"
	"slices"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/skill"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// disassemblyTool は actor が構えている武器が分解定義 def の道具ならそのグレードを返す
func disassemblyTool(world w.World, actor ecs.Entity, def *oapi.Disassembly) (int, bool) {
	if def.ToolCategory == nil || !world.Components.Player.Has(actor) {
		return 0, false
	}
	weapon, ok, err := selectedWeapon(world, actor)
	if err != nil || !ok {
		return 0, false
	}
	tool, ok := raw.FindDisassemblyTool(world.Resources.RawMaster, query.GetEntityID(weapon, world))
	if !ok {
		return 0, false
	}
	if slices.Contains(tool.Categories, *def.ToolCategory) {
		return tool.Grade, true
	}
	return 0, false
}

// toolDamageMultiplier は分解定義を持つ prop を殴るときのダメージ倍率を返す。道具を構えていれば
// グレード1で2倍、以降1ごとに1倍ずつ増え、壊すまでの手数が減る。道具が必須の prop は道具なしでは0倍
func toolDamageMultiplier(world w.World, actor, target ecs.Entity) int {
	if !world.Components.Prop.Has(target) {
		return 1
	}
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(target, world))
	if !ok {
		return 1
	}
	grade, ok := disassemblyTool(world, actor, def)
	switch {
	case ok:
		return 1 + grade
	case def.ToolRequired != nil && *def.ToolRequired:
		return 0
	}
	return 1
}

// salvageRoll は分解定義 def の prop を actor が壊したときの抽選を決める。合う道具を構えていれば
// 確率枠とボーナス枠まで出し、道具の分類が無い定義も同じにする。それ以外は確定枠を低確率で回収するだけ
func salvageRoll(world w.World, actor ecs.Entity, def *oapi.Disassembly) (grade int, full, withTool bool) {
	grade, withTool = disassemblyTool(world, actor, def)
	return grade, withTool || def.ToolCategory == nil, withTool
}

// dropSalvage は actor が壊した prop の産出を足元へ落とす。合う道具で壊したときだけ機械スキルが伸びる
func dropSalvage(world w.World, actor, target ecs.Entity) error {
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(target, world))
	if !ok || !world.Components.GridElement.Has(target) {
		return nil
	}
	grade, full, withTool := salvageRoll(world, actor, def)
	skillValue := 0
	if full {
		skillValue = mechanicSkillValue(actor, world)
	}
	stacks, err := lifecycle.RollDisassemblyYields(world.Resources.Config.RNG, def, skillValue, grade, full)
	if err != nil {
		return fmt.Errorf("failed to roll salvage: %w", err)
	}
	coord := world.Components.GridElement.Get(target).Coord
	if err := lifecycle.SpawnDisassemblyYields(world, stacks, coord.X, coord.Y); err != nil {
		return fmt.Errorf("failed to spawn salvage: %w", err)
	}
	if len(stacks) > 0 && world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "From %s, obtained %s.", gamelog.Tag("item", query.GetEntityName(target, world)), yieldsMarkup(stacks, world))).
			Log()
	}
	if withTool {
		gainMechanicExp(actor, world)
	}
	return nil
}

// gainMechanicExp は道具で prop を壊したときに機械スキルの経験値を与える
func gainMechanicExp(actor ecs.Entity, world w.World) {
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

// mechanicSkillValue はactorの機械スキル値を返す。スキルを持たなければ0
func mechanicSkillValue(actor ecs.Entity, world w.World) int {
	if !world.Components.Skills.Has(actor) {
		return 0
	}
	return world.Components.Skills.Get(actor).Get(gc.SkillMechanic).Value
}

// yieldsMarkup は産出一覧を色付きマークアップ文字列にして返す。各アイテムを読点で連結する。
// 語尾や句読点は呼び出し側のテンプレートに委ね、ここは中身の並びだけを組む
func yieldsMarkup(stacks []lifecycle.YieldStack, world w.World) string {
	parts := make([]string, len(stacks))
	for i, s := range stacks {
		parts[i] = gamelog.Tag("item", s.Name) + fmt.Sprintf(" x%d", s.Count)
	}
	return strings.Join(parts, query.T(world, ", "))
}
