package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// StatsChangedSystem はステータス再計算のダーティフラグが立ったら、ステータス補正まわりを再計算する
// TODO: 最大HP/SPの更新はここでやったほうがよさそう
// TODO: マイナスにならないようにする
type StatsChangedSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys StatsChangedSystem) String() string {
	return "StatsChangedSystem"
}

// Update はステータス再計算フラグをチェックし、必要に応じてステータスを再計算する
// w.Updater interfaceを実装
func (sys *StatsChangedSystem) Update(world w.World) error {
	var updateErr error

	// Remove/Addの構造変更を行うため、対象を集めてから反復後に処理する
	var targets []ecs.Entity
	statsQuery := ecs.NewFilter2[gc.StatsChanged, gc.Abilities](world.World).Query()
	for statsQuery.Next() {
		targets = append(targets, statsQuery.Entity())
	}

	// StatsChangedが付与されたエンティティを処理
	for _, entity := range targets {
		world.Components.StatsChanged.Remove(entity)
		abils := world.Components.Abilities.Get(entity)

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
		equipQuery := ecs.NewFilter2[gc.LocationEquipped, gc.Wearable](world.World).Query()
		for equipQuery.Next() {
			item := equipQuery.Entity()
			equipped := world.Components.LocationEquipped.Get(item)

			// このエンティティの装備のみ処理
			if equipped.Owner != entity {
				continue
			}

			wearable := world.Components.Wearable.Get(item)

			abils.Defense.Modifier += wearable.Defense
			abils.Vitality.Modifier += wearable.EquipBonus.Vitality
			abils.Strength.Modifier += wearable.EquipBonus.Strength
			abils.Sensation.Modifier += wearable.EquipBonus.Sensation
			abils.Dexterity.Modifier += wearable.EquipBonus.Dexterity
			abils.Agility.Modifier += wearable.EquipBonus.Agility
		}

		// 健康ペナルティを加算
		if world.Components.HealthStatus.Has(entity) {
			hs := world.Components.HealthStatus.Get(entity)
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
		if world.Components.Skills.Has(entity) {
			skills := world.Components.Skills.Get(entity)
			var hs *gc.HealthStatus
			if world.Components.HealthStatus.Has(entity) {
				hs = world.Components.HealthStatus.Get(entity)
			}
			effects := gc.RecalculateCharModifiers(skills, abils, hs)
			gc.Upsert(world.Components.CharModifiers, entity, effects)
		}

		// HP/Poolsを更新
		if world.Components.HP.Has(entity) {
			hp := world.Components.HP.Get(entity)
			hp.Max = maxHP(abils)
			hp.Current = min(hp.Max, hp.Current)
		}
		if world.Components.WeightCapacity.Has(entity) {
			// 所持重量を再計算する。力が変化した場合に最大重量が変わるので
			if !world.Components.WeightDirty.Has(entity) {
				world.Components.WeightDirty.Add(entity, &gc.WeightDirty{})
			}
		}

		// APを再計算
		if world.Components.TurnBased.Has(entity) {
			maxAP, err := query.CalculateMaxActionPoints(world, entity)
			if err != nil {
				updateErr = err
				continue
			}
			turnBased := world.Components.TurnBased.Get(entity)

			// 最大APを更新
			turnBased.AP.Max = maxAP

			// 現在APが最大APを超えている場合は切り詰める
			if turnBased.AP.Current > maxAP {
				turnBased.AP.Current = maxAP
			}
		}
	}

	return updateErr
}

// 30+(体力*8+力+感覚)
func maxHP(abils *gc.Abilities) int {
	return 30 + abils.Vitality.Total*8 + abils.Strength.Total + abils.Sensation.Total
}
