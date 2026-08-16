package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 100})

	// 通貨を追加
	err := AddCurrency(world, player, 50)
	require.NoError(t, err)

	// 結果を検証
	currency := GetCurrency(world, player)
	assert.Equal(t, consts.Currency(150), currency, "通貨が150になるべき")

	// クリーンアップ
	world.ECS.RemoveEntity(player)
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 200})

	// 通貨を取得
	currency := GetCurrency(world, player)
	assert.Equal(t, consts.Currency(200), currency, "通貨が200であるべき")

	// Walletがない場合
	playerWithoutWallet := world.ECS.NewEntity()
	currency = GetCurrency(world, playerWithoutWallet)
	assert.Equal(t, consts.Currency(0), currency, "Walletがない場合は0を返すべき")

	// クリーンアップ
	world.ECS.RemoveEntity(player)
	world.ECS.RemoveEntity(playerWithoutWallet)
}

func TestHasCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 100})

	// 通貨チェック
	assert.True(t, HasCurrency(world, player, 50), "50以上持っているべき")
	assert.True(t, HasCurrency(world, player, 100), "100以上持っているべき")
	assert.False(t, HasCurrency(world, player, 101), "101以上は持っていない")

	// クリーンアップ
	world.ECS.RemoveEntity(player)
}

func TestConsumeCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 100})

	// 通貨を消費（成功）
	success := ConsumeCurrency(world, player, 50)
	assert.True(t, success, "消費が成功するべき")
	assert.Equal(t, consts.Currency(50), GetCurrency(world, player), "残り50になるべき")

	// 通貨を消費（失敗：足りない）
	success = ConsumeCurrency(world, player, 100)
	assert.False(t, success, "消費が失敗するべき")
	assert.Equal(t, consts.Currency(50), GetCurrency(world, player), "残り50のまま変わらないべき")

	// 通貨を消費（成功：ちょうど）
	success = ConsumeCurrency(world, player, 50)
	assert.True(t, success, "消費が成功するべき")
	assert.Equal(t, consts.Currency(0), GetCurrency(world, player), "残り0になるべき")

	// クリーンアップ
	world.ECS.RemoveEntity(player)
}

func TestCurrencyOperationsWithoutWallet(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// Walletを持たないエンティティ
	entity := world.ECS.NewEntity()

	// 各操作がエラーを返すことを確認
	err := AddCurrency(world, entity, 100)
	require.Error(t, err, "Walletがない場合はエラーを返すべき")
	assert.Equal(t, consts.Currency(0), GetCurrency(world, entity), "Walletがないので0")

	assert.False(t, HasCurrency(world, entity, 1), "Walletがないのでfalse")
	assert.False(t, ConsumeCurrency(world, entity, 1), "Walletがないのでfalse")

	// クリーンアップ
	world.ECS.RemoveEntity(entity)
}
