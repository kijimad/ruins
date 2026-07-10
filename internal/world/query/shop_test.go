package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCalculateBuyPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		baseValue int
		want      int
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
		want      int
	}{
		{"価値100", 100, 50},
		{"価値50", 50, 25},
		{"価値0", 0, 0},
		{"価値1", 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, CalculateSellPrice(tt.baseValue))
		})
	}
}

func TestGetItemValue(t *testing.T) {
	t.Parallel()

	t.Run("Valueコンポーネントがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.World.NewEntity()
		world.Components.Value.Add(entity, &gc.Value{Value: 80})

		assert.Equal(t, 80, GetItemValue(world, entity))
	})

	t.Run("Valueコンポーネントがない場合は0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.World.NewEntity()

		assert.Equal(t, 0, GetItemValue(world, entity))
	})
}
