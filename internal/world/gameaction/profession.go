package gameaction

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
)

// ApplyProfession はプレイヤーエンティティに職業の属性値・スキル・装備を適用する。
// 職業選択時に使う。
func ApplyProfession(world w.World, player ecs.Entity, prof oapi.Profession) error {
	// 職業IDを保持する。再適用時は既存を更新する
	if err := gc.Upsert(world.ECS, world.Components.Profession, player, &gc.Profession{ID: prof.Id}); err != nil {
		return err
	}

	// 職業の属性値で上書き
	abils := world.Components.Abilities.Get(player)
	abils.Strength = gc.Ability{Base: int(prof.Abilities.Strength)}
	abils.Sensation = gc.Ability{Base: int(prof.Abilities.Sensation)}
	abils.Dexterity = gc.Ability{Base: int(prof.Abilities.Dexterity)}
	abils.Agility = gc.Ability{Base: int(prof.Abilities.Agility)}
	abils.Vitality = gc.Ability{Base: int(prof.Abilities.Vitality)}
	abils.Defense = gc.Ability{Base: int(prof.Abilities.Defense)}

	// 職業のスキル初期値を設定
	playerSkills := world.Components.Skills.Get(player)
	*playerSkills = *gc.NewSkills()
	if prof.Skills != nil {
		for _, ps := range *prof.Skills {
			playerSkills.Get(gc.SkillID(ps.Id)).Value = int(ps.Value)
		}
	}
	modifiers := gc.RecalculateCharModifiers(playerSkills, abils, nil)
	if err := gc.Upsert(world.ECS, world.Components.CharModifiers, player, modifiers); err != nil {
		return err
	}

	// 属性値変更後にHP/APを再計算
	_ = lifecycle.FullRecover(world, player)

	// 初期アイテムをバックパックに付与
	for _, profItem := range prof.Items {
		if _, err := lifecycle.SpawnBackpackItem(world, profItem.Name, int(profItem.Count)); err != nil {
			return fmt.Errorf("職業の初期アイテム生成に失敗: %s: %w", profItem.Name, err)
		}
	}

	// 初期装備を付与して装備する
	for _, equip := range prof.Equips {
		item, err := lifecycle.SpawnBackpackItem(world, equip.Name, 1)
		if err != nil {
			return fmt.Errorf("職業の初期装備生成に失敗: %s: %w", equip.Name, err)
		}
		slot, ok := gc.ParseEquipmentSlot(string(equip.Slot))
		if !ok {
			return fmt.Errorf("不正な装備スロット名: %s (アイテム: %s)", equip.Slot, equip.Name)
		}
		lifecycle.MoveToEquip(world, item, player, slot)
	}

	return nil
}
