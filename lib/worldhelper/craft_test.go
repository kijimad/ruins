package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCanCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 必要な素材を作成（木刀レシピは木の棒2個が必要）
	material, _ := SpawnStackable(world, "木の棒", 5, gc.ItemLocationInBackpack)

	// クラフト可能かテスト
	canCraft, err := CanCraft(world, "木刀")
	assert.True(t, canCraft, "十分な素材があるときはクラフト可能であるべき")
	assert.NoError(t, err, "十分な素材があるときはエラーが発生してはいけない")

	// 素材が不足している場合のテスト
	materialComp := world.Components.Item.Get(material).(*gc.Item)
	materialComp.Count = 1 // 木の棒の量を1にする（2個必要なので不足）

	canCraft, err = CanCraft(world, "木刀")
	assert.False(t, canCraft, "素材が不足しているときはクラフト不可能であるべき")
	assert.NoError(t, err, "素材が不足してもエラーは発生しないべき")

	// 存在しないレシピのテスト
	canCraft, err = CanCraft(world, "存在しない武器")
	assert.False(t, canCraft, "存在しないレシピはクラフト不可能であるべき")
	assert.Error(t, err, "存在しないレシピでエラーが発生するべき")
	assert.Contains(t, err.Error(), "レシピが存在しません", "エラーメッセージにレシピ不存在の内容が含まれるべき")

	// クリーンアップ
	world.Manager.DeleteEntity(material)
}

func TestCraft(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 存在しないレシピでのクラフト試行
	result, err := Craft(world, "存在しない武器")
	assert.Nil(t, result, "存在しないレシピでは結果がnilであるべき")
	assert.Error(t, err, "存在しないレシピでエラーが返されるべき")
	assert.Contains(t, err.Error(), "レシピが存在しません", "エラーメッセージにレシピ不存在の内容が含まれるべき")

	// 素材不足でのクラフト試行（木刀は木の棒2個が必要）
	result, err = Craft(world, "木刀")
	assert.Nil(t, result, "素材不足では結果がnilであるべき")
	assert.Error(t, err, "素材不足でエラーが返されるべき")
	assert.Contains(t, err.Error(), "必要素材が足りません", "エラーメッセージに素材不足の内容が含まれるべき")

	// 素材を用意してクラフト成功
	_, _ = SpawnStackable(world, "木の棒", 5, gc.ItemLocationInBackpack)
	result, err = Craft(world, "木刀")
	assert.NotNil(t, result, "素材が十分ならば結果が返されるべき")
	assert.NoError(t, err, "素材が十分ならばエラーは発生しないべき")
}
