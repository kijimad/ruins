package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSwitchMarker(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		sw   *gc.Switch
		want string
	}{
		{name: "スイッチを持たない物は何も添えない", sw: nil, want: ""},
		{name: "オンの物", sw: &gc.Switch{On: true}, want: "Switched on"},
		{name: "オフの物", sw: &gc.Switch{On: false}, want: "Switched off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			e := world.ECS.NewEntity()
			if tt.sw != nil {
				world.Components.Switch.Add(e, tt.sw)
			}
			want := ""
			if tt.want != "" {
				want = T(world, i18n.Msgid(tt.want))
			}
			assert.Equal(t, want, SwitchMarker(world, e))
		})
	}
}
