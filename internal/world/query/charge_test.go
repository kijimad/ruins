package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestChargeRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		charge *gc.Charge
		want   float64
		wantOK bool
	}{
		{name: "上限に対する割合を返す", charge: &gc.Charge{Max: 600, Current: 150}, want: 0.25, wantOK: true},
		{name: "上限を超えても1に収める", charge: &gc.Charge{Max: 600, Current: 700}, want: 1, wantOK: true},
		{name: "残量を持たなければ ok は偽"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			e := world.ECS.NewEntity()
			if tt.charge != nil {
				world.Components.Charge.Add(e, tt.charge)
			}
			got, ok := ChargeRatio(world, e)
			assert.Equal(t, tt.wantOK, ok)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}
