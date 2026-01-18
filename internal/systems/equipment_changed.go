package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// EquipmentChangedSystem は装備変更のダーティフラグが立ったら、ステータス補正まわりを再計算する
// TODO: 最大HP/SPの更新はここでやったほうがよさそう
// TODO: マイナスにならないようにする
type EquipmentChangedSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys EquipmentChangedSystem) String() string {
	return "EquipmentChangedSystem"
}

// Update は装備変更フラグをチェックし、必要に応じてステータスを再計算する
// w.Updater interfaceを実装
func (sys *EquipmentChangedSystem) Update(world w.World) error {
	var updateErr error

	// EquipmentChangedが付与されたエンティティを処理
	world.Manager.Join(
		world.Components.EquipmentChanged,
		world.Components.Attributes,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entity.RemoveComponent(world.Components.EquipmentChanged)
		attrs := world.Components.Attributes.Get(entity).(*gc.Attributes)

		// Attributes初期化
		{
			attrs.Vitality.Modifier = 0
			attrs.Vitality.Total = attrs.Vitality.Base
			attrs.Strength.Modifier = 0
			attrs.Strength.Total = attrs.Strength.Base
			attrs.Sensation.Modifier = 0
			attrs.Sensation.Total = attrs.Sensation.Base
			attrs.Dexterity.Modifier = 0
			attrs.Dexterity.Total = attrs.Dexterity.Base
			attrs.Agility.Modifier = 0
			attrs.Agility.Total = attrs.Agility.Base
			attrs.Defense.Modifier = 0
			attrs.Defense.Total = attrs.Defense.Base
		}

		// 装備効果を加算
		{
			world.Manager.Join(
				world.Components.ItemLocationEquipped,
				world.Components.Wearable,
			).Visit(ecs.Visit(func(item ecs.Entity) {
				equipped := world.Components.ItemLocationEquipped.Get(item).(*gc.LocationEquipped)

				// このエンティティの装備のみ処理
				if equipped.Owner != entity {
					return
				}

				wearable := world.Components.Wearable.Get(item).(*gc.Wearable)

				attrs.Defense.Modifier += wearable.Defense
				attrs.Defense.Total = attrs.Defense.Base + attrs.Defense.Modifier

				attrs.Vitality.Modifier += wearable.EquipBonus.Vitality
				attrs.Vitality.Total = attrs.Vitality.Base + attrs.Vitality.Modifier
				attrs.Strength.Modifier += wearable.EquipBonus.Strength
				attrs.Strength.Total = attrs.Strength.Base + attrs.Strength.Modifier
				attrs.Sensation.Modifier += wearable.EquipBonus.Sensation
				attrs.Sensation.Total = attrs.Sensation.Base + attrs.Sensation.Modifier
				attrs.Dexterity.Modifier += wearable.EquipBonus.Dexterity
				attrs.Dexterity.Total = attrs.Dexterity.Base + attrs.Dexterity.Modifier
				attrs.Agility.Modifier += wearable.EquipBonus.Agility
				attrs.Agility.Total = attrs.Agility.Base + attrs.Agility.Modifier
			}))
		}

		// Pools（HP/SP）を更新
		if entity.HasComponent(world.Components.Pools) {
			pools := world.Components.Pools.Get(entity).(*gc.Pools)

			pools.HP.Max = maxHP(attrs)
			pools.HP.Current = min(pools.HP.Max, pools.HP.Current)
			pools.SP.Max = maxSP(attrs)
			pools.SP.Current = min(pools.SP.Max, pools.SP.Current)

			// 所持重量を再計算する。力が変化した場合に最大重量が変わるので
			entity.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
		}

		// APを再計算
		if entity.HasComponent(world.Components.TurnBased) {
			maxAP, err := worldhelper.CalculateMaxActionPoints(world, entity)
			if err != nil {
				updateErr = err
				return
			}
			turnBased := world.Components.TurnBased.Get(entity).(*gc.TurnBased)

			// 最大APを更新
			turnBased.AP.Max = maxAP

			// 現在APが最大APを超えている場合は切り詰める
			if turnBased.AP.Current > maxAP {
				turnBased.AP.Current = maxAP
			}
		}
	}))

	return updateErr
}

// 30+(体力*8+力+感覚)
func maxHP(attrs *gc.Attributes) int {
	return 30 + attrs.Vitality.Total*8 + attrs.Strength.Total + attrs.Sensation.Total
}

// 体力*2+器用さ+素早さ
func maxSP(attrs *gc.Attributes) int {
	return attrs.Vitality.Total*2 + attrs.Dexterity.Total + attrs.Agility.Total
}
