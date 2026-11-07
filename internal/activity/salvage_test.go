package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolDamageMultiplier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		weapon string
		prop   string
		want   int
	}{
		{"斧で木を殴るとグレード2で3倍", "axe", "tree_a", 3},
		{"ペンチで木箱を殴るとグレード1で2倍", "monkey_wrench", "wooden_crate", 2},
		{"斧で木箱を殴っても分類が違うので1倍", "axe", "wooden_crate", 1},
		{"道具が必須の木を松明で殴ると0倍", "torch", "tree_a", 0},
		{"道具が必須でない木箱を松明で殴ると1倍", "torch", "wooden_crate", 1},
		{"分解定義の無いpropは1倍", "axe", "grass", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			require.NoError(t, lifecycle.UnequipAll(world, player))
			weapon, err := lifecycle.SpawnBackpackItem(world, tt.weapon, 1)
			require.NoError(t, err)
			lifecycle.MoveToEquip(world, weapon, player, gc.SlotWeapon1)
			query.GetWeaponSelection(world).Slot = 1
			target, err := lifecycle.SpawnProp(world, tt.prop, 10, 9)
			require.NoError(t, err)

			assert.Equal(t, tt.want, toolDamageMultiplier(world, player, target))
		})
	}
}

func TestDropSalvage(t *testing.T) {
	t.Parallel()

	// dropsAt は prop を壊したタイルに落ちたアイテムの id を返す
	dropsAt := func(t *testing.T, weapon, prop string) []string {
		t.Helper()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		require.NoError(t, lifecycle.UnequipAll(world, player))
		w, err := lifecycle.SpawnBackpackItem(world, weapon, 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, w, player, gc.SlotWeapon1)
		query.GetWeaponSelection(world).Slot = 1
		target, err := lifecycle.SpawnProp(world, prop, 10, 9)
		require.NoError(t, err)

		require.NoError(t, dropSalvage(world, player, target))

		pickables := query.PickablesAt(world, consts.Coord[consts.Tile]{X: 10, Y: 9})
		ids := make([]string, 0, len(pickables))
		for _, e := range pickables {
			ids = append(ids, query.GetEntityID(e, world))
		}
		return ids
	}

	t.Run("斧で倒した木は薪を落とす", func(t *testing.T) {
		t.Parallel()
		ids := dropsAt(t, "axe", "tree_a")
		require.NotEmpty(t, ids)
		for _, id := range ids {
			assert.Equal(t, "firewood", id)
		}
	})

	t.Run("道具の分類が無い灌木は何で倒しても枝を落とす", func(t *testing.T) {
		t.Parallel()
		ids := dropsAt(t, "torch", "bush")
		require.NotEmpty(t, ids)
		for _, id := range ids {
			assert.Equal(t, "branch", id)
		}
	})

	t.Run("道具なしで壊した木箱は確率枠を出さず確定枠だけを回収しうる", func(t *testing.T) {
		t.Parallel()
		for _, id := range dropsAt(t, "torch", "wooden_crate") {
			assert.Equal(t, "hardwood", id)
		}
	})
}
