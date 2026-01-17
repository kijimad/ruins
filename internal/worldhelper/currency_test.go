package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestAddCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 100})

	// 通貨を追加
	err := addCurrency(world, player, 50)
	assert.NoError(t, err)

	// 結果を検証
	currency := GetCurrency(world, player)
	assert.Equal(t, 150, currency, "通貨が150になるべき")

	// クリーンアップ
	world.Manager.DeleteEntity(player)
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 200})

	// 通貨を取得
	currency := GetCurrency(world, player)
	assert.Equal(t, 200, currency, "通貨が200であるべき")

	// Walletがない場合
	playerWithoutWallet := world.Manager.NewEntity()
	currency = GetCurrency(world, playerWithoutWallet)
	assert.Equal(t, 0, currency, "Walletがない場合は0を返すべき")

	// クリーンアップ
	world.Manager.DeleteEntity(player)
	world.Manager.DeleteEntity(playerWithoutWallet)
}

func TestHasCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 100})

	// 通貨チェック
	assert.True(t, HasCurrency(world, player, 50), "50以上持っているべき")
	assert.True(t, HasCurrency(world, player, 100), "100以上持っているべき")
	assert.False(t, HasCurrency(world, player, 101), "101以上は持っていない")

	// クリーンアップ
	world.Manager.DeleteEntity(player)
}

func TestConsumeCurrency(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成してWalletを追加
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 100})

	// 通貨を消費（成功）
	success := ConsumeCurrency(world, player, 50)
	assert.True(t, success, "消費が成功するべき")
	assert.Equal(t, 50, GetCurrency(world, player), "残り50になるべき")

	// 通貨を消費（失敗：足りない）
	success = ConsumeCurrency(world, player, 100)
	assert.False(t, success, "消費が失敗するべき")
	assert.Equal(t, 50, GetCurrency(world, player), "残り50のまま変わらないべき")

	// 通貨を消費（成功：ちょうど）
	success = ConsumeCurrency(world, player, 50)
	assert.True(t, success, "消費が成功するべき")
	assert.Equal(t, 0, GetCurrency(world, player), "残り0になるべき")

	// クリーンアップ
	world.Manager.DeleteEntity(player)
}

func TestCurrencyOperationsWithoutWallet(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// Walletを持たないエンティティ
	entity := world.Manager.NewEntity()

	// 各操作がエラーを返すことを確認
	err := addCurrency(world, entity, 100)
	assert.Error(t, err, "Walletがない場合はエラーを返すべき")
	assert.Equal(t, 0, GetCurrency(world, entity), "Walletがないので0")

	assert.False(t, HasCurrency(world, entity, 1), "Walletがないのでfalse")
	assert.False(t, ConsumeCurrency(world, entity, 1), "Walletがないのでfalse")

	// クリーンアップ
	world.Manager.DeleteEntity(entity)
}

func TestFormatCurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   int
		expected string
	}{
		{"0", 0, "CZ 0"},
		{"1桁", 5, "CZ 5"},
		{"2桁", 99, "CZ 99"},
		{"3桁", 999, "CZ 999"},
		{"4桁（カンマ1つ）", 1000, "CZ 1,000"},
		{"5桁", 12345, "CZ 12,345"},
		{"6桁", 100204, "CZ 100,204"},
		{"7桁", 1234567, "CZ 1,234,567"},
		{"8桁", 10000000, "CZ 10,000,000"},
		{"負の数", -1234, "CZ -1,234"},
		{"負の数（大きい）", -1234567, "CZ -1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FormatCurrency(tt.amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}
