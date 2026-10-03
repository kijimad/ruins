package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	material, _ := lifecycle.SpawnBackpackItem(world, "wooden_stick", 5)

	canCraft, err := CanCraft(world, "wooden_sword")
	assert.True(t, canCraft, "十分な素材があるときはクラフト可能であるべき")
	require.NoError(t, err, "十分な素材があるときはエラーが発生してはいけない")

	// 実消費量はスキルと能力で変動するので、素材無しの状態で判定する
	require.NoError(t, lifecycle.ChangeItemCount(world, material, -5))

	canCraft, err = CanCraft(world, "wooden_sword")
	assert.False(t, canCraft, "素材が無いときはクラフト不可能であるべき")
	require.NoError(t, err, "素材が無くてもエラーは発生しないべき")

	canCraft, err = CanCraft(world, "存在しない武器")
	assert.False(t, canCraft, "存在しないレシピはクラフト不可能であるべき")
	require.Error(t, err, "存在しないレシピでエラーが発生するべき")
	require.ErrorIs(t, err, errRecipeNotFound, "レシピ不存在のエラーを返すべき")
}

func TestCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	_, err = Craft(world, "存在しない武器")
	require.Error(t, err, "存在しないレシピでエラーが返されるべき")
	require.ErrorIs(t, err, errRecipeNotFound, "レシピ不存在のエラーを返すべき")

	_, err = Craft(world, "wooden_sword")
	require.Error(t, err, "素材不足でエラーが返されるべき")
	require.ErrorIs(t, err, errInsufficientMaterials, "素材不足のエラーを返すべき")

	_, _ = lifecycle.SpawnBackpackItem(world, "wooden_stick", 5)
	result, err := Craft(world, "wooden_sword")
	assert.NotEqual(t, gc.InvalidEntity, result, "素材が十分ならば有効なエンティティが返されるべき")
	assert.NoError(t, err, "素材が十分ならばエラーは発生しないべき")
}

// TestCraft_StackTwice はスタックアイテムを連続でクラフトしても
// パニックせず、統合先の生存エンティティが返ることを検証する。
// 2回目のクラフトで新エンティティが既存スタックへ統合されて削除される回帰ケース。
func TestCraft_StackTwice(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 統合はowner(プレイヤー)配下でのみ行われるため、プレイヤーを用意する
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
	require.NoError(t, err)

	// 回復薬は緑ハーブ×1・黄ハーブ×1でクラフトできるスタックアイテム
	_, _ = lifecycle.SpawnBackpackItem(world, "green_herb", 2)
	_, _ = lifecycle.SpawnBackpackItem(world, "yellow_herb", 2)

	first, err := Craft(world, "healing_potion")
	require.NoError(t, err, "1回目のクラフトは成功するべき")
	assert.True(t, world.ECS.Alive(first), "1回目の結果エンティティは生存しているべき")

	// 2回目: 新エンティティが既存スタックへ統合されるが、統合先を結果として返すべき
	second, err := Craft(world, "healing_potion")
	require.NoError(t, err, "2回目のクラフトもパニックせず成功するべき")
	assert.True(t, world.ECS.Alive(second), "統合されても生存する結果エンティティが返るべき")
	assert.Equal(t, 2, query.GetEntityCount(world, second), "回復薬が2個に統合されているべき")
}

func TestCraft_クラフト倍率で実消費量が減る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	// 木刀は木の棒×2が基本必要量。ちょうどだけ持たせる
	required := requiredMaterials(world, "wooden_sword")
	require.NotEmpty(t, required)
	for _, in := range required {
		_, err = lifecycle.SpawnBackpackItem(world, in.ID, in.Amount)
		require.NoError(t, err)
	}

	// クラフトLv10: CraftCost = 100 + 10*(-3) = 70 以下。器用が高いとさらに下がる。
	// 木の棒2本の実消費量 = max(2*70/100, 1) = 1 で、1本残してクラフトできる
	world.Components.Skills.Get(player).Get(gc.SkillCrafting).Value = 10

	result, err := Craft(world, "wooden_sword")
	require.NoError(t, err, "実消費量が減りクラフトは成功する")
	assert.True(t, world.ECS.Alive(result), "完成品エンティティが返る")

	stick, found := query.FindStackInInventory(world, "wooden_stick")
	require.True(t, found, "素材が残る")
	assert.Equal(t, 1, query.GetEntityCount(world, stick), "実消費量1で木の棒が1本残る")
}

// TestCraft_射撃武器をクラフトするとMeleeとFireの両方に乱数調整が入る は randomize が
// Melee・Fire の両成分を持つ完成品でそれぞれ基準値から式どおりの範囲内に補正することを検証する。
// レイガンは近接成分と射撃成分を両方持つ数少ないレシピで、Fire 分岐を実クラフト経路で踏める。
func TestCraft_射撃武器をクラフトするとMeleeとFireの両方に乱数調整が入る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	required := requiredMaterials(world, "ray_gun")
	require.NotEmpty(t, required)
	for _, in := range required {
		_, err = lifecycle.SpawnBackpackItem(world, in.ID, in.Amount)
		require.NoError(t, err)
	}

	result, err := Craft(world, "ray_gun")
	require.NoError(t, err)
	require.True(t, world.ECS.Alive(result))

	// 基準はレイガンの Melee: accuracy=90, damage=4。乱数調整は
	// accuracy: -10〜+9、damage: -5〜+9、品質ボーナスは基準プレイヤーで0
	require.True(t, world.Components.Melee.Has(result), "レイガンは近接成分も持つ")
	melee := world.Components.Melee.Get(result)
	assert.GreaterOrEqual(t, melee.Accuracy, 90-10)
	assert.LessOrEqual(t, melee.Accuracy, 90+9)
	assert.GreaterOrEqual(t, melee.Damage, 4-5)
	assert.LessOrEqual(t, melee.Damage, 4+9)

	// 基準はレイガンの Fire: accuracy=90, damage=20。調整幅はMeleeと同じ式
	require.True(t, world.Components.Fire.Has(result), "レイガンは射撃武器")
	fire := world.Components.Fire.Get(result)
	assert.GreaterOrEqual(t, fire.Accuracy, 90-10)
	assert.LessOrEqual(t, fire.Accuracy, 90+9)
	assert.GreaterOrEqual(t, fire.Damage, 20-5)
	assert.LessOrEqual(t, fire.Damage, 20+9)
}

func TestCraft_防具をクラフトするとWearableに乱数調整が入る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	required := requiredMaterials(world, "western_armor")
	require.NotEmpty(t, required)
	for _, in := range required {
		_, err = lifecycle.SpawnBackpackItem(world, in.ID, in.Amount)
		require.NoError(t, err)
	}

	result, err := Craft(world, "western_armor")
	require.NoError(t, err)
	require.True(t, world.ECS.Alive(result))

	// 基準は西洋鎧の Wearable: defense=8。乱数調整は -4〜+15、品質ボーナスは基準プレイヤーで0
	require.True(t, world.Components.Wearable.Has(result), "西洋鎧は防具")
	wearable := world.Components.Wearable.Get(result)
	assert.GreaterOrEqual(t, wearable.Defense, 8-4)
	assert.LessOrEqual(t, wearable.Defense, 8+15)
}
