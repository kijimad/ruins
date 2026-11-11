package activity

import (
	"fmt"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/skill"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// wieldedTool は actor が構えている武器が分解定義 def に合う道具ならそのグレードを返す
func wieldedTool(world w.World, actor ecs.Entity, def *rawschema.Disassembly) (int, bool) {
	if def.ToolCategory == nil || !world.Components.Player.Has(actor) {
		return 0, false
	}
	weapon, ok, err := selectedWeapon(world, actor)
	if err != nil || !ok {
		return 0, false
	}
	tool, ok := raw.FindTool(world.Resources.RawMaster, query.GetEntityID(weapon, world))
	if !ok {
		return 0, false
	}
	if slices.Contains(tool.Categories, *def.ToolCategory) {
		return tool.Grade, true
	}
	return 0, false
}

// toolDamageMultiplier は prop を殴るときの道具のダメージ倍率を返す。合う道具で 1+グレード、道具が必須で無ければ0、他は1
func toolDamageMultiplier(world w.World, actor, target ecs.Entity) int {
	if !world.Components.Prop.Has(target) {
		return 1
	}
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(target, world))
	if !ok {
		return 1
	}
	grade, ok := wieldedTool(world, actor, def)
	switch {
	case ok:
		return 1 + grade
	case def.ToolRequired != nil && *def.ToolRequired:
		return 0
	}
	return 1
}

// salvageRoll は壊したときの抽選を決める。合う道具を構えているか、道具の分類が無い定義なら全枠を出す
func salvageRoll(world w.World, actor ecs.Entity, def *rawschema.Disassembly) (grade int, full, withTool bool) {
	grade, withTool = wieldedTool(world, actor, def)
	return grade, withTool || def.ToolCategory == nil, withTool
}

// dropSalvage は actor が壊した prop の産出を足元へ落とす
func dropSalvage(world w.World, actor, target ecs.Entity) error {
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(target, world))
	if !ok || !world.Components.GridElement.Has(target) {
		return nil
	}
	grade, full, withTool := salvageRoll(world, actor, def)
	var skillID rawschema.SkillId
	hasSkill := false
	if withTool {
		skillID, hasSkill = world.Resources.SkillDefs.ToolSkills[*def.ToolCategory]
	}
	skillValue := 0
	if hasSkill {
		skillValue = skillValueOf(actor, world, skillID)
	}
	stacks, err := lifecycle.RollDisassemblyYields(world.Resources.Config.RNG, def, skillValue, grade, full)
	if err != nil {
		return fmt.Errorf("failed to roll salvage: %w", err)
	}
	coord := world.Components.GridElement.Get(target).Coord
	if err := lifecycle.SpawnDisassemblyYields(world, stacks, coord.X, coord.Y); err != nil {
		return fmt.Errorf("failed to spawn salvage: %w", err)
	}
	if hasSkill {
		gainSkillExp(actor, world, skillID)
	}
	return nil
}

// gainSkillExp は道具で prop を壊したときに、その道具の分類のスキルの経験値を与える
func gainSkillExp(actor ecs.Entity, world w.World, skillID rawschema.SkillId) {
	if !world.Components.Skills.Has(actor) {
		return
	}
	s := world.Components.Skills.Get(actor).Get(skillID)

	abilityValue := 0
	if world.Components.Abilities.Has(actor) {
		abilityValue = world.Components.Abilities.Get(actor).ValueOf(world.Resources.SkillDefs.Ability(skillID))
	}

	if skill.GainExp(s, abilityValue) {
		// 同一ターン内の別処理が既にマーカーを付けていることがあるため、二重付与を避ける
		if !world.Components.StatsChanged.Has(actor) {
			world.Components.StatsChanged.Add(actor, &gc.StatsChanged{})
		}
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s skill rose to %d", world.Resources.SkillDefs.Name(skillID), s.Value)).
			Log()
	}
}

// skillValueOf は actor のスキル skillID の値を返す。スキルを持たなければ0
func skillValueOf(actor ecs.Entity, world w.World, skillID rawschema.SkillId) int {
	if !world.Components.Skills.Has(actor) {
		return 0
	}
	return world.Components.Skills.Get(actor).Level(skillID)
}
