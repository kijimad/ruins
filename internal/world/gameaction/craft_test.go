package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 必要な素材を作成（木刀レシピは木の棒2個が必要）
	material, _ := lifecycle.SpawnBackpackItem(world, "木の棒", 5)

	// クラフト可能かテスト
	canCraft, err := CanCraft(world, "木刀")
	assert.True(t, canCraft, "十分な素材があるときはクラフト可能であるべき")
	require.NoError(t, err, "十分な素材があるときはエラーが発生してはいけない")

	// 素材が不足している場合のテスト
	materialComp := world.Components.Stackable.Get(material)
	materialComp.Count = 1 // 木の棒の量を1にする（2個必要なので不足）

	canCraft, err = CanCraft(world, "木刀")
	assert.False(t, canCraft, "素材が不足しているときはクラフト不可能であるべき")
	require.NoError(t, err, "素材が不足してもエラーは発生しないべき")

	// 存在しないレシピのテスト
	canCraft, err = CanCraft(world, "存在しない武器")
	assert.False(t, canCraft, "存在しないレシピはクラフト不可能であるべき")
	require.Error(t, err, "存在しないレシピでエラーが発生するべき")
	assert.Contains(t, err.Error(), "レシピが存在しません", "エラーメッセージにレシピ不存在の内容が含まれるべき")

	// クリーンアップ
	world.World.RemoveEntity(material)
}

func TestCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 存在しないレシピでのクラフト試行
	_, err := Craft(world, "存在しない武器")
	require.Error(t, err, "存在しないレシピでエラーが返されるべき")
	assert.Contains(t, err.Error(), "レシピが存在しません", "エラーメッセージにレシピ不存在の内容が含まれるべき")

	// 素材不足でのクラフト試行（木刀は木の棒2個が必要）
	_, err = Craft(world, "木刀")
	require.Error(t, err, "素材不足でエラーが返されるべき")
	assert.Contains(t, err.Error(), "必要素材が足りません", "エラーメッセージに素材不足の内容が含まれるべき")

	// 素材を用意してクラフト成功
	_, _ = lifecycle.SpawnBackpackItem(world, "木の棒", 5)
	result, err := Craft(world, "木刀")
	assert.NotEqual(t, consts.InvalidEntity, result, "素材が十分ならば有効なエンティティが返されるべき")
	assert.NoError(t, err, "素材が十分ならばエラーは発生しないべき")
}
