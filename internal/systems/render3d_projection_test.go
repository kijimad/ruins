package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildScene_世界描画と重ねる側が同じ投影を使う(t *testing.T) {
	t.Parallel()

	// 世界レイヤの床と、その上へ重ねるカーソルが同じ場所に来ることを固定する。
	// 投影の組み立てが2箇所に分かれると、片方だけ直して片方が取り残される
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(960, 720)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, "ash")
	require.NoError(t, err)

	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	camera.Orient = 3
	camera.Pitch = 0.9

	// スプライトは原寸で重ねるので WorldProjectorSized(原寸) を使う。これがオーバーレイの
	// WorldProjector と一致しないと、カーソルや当たり判定が見た目とずれる
	fromWorld, err := render3d.WorldProjector(world)
	require.NoError(t, err)
	sized, err := render3d.WorldProjectorSized(world, 960, 720)
	require.NoError(t, err)
	assert.Equal(t, fromWorld, sized, "原寸のスプライト投影がオーバーレイと同じになる")
}
