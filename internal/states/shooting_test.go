package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShootingState_checkFireWeaponStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hold     string
		magazine int
		want     i18n.Msgid
	}{
		{name: "何も持っていない", want: "No ranged weapon equipped"},
		{name: "松明は射撃に使えない", hold: "torch", want: "No ranged weapon equipped"},
		{name: "弾の無い弓は撃てない", hold: "hunter_bow", want: "Not loaded"},
		{name: "弾のある弓は撃てる", hold: "hunter_bow", magazine: 1, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			require.NoError(t, lifecycle.UnequipAll(world, player))
			if tt.hold != "" {
				item, err := lifecycle.SpawnBackpackItem(world, tt.hold, 1)
				require.NoError(t, err)
				lifecycle.MoveToEquip(world, item, player, gc.SlotHand1)
				if world.Components.Fire.Has(item) {
					world.Components.Fire.Get(item).Magazine = tt.magazine
				}
			}
			query.GetHandSelection(world).Slot = gc.SlotHand1

			want := ""
			if tt.want != "" {
				want = query.T(world, tt.want)
			}
			assert.Equal(t, want, (&ShootingState{}).checkFireWeaponStatus(world))
		})
	}
}
