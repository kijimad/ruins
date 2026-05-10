package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestPreviewEndRun(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	prof := world.Resources.RawMaster.Raws.Professions[0]
	player, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	require.NoError(t, ApplyProfession(world, player, prof))

	// バックパックにアイテムを追加する
	item1, err := SpawnItem(world, "回復薬", 1, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)
	item2, err := SpawnItem(world, "回復薬", 1, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)

	// プレビューを生成する
	result, err := PreviewEndRun(world, player)
	require.NoError(t, err)

	assert.Greater(t, result.Total, 0, "売却合計が0より大きい")

	// 追加した回復薬が個別エンティティとして含まれていることを確認する
	healingCount := 0
	for _, item := range result.Items {
		if item.Name == "回復薬" {
			healingCount++
			assert.NotZero(t, item.Entity, "エンティティが設定されている")
		}
	}
	assert.GreaterOrEqual(t, healingCount, 2, "追加した回復薬が個別に含まれている")

	// プレビュー段階ではエンティティが残っていることを確認する
	assert.True(t, item1.HasComponent(world.Components.Item), "プレビュー段階ではアイテム1が残っている")
	assert.True(t, item2.HasComponent(world.Components.Item), "プレビュー段階ではアイテム2が残っている")
}

func TestExecuteEndRun(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	prof := world.Resources.RawMaster.Raws.Professions[0]
	player, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	require.NoError(t, ApplyProfession(world, player, prof))

	walletBefore := world.Components.Wallet.Get(player).(*gc.Wallet).Currency

	// プレビュー → 実行
	result, err := PreviewEndRun(world, player)
	require.NoError(t, err)

	err = ExecuteEndRun(world, player, result.Total)
	require.NoError(t, err)

	// 所持金が増えていることを確認する
	walletAfter := world.Components.Wallet.Get(player).(*gc.Wallet).Currency
	assert.Equal(t, walletBefore+result.Total, walletAfter, "売却金額が所持金に加算されている")

	// 職業が再適用されていることを確認する
	hasEquipped := false
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationEquipped,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		hasEquipped = true
	}))
	if len(prof.Equips) > 0 {
		assert.True(t, hasEquipped, "職業の初期装備が再適用されている")
	}
}

func TestExecuteEndRunNoItems(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	prof := world.Resources.RawMaster.Raws.Professions[0]
	player.AddComponent(world.Components.Profession, &gc.Profession{ID: prof.Id})

	result, err := PreviewEndRun(world, player)
	require.NoError(t, err)

	err = ExecuteEndRun(world, player, result.Total)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, result.Total, 0, "合計が0以上")
}
