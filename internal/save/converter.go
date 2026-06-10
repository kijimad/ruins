package save

// gc型とoapi生成型(SaveData*)の相互変換を提供する。
// セーブ時はgc型→SaveData型、ロード時はSaveData型→gc型に変換する。

import (
	"fmt"
	"image/color"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// ================== StableID変換 ==================

func stableIDToSaveData(id StableID) oapi.SaveDataStableID {
	return oapi.SaveDataStableID{
		Index:      id.Index,
		Generation: id.Generation,
	}
}

func stableIDFromSaveData(sd oapi.SaveDataStableID) StableID {
	return StableID{
		Index:      sd.Index,
		Generation: sd.Generation,
	}
}

// ================== GameProgress変換 ==================

func gameProgressToSaveData(gp *gc.GameProgress) *oapi.SaveDataGameProgress {
	if gp == nil {
		return nil
	}
	events := make(map[string]oapi.SaveDataEventState, len(gp.Events))
	for k, v := range gp.Events {
		events[k] = oapi.SaveDataEventState{Active: v.Active, Seen: v.Seen}
	}
	return &oapi.SaveDataGameProgress{
		ClearedDungeons: gp.ClearedDungeons,
		Events:          events,
	}
}

func gameProgressFromSaveData(sd *oapi.SaveDataGameProgress) *gc.GameProgress {
	if sd == nil {
		return nil
	}
	events := make(map[string]gc.EventState, len(sd.Events))
	for k, v := range sd.Events {
		events[k] = gc.EventState{Active: v.Active, Seen: v.Seen}
	}
	return &gc.GameProgress{
		ClearedDungeons: sd.ClearedDungeons,
		Events:          events,
	}
}

// ================== データコンポーネント変換 ==================

func hpToSaveData(hp gc.HP) oapi.SaveDataHPComponent {
	return oapi.SaveDataHPComponent{Max: int32(hp.Max), Current: int32(hp.Current)}
}

func hpFromSaveData(sd oapi.SaveDataHPComponent) gc.HP {
	return gc.HP{Pool: gc.Pool{Max: int(sd.Max), Current: int(sd.Current)}}
}

func carryWeightToSaveData(cw gc.CarryWeight) oapi.SaveDataCarryWeightComponent {
	return oapi.SaveDataCarryWeightComponent{Max: cw.Max, Current: cw.Current}
}

func carryWeightFromSaveData(sd oapi.SaveDataCarryWeightComponent) gc.CarryWeight {
	return gc.CarryWeight{PoolFloat: gc.PoolFloat{Max: sd.Max, Current: sd.Current}}
}

func turnBasedToSaveData(tb gc.TurnBased) oapi.SaveDataTurnBasedComponent {
	return oapi.SaveDataTurnBasedComponent{
		AP:    oapi.SaveDataIntPool{Max: int32(tb.AP.Max), Current: int32(tb.AP.Current)},
		Speed: int32(tb.Speed),
	}
}

func turnBasedFromSaveData(sd oapi.SaveDataTurnBasedComponent) gc.TurnBased {
	return gc.TurnBased{
		AP:    gc.Pool{Max: int(sd.AP.Max), Current: int(sd.AP.Current)},
		Speed: int(sd.Speed),
	}
}

func abilitiesToSaveData(a gc.Abilities) oapi.SaveDataAbilitiesComponent {
	conv := func(ab gc.Ability) oapi.SaveDataAbilityValue {
		return oapi.SaveDataAbilityValue{Base: int32(ab.Base), Modifier: int32(ab.Modifier), Total: int32(ab.Total)}
	}
	return oapi.SaveDataAbilitiesComponent{
		Vitality: conv(a.Vitality), Strength: conv(a.Strength), Sensation: conv(a.Sensation),
		Dexterity: conv(a.Dexterity), Agility: conv(a.Agility), Defense: conv(a.Defense),
	}
}

func abilitiesFromSaveData(sd oapi.SaveDataAbilitiesComponent) gc.Abilities {
	conv := func(v oapi.SaveDataAbilityValue) gc.Ability {
		return gc.Ability{Base: int(v.Base), Modifier: int(v.Modifier), Total: int(v.Total)}
	}
	return gc.Abilities{
		Vitality: conv(sd.Vitality), Strength: conv(sd.Strength), Sensation: conv(sd.Sensation),
		Dexterity: conv(sd.Dexterity), Agility: conv(sd.Agility), Defense: conv(sd.Defense),
	}
}

// ================== 表示コンポーネント変換 ==================

func cameraToSaveData(c gc.Camera) oapi.SaveDataCameraComponent {
	return oapi.SaveDataCameraComponent{
		Scale: c.Scale, ScaleTo: c.ScaleTo,
		X: c.X, Y: c.Y, TargetX: c.TargetX, TargetY: c.TargetY,
	}
}

func cameraFromSaveData(sd oapi.SaveDataCameraComponent) gc.Camera {
	return gc.Camera{
		Scale: sd.Scale, ScaleTo: sd.ScaleTo,
		X: sd.X, Y: sd.Y, TargetX: sd.TargetX, TargetY: sd.TargetY,
	}
}

func gridElementToSaveData(g gc.GridElement) oapi.SaveDataGridElementComponent {
	return oapi.SaveDataGridElementComponent{X: int32(g.X), Y: int32(g.Y)}
}

func gridElementFromSaveData(sd oapi.SaveDataGridElementComponent) gc.GridElement {
	return gc.GridElement{X: consts.Tile(sd.X), Y: consts.Tile(sd.Y)}
}

func spriteRenderToSaveData(sr gc.SpriteRender) oapi.SaveDataSpriteRenderComponent {
	result := oapi.SaveDataSpriteRenderComponent{
		SpriteSheetName: sr.SpriteSheetName,
		SpriteKey:       sr.SpriteKey,
		Depth:           int32(sr.Depth),
	}
	if sr.AnimKeys != nil {
		keys := make([]string, len(sr.AnimKeys))
		copy(keys, sr.AnimKeys)
		result.AnimKeys = &keys
	}
	return result
}

func spriteRenderFromSaveData(sd oapi.SaveDataSpriteRenderComponent) gc.SpriteRender {
	sr := gc.SpriteRender{
		SpriteSheetName: sd.SpriteSheetName,
		SpriteKey:       sd.SpriteKey,
		Depth:           gc.DepthNum(sd.Depth),
	}
	if sd.AnimKeys != nil {
		sr.AnimKeys = make([]string, len(*sd.AnimKeys))
		copy(sr.AnimKeys, *sd.AnimKeys)
	}
	return sr
}

func lightSourceToSaveData(ls gc.LightSource) oapi.SaveDataLightSourceComponent {
	return oapi.SaveDataLightSourceComponent{
		Radius:  int32(ls.Radius),
		Enabled: ls.Enabled,
		Color: oapi.SaveDataRGBAColor{
			R: ls.Color.R, G: ls.Color.G, B: ls.Color.B, A: ls.Color.A,
		},
	}
}

func lightSourceFromSaveData(sd oapi.SaveDataLightSourceComponent) gc.LightSource {
	return gc.LightSource{
		Radius:  consts.Tile(sd.Radius),
		Enabled: sd.Enabled,
		Color:   color.RGBA{R: sd.Color.R, G: sd.Color.G, B: sd.Color.B, A: sd.Color.A},
	}
}

// ================== アイテム属性コンポーネント変換 ==================

func targetTypeToSaveData(tt gc.TargetType) oapi.SaveDataTargetTypeData {
	return oapi.SaveDataTargetTypeData{
		TargetGroup: oapi.TargetGroup(tt.TargetGroup),
		TargetNum:   oapi.TargetNum(tt.TargetNum),
	}
}

func targetTypeFromSaveData(sd oapi.SaveDataTargetTypeData) gc.TargetType {
	return gc.TargetType{
		TargetGroup: gc.TargetGroupType(sd.TargetGroup),
		TargetNum:   gc.TargetNumType(sd.TargetNum),
	}
}

func attackCategoryToSaveData(at gc.AttackType) oapi.SaveDataAttackCategoryData {
	return oapi.SaveDataAttackCategoryData{
		Type:  at.Type,
		Range: oapi.SaveDataAttackRangeType(at.Range),
		Label: at.Label,
	}
}

func attackCategoryFromSaveData(sd oapi.SaveDataAttackCategoryData) gc.AttackType {
	return gc.AttackType{
		Type:  sd.Type,
		Range: gc.AttackRangeType(sd.Range),
		Label: sd.Label,
	}
}

func wearableToSaveData(w gc.Wearable) oapi.SaveDataWearableComponent {
	return oapi.SaveDataWearableComponent{
		Defense:           int32(w.Defense),
		EquipmentCategory: oapi.EquipmentCategory(w.EquipmentCategory),
		EquipBonus: oapi.SaveDataEquipBonusData{
			Vitality: int32(w.EquipBonus.Vitality), Strength: int32(w.EquipBonus.Strength),
			Sensation: int32(w.EquipBonus.Sensation), Dexterity: int32(w.EquipBonus.Dexterity),
			Agility: int32(w.EquipBonus.Agility),
		},
		InsulationCold: int32(w.InsulationCold),
		InsulationHeat: int32(w.InsulationHeat),
	}
}

func wearableFromSaveData(sd oapi.SaveDataWearableComponent) gc.Wearable {
	return gc.Wearable{
		Defense:           int(sd.Defense),
		EquipmentCategory: gc.EquipmentType(sd.EquipmentCategory),
		EquipBonus: gc.EquipBonus{
			Vitality: int(sd.EquipBonus.Vitality), Strength: int(sd.EquipBonus.Strength),
			Sensation: int(sd.EquipBonus.Sensation), Dexterity: int(sd.EquipBonus.Dexterity),
			Agility: int(sd.EquipBonus.Agility),
		},
		InsulationCold: int(sd.InsulationCold),
		InsulationHeat: int(sd.InsulationHeat),
	}
}

func meleeToSaveData(m gc.Melee) oapi.SaveDataMeleeComponent {
	return oapi.SaveDataMeleeComponent{
		Accuracy:       int32(m.Accuracy),
		Damage:         int32(m.Damage),
		AttackCount:    int32(m.AttackCount),
		Element:        oapi.Element(m.Element),
		AttackCategory: attackCategoryToSaveData(m.AttackCategory),
		Cost:           int32(m.Cost),
		TargetType:     targetTypeToSaveData(m.TargetType),
	}
}

func meleeFromSaveData(sd oapi.SaveDataMeleeComponent) gc.Melee {
	return gc.Melee{
		Accuracy:       int(sd.Accuracy),
		Damage:         int(sd.Damage),
		AttackCount:    int(sd.AttackCount),
		Element:        gc.ElementType(sd.Element),
		AttackCategory: attackCategoryFromSaveData(sd.AttackCategory),
		Cost:           int(sd.Cost),
		TargetType:     targetTypeFromSaveData(sd.TargetType),
	}
}

func fireToSaveData(f gc.Fire) oapi.SaveDataFireComponent {
	return oapi.SaveDataFireComponent{
		Accuracy:            int32(f.Accuracy),
		Damage:              int32(f.Damage),
		AttackCount:         int32(f.AttackCount),
		Element:             oapi.Element(f.Element),
		AttackCategory:      attackCategoryToSaveData(f.AttackCategory),
		Cost:                int32(f.Cost),
		TargetType:          targetTypeToSaveData(f.TargetType),
		Magazine:            int32(f.Magazine),
		MagazineSize:        int32(f.MagazineSize),
		ReloadEffort:        int32(f.ReloadEffort),
		AmmoTag:             f.AmmoTag,
		LoadedDamageBonus:   int32(f.LoadedDamageBonus),
		LoadedAccuracyBonus: int32(f.LoadedAccuracyBonus),
	}
}

func fireFromSaveData(sd oapi.SaveDataFireComponent) gc.Fire {
	return gc.Fire{
		Accuracy:            int(sd.Accuracy),
		Damage:              int(sd.Damage),
		AttackCount:         int(sd.AttackCount),
		Element:             gc.ElementType(sd.Element),
		AttackCategory:      attackCategoryFromSaveData(sd.AttackCategory),
		Cost:                int(sd.Cost),
		TargetType:          targetTypeFromSaveData(sd.TargetType),
		Magazine:            int(sd.Magazine),
		MagazineSize:        int(sd.MagazineSize),
		ReloadEffort:        int(sd.ReloadEffort),
		AmmoTag:             sd.AmmoTag,
		LoadedDamageBonus:   int(sd.LoadedDamageBonus),
		LoadedAccuracyBonus: int(sd.LoadedAccuracyBonus),
	}
}

func recipeToSaveData(r gc.Recipe) oapi.SaveDataRecipeComponent {
	inputs := make([]oapi.SaveDataRecipeInputData, len(r.Inputs))
	for i, inp := range r.Inputs {
		inputs[i] = oapi.SaveDataRecipeInputData{Name: inp.Name, Amount: int32(inp.Amount)}
	}
	return oapi.SaveDataRecipeComponent{Inputs: inputs}
}

func recipeFromSaveData(sd oapi.SaveDataRecipeComponent) gc.Recipe {
	inputs := make([]gc.RecipeInput, len(sd.Inputs))
	for i, inp := range sd.Inputs {
		inputs[i] = gc.RecipeInput{Name: inp.Name, Amount: int(inp.Amount)}
	}
	return gc.Recipe{Inputs: inputs}
}

func ammoToSaveData(a gc.Ammo) oapi.SaveDataAmmoComponent {
	return oapi.SaveDataAmmoComponent{
		AmmoTag:       a.AmmoTag,
		DamageBonus:   int32(a.DamageBonus),
		AccuracyBonus: int32(a.AccuracyBonus),
	}
}

func ammoFromSaveData(sd oapi.SaveDataAmmoComponent) gc.Ammo {
	return gc.Ammo{
		AmmoTag:       sd.AmmoTag,
		DamageBonus:   int(sd.DamageBonus),
		AccuracyBonus: int(sd.AccuracyBonus),
	}
}

// ================== アイテム効果コンポーネント変換 ==================

func consumableToSaveData(c gc.Consumable) oapi.SaveDataConsumableComponent {
	return oapi.SaveDataConsumableComponent{
		UsableScene: oapi.UsableScene(c.UsableScene),
		TargetType:  targetTypeToSaveData(c.TargetType),
	}
}

func consumableFromSaveData(sd oapi.SaveDataConsumableComponent) gc.Consumable {
	return gc.Consumable{
		UsableScene: gc.UsableSceneType(sd.UsableScene),
		TargetType:  targetTypeFromSaveData(sd.TargetType),
	}
}

func providesHealingToSaveData(ph gc.ProvidesHealing) oapi.SaveDataProvidesHealingComponent {
	amountData := oapi.SaveDataHealingAmountData{}
	switch a := ph.Amount.(type) {
	case gc.RatioAmount:
		amountData.Type = oapi.Ratio
		ratio := a.Ratio
		amountData.Ratio = &ratio
	case gc.NumeralAmount:
		amountData.Type = oapi.Numeral
		numeral := int32(a.Numeral)
		amountData.Numeral = &numeral
	default:
		panic(fmt.Sprintf("未知のAmounter型: %T", ph.Amount))
	}
	return oapi.SaveDataProvidesHealingComponent{Amount: amountData}
}

func providesHealingFromSaveData(sd oapi.SaveDataProvidesHealingComponent) gc.ProvidesHealing {
	var amount gc.Amounter
	switch sd.Amount.Type {
	case oapi.Ratio:
		if sd.Amount.Ratio != nil {
			amount = gc.RatioAmount{Ratio: *sd.Amount.Ratio}
		}
	case oapi.Numeral:
		if sd.Amount.Numeral != nil {
			amount = gc.NumeralAmount{Numeral: int(*sd.Amount.Numeral)}
		}
	default:
		panic(fmt.Sprintf("未知のHealingAmountData型: %s", sd.Amount.Type))
	}
	return gc.ProvidesHealing{Amount: amount}
}

// ================== マーカーコンポーネント ==================

// emptyMarker はマーカーコンポーネント用の空マップを返す
func emptyMarker() *oapi.SaveDataMarkerComponent {
	m := oapi.SaveDataMarkerComponent{}
	return &m
}
