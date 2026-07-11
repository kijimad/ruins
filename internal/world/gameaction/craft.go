package gameaction

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// Craft はアイテムをクラフトする
func Craft(world w.World, name string) (ecs.Entity, error) {
	canCraft, err := CanCraft(world, name)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if !canCraft {
		return consts.InvalidEntity, fmt.Errorf("必要素材が足りません")
	}

	craftCostPct, smithQualityPct := 100, 100
	player, playerErr := query.GetPlayerEntity(world)
	if playerErr == nil && world.Components.CharModifiers.Has(player) {
		mods := world.Components.CharModifiers.Get(player)
		craftCostPct = mods.CraftCost
		smithQualityPct = mods.SmithQuality
	}

	resultEntity, err := lifecycle.SpawnBackpackItem(world, name, 1)
	if err != nil {
		return consts.InvalidEntity, fmt.Errorf("アイテム生成に失敗: %w", err)
	}
	// Stackableアイテムの合成では、SpawnBackpackItem内の統合処理で
	// resultEntityが既存スタックへ統合されて削除されることがある。
	// その場合は統合先の生存エンティティを結果として扱う
	if !world.World.Alive(resultEntity) {
		if survivor, found := query.FindStackableInInventory(world, name); found {
			resultEntity = survivor
		}
	}
	randomize(world, resultEntity, smithQualityPct)
	if err := consumeMaterials(world, name, craftCostPct); err != nil {
		return consts.InvalidEntity, fmt.Errorf("素材消費に失敗: %w", err)
	}

	return resultEntity, nil
}

// CanCraft は所持数と必要数を比較してクラフト可能か判定する
func CanCraft(world w.World, name string) (bool, error) {
	required := requiredMaterials(world, name)
	if len(required) == 0 {
		return false, fmt.Errorf("レシピが存在しません: %s", name)
	}

	for _, recipeInput := range required {
		entity, found := query.FindStackableInInventory(world, recipeInput.Name)
		if !found {
			return false, nil
		}
		count := query.GetEntityCount(world, entity)
		if count < recipeInput.Amount {
			return false, nil
		}
	}

	return true, nil
}

// consumeMaterials はアイテム合成に必要な素材を消費する。
// craftCostPctは素材消費量の倍率%で、100が基準。低いほど素材が節約できる。
func consumeMaterials(world w.World, goal string, craftCostPct int) error {
	for _, recipeInput := range requiredMaterials(world, goal) {
		consumed := max(recipeInput.Amount*craftCostPct/100, 1)
		err := lifecycle.ChangeStackableCount(world, recipeInput.Name, -consumed)
		if err != nil {
			return err
		}
	}
	return nil
}

// requiredMaterials は指定したレシピに必要な素材一覧
func requiredMaterials(world w.World, need string) []gc.RecipeInput {
	rawMaster := world.Resources.RawMaster

	spec, err := raw.NewRecipeSpec(rawMaster, need)
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
	// Stackableなアイテムを合成した場合、SpawnBackpackItem内の統合処理で
	// このエンティティが既存スタックに統合されて削除されていることがある。
	// 統合済みStackableに武器/防具の乱数化は不要なので、死亡していれば何もしない
	if !world.World.Alive(entity) {
		return
	}

	qualityBonus := (smithQualityPct - 100) / 10

	if world.Components.Melee.Has(entity) {
		melee := world.Components.Melee.Get(entity)
		melee.Accuracy += (-10 + rand.IntN(20)) + qualityBonus
		melee.Damage += (-5 + rand.IntN(15)) + qualityBonus
	}
	if world.Components.Fire.Has(entity) {
		fire := world.Components.Fire.Get(entity)
		fire.Accuracy += (-10 + rand.IntN(20)) + qualityBonus
		fire.Damage += (-5 + rand.IntN(15)) + qualityBonus
	}
	if world.Components.Wearable.Has(entity) {
		wearable := world.Components.Wearable.Get(entity)
		wearable.Defense += (-4 + rand.IntN(20)) + qualityBonus
	}
}
