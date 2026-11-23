package overworld

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// TestGolden_Scatter は wasteland チャンクの生成結果を生成の実経路 NewChunkGen で作り、タイルと prop を
// 直接 PNG へ描いて固定する。草・低木・岩の散布と幅を持たせた道が視界に入る。State を経由しない生成レベルの
// VRT なので、HUD やカメラでなくマップそのものの退行を捕らえる。散布は決定的な純関数なので RunSeed 67 で
// golden が安定する。散布・道の見た目や密度を変えると退行する。
func TestGolden_Scatter(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	const cols consts.Chunk = 3
	world := testutil.InitTestWorld(t)
	gen := NewChunkGen(world, 67, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range cols {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{X: i}, i.Tiles(chunkW), 0))
	}

	img, err := RenderRegion(world, image.Rect(0, 0, int(cols)*int(chunkW), int(chunkH)), RegionSpritePx)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), buf.Bytes())
}
