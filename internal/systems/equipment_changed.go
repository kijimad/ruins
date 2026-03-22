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
		world.Components.Abilities,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entity.RemoveComponent(world.Components.EquipmentChanged)
		abils := world.Components.Abilities.Get(entity).(*gc.Abilities)

		// Abilities初期化
		{
			abils.Vitality.Modifier = 0
			abils.Vitality.Total = abils.Vitality.Base
			abils.Strength.Modifier = 0
			abils.Strength.Total = abils.Strength.Base
			abils.Sensation.Modifier = 0
			abils.Sensation.Total = abils.Sensation.Base
			abils.Dexterity.Modifier = 0
			abils.Dexterity.Total = abils.Dexterity.Base
			abils.Agility.Modifier = 0
			abils.Agility.Total = abils.Agility.Base
			abils.Defense.Modifier = 0
			abils.Defense.Total = abils.Defense.Base
		}

		// 装備効果を加算
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

			abils.Defense.Modifier += wearable.Defense
			abils.Vitality.Modifier += wearable.EquipBonus.Vitality
			abils.Strength.Modifier += wearable.EquipBonus.Strength
			abils.Sensation.Modifier += wearable.EquipBonus.Sensation
			abils.Dexterity.Modifier += wearable.EquipBonus.Dexterity
			abils.Agility.Modifier += wearable.EquipBonus.Agility
		}))

		// 健康ペナルティを加算
		if entity.HasComponent(world.Components.HealthStatus) {
			hs := world.Components.HealthStatus.Get(entity).(*gc.HealthStatus)
			abils.Vitality.Modifier += hs.GetStatModifier(gc.StatVitality)
			abils.Strength.Modifier += hs.GetStatModifier(gc.StatStrength)
			abils.Sensation.Modifier += hs.GetStatModifier(gc.StatSensation)
			abils.Dexterity.Modifier += hs.GetStatModifier(gc.StatDexterity)
			abils.Agility.Modifier += hs.GetStatModifier(gc.StatAgility)
			abils.Defense.Modifier += hs.GetStatModifier(gc.StatDefense)
		}

		// Total を計算
		abils.Vitality.Total = abils.Vitality.Base + abils.Vitality.Modifier
		abils.Strength.Total = abils.Strength.Base + abils.Strength.Modifier
		abils.Sensation.Total = abils.Sensation.Base + abils.Sensation.Modifier
		abils.Dexterity.Total = abils.Dexterity.Base + abils.Dexterity.Modifier
		abils.Agility.Total = abils.Agility.Base + abils.Agility.Modifier
		abils.Defense.Total = abils.Defense.Base + abils.Defense.Modifier

		// スキル効果倍率を再計算する。能力値変更後に行う
		if entity.HasComponent(world.Components.Skills) {
			skills := world.Components.Skills.Get(entity).(*gc.Skills)
			var hs *gc.HealthStatus
			if entity.HasComponent(world.Components.HealthStatus) {
				hs = world.Components.HealthStatus.Get(entity).(*gc.HealthStatus)
			}
			effects := gc.RecalculateCharModifiers(skills, abils, hs)
			entity.AddComponent(world.Components.CharModifiers, effects)
		}

		// Pools（HP/SP）を更新
		if entity.HasComponent(world.Components.Pools) {
			pools := world.Components.Pools.Get(entity).(*gc.Pools)

			pools.HP.Max = maxHP(abils)
			pools.HP.Current = min(pools.HP.Max, pools.HP.Current)
			pools.SP.Max = maxSP(abils)
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
func maxHP(abils *gc.Abilities) int {
	return 30 + abils.Vitality.Total*8 + abils.Strength.Total + abils.Sensation.Total
}

// 体力*2+器用さ+素早さ
func maxSP(abils *gc.Abilities) int {
	return abils.Vitality.Total*2 + abils.Dexterity.Total + abils.Agility.Total
}
