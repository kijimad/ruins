package gameaction

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
)

var (
	// errProfessionItem は職業の初期アイテムを生成できないときに返す
	errProfessionItem = errors.New("failed to generate profession starting item")
	// errProfessionEquipment は職業の初期装備を生成できないときに返す
	errProfessionEquipment = errors.New("failed to generate profession starting equipment")
	// errInvalidEquipmentSlot は職業の初期装備のスロット名が不正なときに返す
	errInvalidEquipmentSlot = errors.New("invalid equipment slot name")
)

// ApplyProfession はプレイヤーエンティティに職業の属性値・スキル・装備を適用する。
// 職業選択時に使う。
func ApplyProfession(world w.World, player ecs.Entity, prof rawschema.Profession) error {
	// 職業IDを保持する。再適用時は既存を更新する
	if err := gc.Upsert(world.ECS, world.Components.Profession, player, &gc.Profession{ID: prof.Id}); err != nil {
		return err
	}

	// 職業の属性値で上書き
	abils := world.Components.Abilities.Get(player)
	abils.Strength = gc.Ability{Base: prof.Abilities.Strength}
	abils.Sensation = gc.Ability{Base: prof.Abilities.Sensation}
	abils.Dexterity = gc.Ability{Base: prof.Abilities.Dexterity}
	abils.Agility = gc.Ability{Base: prof.Abilities.Agility}
	abils.Vitality = gc.Ability{Base: prof.Abilities.Vitality}
	abils.Defense = gc.Ability{Base: prof.Abilities.Defense}

	// 職業のスキル初期値を設定
	playerSkills := world.Components.Skills.Get(player)
	*playerSkills = *gc.NewSkills()
	if prof.Skills != nil {
		for _, ps := range *prof.Skills {
			playerSkills.Get(ps.Id).Value = ps.Value
		}
	}
	// 属性値変更後にHP/APを再計算
	_ = lifecycle.FullRecover(world, player)

	// 初期アイテムをバックパックに付与
	for _, profItem := range prof.Items {
		if _, err := lifecycle.SpawnBackpackItem(world, profItem.Name, profItem.Count); err != nil {
			return fmt.Errorf("%w: %s: %w", errProfessionItem, profItem.Name, err)
		}
	}

	// 初期装備を付与して装備する
	for _, equip := range prof.Equips {
		item, err := lifecycle.SpawnBackpackItem(world, equip.Name, 1)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", errProfessionEquipment, equip.Name, err)
		}
		slot, ok := gc.ParseEquipmentSlot(equip.Slot)
		if !ok {
			return fmt.Errorf("%w: %s (item: %s)", errInvalidEquipmentSlot, equip.Slot, equip.Name)
		}
		lifecycle.MoveToEquip(world, item, player, slot)
	}

	return nil
}
