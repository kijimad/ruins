package worldhelper

import (
	"fmt"
	"math/rand/v2"

	ecs "github.com/x-hgg-x/goecs/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// Craft はアイテムをクラフトする
func Craft(world w.World, name string) (*ecs.Entity, error) {
	canCraft, err := CanCraft(world, name)
	if err != nil {
		// レシピが存在しない場合
		return nil, err
	}
	if !canCraft {
		// 素材不足の場合
		return nil, fmt.Errorf("必要素材が足りません")
	}

	// プレイヤーのCharModifiersを取得する
	craftCostPct, smithQualityPct := 100, 100
	player, playerErr := GetPlayerEntity(world)
	if playerErr == nil && player.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.Get(player).(*gc.CharModifiers)
		craftCostPct = mods.CraftCost
		smithQualityPct = mods.SmithQuality
	}

	resultEntity, err := SpawnItem(world, name, 1, gc.ItemLocationInPlayerBackpack)
	if err != nil {
		return nil, fmt.Errorf("アイテム生成に失敗: %w", err)
	}
	randomize(world, resultEntity, smithQualityPct)
	if err := consumeMaterials(world, name, craftCostPct); err != nil {
		return nil, fmt.Errorf("素材消費に失敗: %w", err)
	}

	return &resultEntity, nil
}

// CanCraft は所持数と必要数を比較してクラフト可能か判定する
func CanCraft(world w.World, name string) (bool, error) {
	required := requiredMaterials(world, name)
	// レシピが存在しない場合はエラー
	if len(required) == 0 {
		return false, fmt.Errorf("レシピが存在しません: %s", name)
	}

	// 素材不足をチェックする。素材不足はエラーではなくfalseを返す
	for _, recipeInput := range required {
		entity, found := FindStackableInInventory(world, recipeInput.Name)
		if !found {
			return false, nil
		}
		item := world.Components.Item.Get(entity).(*gc.Item)
		if item.Count < recipeInput.Amount {
			return false, nil
		}
	}

	return true, nil
}

// consumeMaterials はアイテム合成に必要な素材を消費する。
// craftCostPctは素材消費量の倍率%で、100が基準。低いほど素材が節約できる。
func consumeMaterials(world w.World, goal string, craftCostPct int) error {
	for _, recipeInput := range requiredMaterials(world, goal) {
		consumed := recipeInput.Amount * craftCostPct / 100
		if consumed < 1 {
			consumed = 1
		}
		err := ChangeStackableCount(world, recipeInput.Name, -consumed)
		if err != nil {
			return err
		}
	}
	return nil
}

// requiredMaterials は指定したレシピに必要な素材一覧
func requiredMaterials(world w.World, need string) []gc.RecipeInput {
	rawMaster := world.Resources.RawMaster

	// RawMasterからレシピを取得
	spec, err := rawMaster.NewRecipeSpec(need)
	if err != nil {
		return []gc.RecipeInput{}
	}

	if spec.Recipe == nil {
		return []gc.RecipeInput{}
	}

	return spec.Recipe.Inputs
}

// randomize はアイテムにランダム値を設定する。
// smithQualityPctは品質倍率%で、100が基準。高いほどボーナスが大きくなる。
func randomize(world w.World, entity ecs.Entity, smithQualityPct int) {
	// 品質ボーナス: 100%→+0, 130%→+3, 50%→-5
	qualityBonus := (smithQualityPct - 100) / 10

	if entity.HasComponent(world.Components.Melee) {
		melee := world.Components.Melee.Get(entity).(*gc.Melee)
		melee.Accuracy += (-10 + rand.IntN(20)) + qualityBonus
		melee.Damage += (-5 + rand.IntN(15)) + qualityBonus
	}
	if entity.HasComponent(world.Components.Fire) {
		fire := world.Components.Fire.Get(entity).(*gc.Fire)
		fire.Accuracy += (-10 + rand.IntN(20)) + qualityBonus
		fire.Damage += (-5 + rand.IntN(15)) + qualityBonus
	}
	if entity.HasComponent(world.Components.Wearable) {
		wearable := world.Components.Wearable.Get(entity).(*gc.Wearable)

		wearable.Defense += (-4 + rand.IntN(20)) + qualityBonus // -4 ~ +9 + 品質ボーナス
	}
}
