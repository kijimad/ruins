package save

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRaider_襲撃の敵の印を保存しロードで復元する(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	raider, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "stray_dog")
	require.NoError(t, err)
	world.Components.Raider.Add(raider, &gc.Raider{CubeOnly: true})
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "stray_dog")
	require.NoError(t, err)
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	fresh := testutil.InitTestWorld(t)
	require.NoError(t, sm.LoadWorld(fresh, "slot1"))

	var raiders []consts.Coord[consts.Tile]
	var marks []gc.Raider
	q := ecs.NewFilter2[gc.Raider, gc.GridElement](fresh.ECS).Query()
	for q.Next() {
		r, g := q.Get()
		raiders = append(raiders, g.Coord)
		marks = append(marks, *r)
	}
	assert.Equal(t, []consts.Coord[consts.Tile]{{X: 10, Y: 10}}, raiders, "印のある敵だけが印を持ったまま復元される")
	assert.Equal(t, []gc.Raider{{CubeOnly: true}}, marks, "キューブだけを狙うかも復元される")
}
