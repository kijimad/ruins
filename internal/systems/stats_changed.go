package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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

	var targets []ecs.Entity
	statsQuery := query.ActiveFilter2[gc.StatsChanged, gc.Abilities](world).Query()
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
		equipQuery := query.ActiveFilter2[gc.LocationEquipped, gc.Wearable](world).Query()
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

		// 装備した光源をプレイヤー自身の LightSource へ写す。装備品は位置を持たず vision に
		// 拾われないので、位置を持つプレイヤーが代わりに光源になる。複数あれば最も明るいもの。
		// プレイヤー限定なのは、発光する member 等の内蔵光源を装備の有無で消さないため。
		if world.Components.Player.Has(entity) && world.Components.LightSource.Has(entity) {
			ls := world.Components.LightSource.Get(entity)
			ls.Enabled = false
			var bestRadius consts.Tile
			lightQuery := query.ActiveFilter2[gc.LocationEquipped, gc.LightSource](world).Query()
			for lightQuery.Next() {
				litem := lightQuery.Entity()
				if world.Components.LocationEquipped.Get(litem).Owner != entity {
					continue
				}
				src := world.Components.LightSource.Get(litem)
				if src.Enabled && src.Radius > bestRadius {
					bestRadius = src.Radius
					ls.Radius = src.Radius
					ls.Color = src.Color
					ls.Enabled = true
				}
			}
			// 光源が変わったので視界を再計算させる
			query.GetVisionState(world).RequestUpdate()
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

		// HP/Poolsを更新。abils を使うので、CharModifiers の Upsert による構造変更で
		// abils ポインタが無効化される前に行う
		if world.Components.HP.Has(entity) {
			hp := world.Components.HP.Get(entity)
			hp.Max = maxHP(abils)
			hp.Current = min(hp.Max, hp.Current)
		}

		// スキル効果倍率を再計算する。能力値変更後に行う。CharModifiers の Add は構造変更で
		// abils ポインタを無効化するので、abils を使う処理はこれより前に済ませる
		if world.Components.Skills.Has(entity) {
			skills := world.Components.Skills.Get(entity)
			var hs *gc.HealthStatus
			if world.Components.HealthStatus.Has(entity) {
				hs = world.Components.HealthStatus.Get(entity)
			}
			effects := gc.RecalculateCharModifiers(skills, abils, hs)
			if err := gc.Upsert(world.ECS, world.Components.CharModifiers, entity, effects); err != nil {
				return err
			}
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
