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

	// 必要な素材を作成（木刀レシピは木の棒2個が必要）
	material, _ := lifecycle.SpawnBackpackItem(world, "wooden_stick", 5)

	// クラフト可能かテスト
	canCraft, err := CanCraft(world, "wooden_sword")
	assert.True(t, canCraft, "十分な素材があるときはクラフト可能であるべき")
	require.NoError(t, err, "十分な素材があるときはエラーが発生してはいけない")

	// 素材が不足している場合のテスト。木の棒を1個に減らす。2個必要なので不足する
	require.NoError(t, lifecycle.ChangeItemCount(world, material, -4))

	canCraft, err = CanCraft(world, "wooden_sword")
	assert.False(t, canCraft, "素材が不足しているときはクラフト不可能であるべき")
	require.NoError(t, err, "素材が不足してもエラーは発生しないべき")

	// 存在しないレシピのテスト
	canCraft, err = CanCraft(world, "存在しない武器")
	assert.False(t, canCraft, "存在しないレシピはクラフト不可能であるべき")
	require.Error(t, err, "存在しないレシピでエラーが発生するべき")
	assert.Contains(t, err.Error(), "recipe not found", "エラーメッセージにレシピ不存在の内容が含まれるべき")

	// クリーンアップ
	world.ECS.RemoveEntity(material)
}

func TestCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 存在しないレシピでのクラフト試行
	_, err := Craft(world, "存在しない武器")
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

// TestCraft_StackTwice はスタックアイテムを連続で合成しても
// パニックせず、統合先の生存エンティティが返ることを検証する。
// 2回目の合成で新エンティティが既存スタックへ統合されて削除される回帰ケース。
func TestCraft_StackTwice(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 統合はowner(プレイヤー)配下でのみ行われるため、プレイヤーを用意する
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
	require.NoError(t, err)

	// 回復薬は緑ハーブ×1・黄ハーブ×1で合成できるスタックアイテム
	_, _ = lifecycle.SpawnBackpackItem(world, "green_herb", 2)
	_, _ = lifecycle.SpawnBackpackItem(world, "yellow_herb", 2)

	first, err := Craft(world, "healing_potion")
	require.NoError(t, err, "1回目の合成は成功するべき")
	assert.True(t, world.ECS.Alive(first), "1回目の結果エンティティは生存しているべき")

	// 2回目: 新エンティティが既存スタックへ統合されるが、統合先を結果として返すべき
	second, err := Craft(world, "healing_potion")
	require.NoError(t, err, "2回目の合成もパニックせず成功するべき")
	assert.True(t, world.ECS.Alive(second), "統合されても生存する結果エンティティが返るべき")
	assert.Equal(t, 2, query.GetEntityCount(world, second), "回復薬が2個に統合されているべき")
}
