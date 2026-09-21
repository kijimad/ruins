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

func TestDeployed_展開状態を保存しロードで復元する(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, lifecycle.DeployCube(world, cube))
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	fresh := testutil.InitTestWorld(t)
	require.NoError(t, sm.LoadWorld(fresh, "slot1"))

	deployed := 0
	q := ecs.NewFilter1[gc.Deployed](fresh.ECS).Query()
	for q.Next() {
		deployed++
	}
	assert.Equal(t, 1, deployed, "展開状態がロードで復元される")
}

func TestLocationStowed_貨物のOwnerがロードで同じキューブを指す(t *testing.T) {
	t.Parallel()
	sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	// 圧縮中の貨物を1つ畳み込む。エンティティ参照 Owner がロードで再マップされるかを見る
	require.NoError(t, lifecycle.StowDefaultCubeCargo(world, cube))
	require.NoError(t, sm.SaveWorld(world, "slot1"))

	fresh := testutil.InitTestWorld(t)
	require.NoError(t, sm.LoadWorld(fresh, "slot1"))

	var loadedCube ecs.Entity
	cubeCount := 0
	cq := ecs.NewFilter1[gc.Drivable](fresh.ECS).Query()
	for cq.Next() {
		loadedCube = cq.Entity()
		cubeCount++
	}
	require.Equal(t, 1, cubeCount, "キューブがロードで復元される")

	var cargo ecs.Entity
	cargoCount := 0
	sq := ecs.NewFilter1[gc.LocationStowed](fresh.ECS).Query()
	for sq.Next() {
		cargo = sq.Entity()
		cargoCount++
	}
	require.Equal(t, 1, cargoCount, "畳み込んだ貨物がロードで復元される")
	owner := fresh.Components.LocationStowed.Get(cargo).Owner
	require.True(t, fresh.ECS.Alive(owner), "Owner が生存エンティティを指す")
	assert.Equal(t, loadedCube, owner, "Owner はロード後のキューブを指す")
}
