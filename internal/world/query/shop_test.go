package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCalculateBuyPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		baseValue int
		want      consts.Currency
	}{
		{"価値100", 100, 200},
		{"価値50", 50, 100},
		{"価値0", 0, 0},
		{"価値1", 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, CalculateBuyPrice(tt.baseValue))
		})
	}
}

func TestCalculateSellPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		baseValue int
		want      consts.Currency
	}{
		{"価値100", 100, 50},
		{"価値50", 50, 25},
		{"価値0", 0, 0},
		{"価値1は切り捨てで0", 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, CalculateSellPrice(tt.baseValue))
		})
	}
}

// TestSellPrice は表示と取引で共有する売値を検証する。交渉倍率・個数連動・無価値な品の対価0を覆う
func TestSellPrice(t *testing.T) {
	t.Parallel()

	t.Run("価値0の売値は0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		item := world.ECS.NewEntity()
		world.Components.Value.Add(item, &gc.Value{Value: 0})

		assert.Equal(t, consts.Currency(0), SellPrice(world, player, item), "無価値な品の対価は0")
	})

	t.Run("倍率なしは価値の半分", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		item := world.ECS.NewEntity()
		world.Components.Value.Add(item, &gc.Value{Value: 100})

		assert.Equal(t, consts.Currency(50), SellPrice(world, player, item), "CalculateSellPrice(100)=50")
	})

	t.Run("交渉スキルの売値倍率が乗る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.CharModifiers.Add(player, &gc.CharModifiers{SellPrice: 200})
		item := world.ECS.NewEntity()
		world.Components.Value.Add(item, &gc.Value{Value: 100})

		assert.Equal(t, consts.Currency(100), SellPrice(world, player, item), "売値倍率200%で50が倍額100")
	})

	t.Run("個数分だけ価値が乗る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		item := world.ECS.NewEntity()
		world.Components.Value.Add(item, &gc.Value{Value: 100})
		world.Components.Stackable.Add(item, &gc.Stackable{Count: 3})

		assert.Equal(t, consts.Currency(150), SellPrice(world, player, item), "価値100×3個の半分")
	})
}

// TestBuyPrice は表示と取引で共有する買値を検証する。交渉倍率・個数連動を覆う
func TestBuyPrice(t *testing.T) {
	t.Parallel()

	t.Run("倍率なしは価値の2倍", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		item := world.ECS.NewEntity()
		world.Components.Value.Add(item, &gc.Value{Value: 100})

		assert.Equal(t, consts.Currency(200), BuyPrice(world, player, item), "CalculateBuyPrice(100)=200")
	})

	t.Run("交渉スキルの買値倍率が乗る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.CharModifiers.Add(player, &gc.CharModifiers{BuyPrice: 50})
		item := world.ECS.NewEntity()
		world.Components.Value.Add(item, &gc.Value{Value: 100})

		assert.Equal(t, consts.Currency(100), BuyPrice(world, player, item), "買値倍率50%で200が半額100")
	})
}

func TestGetItemValue(t *testing.T) {
	t.Parallel()

	t.Run("Valueコンポーネントがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.Value.Add(entity, &gc.Value{Value: 80})

		assert.Equal(t, 80, GetItemValue(world, entity))
	})

	t.Run("Valueコンポーネントがない場合は0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()

		assert.Equal(t, 0, GetItemValue(world, entity))
	})
}
