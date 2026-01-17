package worldhelper

import (
	"fmt"
	"math/rand/v2"

	ecs "github.com/x-hgg-x/goecs/v2"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/raw"
	w "github.com/kijimaD/ruins/lib/world"
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

	resultEntity, err := SpawnItem(world, name, 1, gc.ItemLocationInBackpack)
	if err != nil {
		return nil, fmt.Errorf("アイテム生成に失敗: %w", err)
	}
	randomize(world, resultEntity)
	if err := consumeMaterials(world, name); err != nil {
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

// consumeMaterials はアイテム合成に必要な素材を消費する
func consumeMaterials(world w.World, goal string) error {
	for _, recipeInput := range requiredMaterials(world, goal) {
		err := ChangeStackableCount(world, recipeInput.Name, -recipeInput.Amount)
		if err != nil {
			return err
		}
	}
	return nil
}

// requiredMaterials は指定したレシピに必要な素材一覧
func requiredMaterials(world w.World, need string) []gc.RecipeInput {
	rawMaster := world.Resources.RawMaster.(*raw.Master)

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

// randomize はアイテムにランダム値を設定する
func randomize(world w.World, entity ecs.Entity) {
	if entity.HasComponent(world.Components.Attack) {
		attack := world.Components.Attack.Get(entity).(*gc.Attack)

		attack.Accuracy += (-10 + rand.IntN(20)) // -10 ~ +9
		attack.Damage += (-5 + rand.IntN(15))    // -5  ~ +9
	}
	if entity.HasComponent(world.Components.Wearable) {
		wearable := world.Components.Wearable.Get(entity).(*gc.Wearable)

		wearable.Defense += (-4 + rand.IntN(20)) // -4 ~ +9
	}
}
