package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ApplyProfession はプレイヤーエンティティに職業の属性値・スキル・装備を適用する。
// 職業選択時とラン終了時の再適用で使う。
func ApplyProfession(world w.World, player ecs.Entity, prof oapi.Profession) error {
	// 職業IDを保持する
	player.AddComponent(world.Components.Profession, &gc.Profession{ID: prof.Id})

	// 職業の属性値で上書き
	abils := world.Components.Abilities.Get(player).(*gc.Abilities)
	abils.Strength = gc.Ability{Base: int(prof.Abilities.Strength)}
	abils.Sensation = gc.Ability{Base: int(prof.Abilities.Sensation)}
	abils.Dexterity = gc.Ability{Base: int(prof.Abilities.Dexterity)}
	abils.Agility = gc.Ability{Base: int(prof.Abilities.Agility)}
	abils.Vitality = gc.Ability{Base: int(prof.Abilities.Vitality)}
	abils.Defense = gc.Ability{Base: int(prof.Abilities.Defense)}

	// 職業のスキル初期値を設定
	skills := world.Components.Skills.Get(player).(*gc.Skills)
	*skills = *gc.NewSkills()
	for _, ps := range prof.Skills {
		skills.Get(gc.SkillID(ps.Id)).Value = int(ps.Value)
	}
	modifiers := gc.RecalculateCharModifiers(skills, abils, nil)
	player.AddComponent(world.Components.CharModifiers, modifiers)

	// 属性値変更後にHP/SP/EP/APを再計算
	_ = FullRecover(world, player)

	// 初期アイテムをバックパックに付与
	for _, profItem := range prof.Items {
		if _, err := SpawnItem(world, profItem.Name, int(profItem.Count), gc.ItemLocationInPlayerBackpack); err != nil {
			return fmt.Errorf("職業の初期アイテム生成に失敗: %s: %w", profItem.Name, err)
		}
	}

	// 初期装備を付与して装備する
	for _, equip := range prof.Equips {
		item, err := SpawnItem(world, equip.Name, 1, gc.ItemLocationInPlayerBackpack)
		if err != nil {
			return fmt.Errorf("職業の初期装備生成に失敗: %s: %w", equip.Name, err)
		}
		slot, ok := gc.ParseEquipmentSlot(string(equip.Slot))
		if !ok {
			return fmt.Errorf("不正な装備スロット名: %s (アイテム: %s)", equip.Slot, equip.Name)
		}
		MoveToEquip(world, item, player, slot)
	}

	return nil
}
