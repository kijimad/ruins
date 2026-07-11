package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewEndRun(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	prof := (*world.Resources.RawMaster.Professions)[0]
	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	require.NoError(t, ApplyProfession(world, player, prof))

	// バックパックにStackableアイテムを追加する
	const healingPotion = "回復薬"
	_, err = lifecycle.SpawnBackpackItem(world, healingPotion, 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, healingPotion, 1)
	require.NoError(t, err)

	// プレビューを生成する
	result, err := PreviewEndRun(world, player)
	require.NoError(t, err)

	assert.Positive(t, result.Total, "売却合計が0より大きい")

	// 回復薬はStackable統合により1エンティティになっている
	healingCount := 0
	for _, item := range result.Items {
		if item.Name == healingPotion {
			healingCount++
			assert.NotZero(t, item.Entity, "エンティティが設定されている")
		}
	}
	assert.Equal(t, 1, healingCount, "回復薬は統合されて1エンティティ")
}

func TestExecuteEndRun(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	prof := (*world.Resources.RawMaster.Professions)[0]
	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	require.NoError(t, ApplyProfession(world, player, prof))

	walletBefore := world.Components.Wallet.Get(player).Currency

	// プレビュー → 実行
	result, err := PreviewEndRun(world, player)
	require.NoError(t, err)

	err = ExecuteEndRun(world, player, result.Total)
	require.NoError(t, err)

	// 所持金が増えていることを確認する
	walletAfter := world.Components.Wallet.Get(player).Currency
	assert.Equal(t, walletBefore+result.Total, walletAfter, "売却金額が所持金に加算されている")

	// 職業が再適用されていることを確認する
	hasEquipped := false
	equippedQuery := ecs.NewFilter1[gc.LocationEquipped](world.ECS).Query()
	for equippedQuery.Next() {
		hasEquipped = true
	}
	if len(prof.Equips) > 0 {
		assert.True(t, hasEquipped, "職業の初期装備が再適用されている")
	}
}

func TestExecuteEndRunNoItems(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	prof := (*world.Resources.RawMaster.Professions)[0]
	world.Components.Profession.Add(player, &gc.Profession{ID: prof.Id})

	result, err := PreviewEndRun(world, player)
	require.NoError(t, err)

	err = ExecuteEndRun(world, player, result.Total)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, result.Total, 0, "合計が0以上")
}
