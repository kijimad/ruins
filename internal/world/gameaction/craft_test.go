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

	// 必要な素材を作成（木刀レシピは木の棒2個が必要）
	material, _ := lifecycle.SpawnBackpackItem(world, "wooden_stick", 5)

	// クラフト可能かテスト
	canCraft, err := CanCraft(world, "wooden_sword")
	assert.True(t, canCraft, "十分な素材があるときはクラフト可能であるべき")
	require.NoError(t, err, "十分な素材があるときはエラーが発生してはいけない")

	// 素材が無い場合のテスト。実消費量はスキルと能力で変動するので、無しの状態で判定する
	require.NoError(t, lifecycle.ChangeItemCount(world, material, -5))

	canCraft, err = CanCraft(world, "wooden_sword")
	assert.False(t, canCraft, "素材が無いときはクラフト不可能であるべき")
	require.NoError(t, err, "素材が無くてもエラーは発生しないべき")

	// 存在しないレシピのテスト
	canCraft, err = CanCraft(world, "存在しない武器")
	assert.False(t, canCraft, "存在しないレシピはクラフト不可能であるべき")
	require.Error(t, err, "存在しないレシピでエラーが発生するべき")
	assert.Contains(t, err.Error(), "recipe not found", "エラーメッセージにレシピ不存在の内容が含まれるべき")
}

func TestCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	// 存在しないレシピでのクラフト試行
	_, err = Craft(world, "存在しない武器")
	require.Error(t, err, "存在しないレシピでエラーが返されるべき")
	assert.Contains(t, err.Error(), "recipe not found", "エラーメッセージにレシピ不存在の内容が含まれるべき")

	// 素材不足でのクラフト試行（木刀は木の棒2個が必要）
	_, err = Craft(world, "wooden_sword")
	require.Error(t, err, "素材不足でエラーが返されるべき")
	assert.Contains(t, err.Error(), "insufficient materials", "エラーメッセージに素材不足の内容が含まれるべき")

	// 素材を用意してクラフト成功
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
