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
		return gc.InvalidEntity, err
	}
	if !canCraft {
		return gc.InvalidEntity, fmt.Errorf("insufficient materials")
	}

	craftCostPct, smithQualityPct := playerCraftMods(world)

	// 完成品を生成する前に素材を消費する。生成が先だと、CraftCost が100%を超えて消費が
	// 失敗したとき完成品だけが手元に残る。CanCraft と同じ実消費量を先に引くことで整合させる
	if err := consumeMaterials(world, name, craftCostPct); err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to consume materials: %w", err)
	}

	resultEntity, err := lifecycle.SpawnBackpackItem(world, name, 1)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to generate item: %w", err)
	}
	randomize(world, resultEntity, smithQualityPct)

	return resultEntity, nil
}

// CanCraft は所持数と実消費量を比較してクラフト可能か判定する。実消費量は CraftCost 倍率込みで、
// consumeMaterials と同じ式にして判定と消費のズレをなくす
func CanCraft(world w.World, name string) (bool, error) {
	required := requiredMaterials(world, name)
	if len(required) == 0 {
		return false, fmt.Errorf("recipe not found: %s", name)
	}

	craftCostPct, _ := playerCraftMods(world)
	for _, recipeInput := range required {
		entity, found := query.FindStackInInventory(world, recipeInput.ID)
		if !found {
			return false, nil
		}
		count := query.GetEntityCount(world, entity)
		if count < craftConsumedAmount(craftCostPct, recipeInput.Amount) {
			return false, nil
		}
	}

	return true, nil
}

// playerCraftMods はプレイヤーのクラフト倍率を返す。プレイヤーや修正が無ければ基準値
func playerCraftMods(world w.World) (craftCost, smithQuality consts.Percent) {
	craftCost, smithQuality = consts.PercentBase, consts.PercentBase
	player, err := query.GetPlayerEntity(world)
	if err == nil && world.Components.CharModifiers.Has(player) {
		mods := world.Components.CharModifiers.Get(player)
		craftCost = mods.CraftCost
		smithQuality = mods.SmithQuality
	}
	return craftCost, smithQuality
}

// craftConsumedAmount は CraftCost 倍率を掛けた実消費量。最低でも1は消費する
func craftConsumedAmount(craftCostPct consts.Percent, amount int) int {
	return max(craftCostPct.ApplyInt(amount), 1)
}

// consumeMaterials はアイテムクラフトに必要な素材を消費する。
// craftCostPctは素材消費量の倍率%で、100が基準。低いほど素材が節約できる。
func consumeMaterials(world w.World, goal string, craftCostPct consts.Percent) error {
	for _, recipeInput := range requiredMaterials(world, goal) {
		consumed := craftConsumedAmount(craftCostPct, recipeInput.Amount)
		err := lifecycle.ChangeStackCount(world, recipeInput.ID, -consumed)
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
func randomize(world w.World, entity ecs.Entity, smithQualityPct consts.Percent) {
	// 基準からの乖離を10%刻みでボーナス段階に換算する。倍率そのものでなく段数なので int で扱う
	qualityBonus := (int(smithQualityPct) - int(consts.PercentBase)) / 10

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
