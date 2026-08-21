package overworld_test

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"sort"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// scatterCellPx は1タイルの描画辺長。スプライトが 32x32 なのでセルも 32px にして等倍で置く。
const scatterCellPx = 32

// スプライトのソース PNG の場所。テストはパッケージ直下で走るのでリポジトリ根へ遡る。
const (
	tileSpriteDir  = "../../assets/file/textures/tiles/"
	fieldSpriteDir = "../../assets/file/textures/single/"
)

// TestGolden_Scatter は wasteland チャンクの生成結果を生成の実経路 NewChunkGen で作り、タイルと prop を
// 直接 PNG へ描いて固定する。草・低木・岩の散布と幅を持たせた道が視界に入る。State を経由しない生成レベルの
// VRT なので、HUD やカメラでなくマップそのものの退行を捕らえる。散布は決定的な純関数なので RunSeed 66 で
// golden が安定する。散布・道の見た目や密度を変えると退行する。
func TestGolden_Scatter(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	const cols consts.Chunk = 3
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 66, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range cols {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{X: i}, i.Tiles(chunkW), 0))
	}

	img := renderOverworldRegion(t, world, 0, 0, int(cols)*int(chunkW), int(chunkH))
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	g := goldie.New(t, goldie.WithNameSuffix(".png"), goldie.WithEqualFn(testutil.PNGPixelEqual))
	g.Assert(t, t.Name(), buf.Bytes())
}

// renderOverworldRegion は帯ローカル矩形 [x0, x0+wTiles) × [y0, y0+hTiles) のタイルと prop を1枚の
// *image.RGBA へ描く。各エンティティの SpriteRender からスプライトの実 PNG を引き、Depth 昇順、床→prop、で
// 重ねる。ゲーム本体の描画と同じスプライトを使い、生成結果そのものを写す。
func renderOverworldRegion(t *testing.T, world w.World, x0, y0, wTiles, hTiles int) *image.RGBA {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, wTiles*scatterCellPx, hTiles*scatterCellPx))
	// 背景を暗色で塗る。タイルが全面を覆うが、未知スプライトで欠けても目立たないようにする
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 20, G: 18, B: 15, A: 255}), image.Point{}, draw.Src)

	type ent struct {
		g  gc.GridElement
		sr gc.SpriteRender
	}
	var ents []ent
	q := query.ActiveFilter2[gc.GridElement, gc.SpriteRender](world).Query()
	for q.Next() {
		e := q.Entity()
		ge := *world.Components.GridElement.Get(e)
		if int(ge.X) < x0 || int(ge.X) >= x0+wTiles || int(ge.Y) < y0 || int(ge.Y) >= y0+hTiles {
			continue
		}
		ents = append(ents, ent{g: ge, sr: *world.Components.SpriteRender.Get(e)})
	}
	// Depth 昇順で床を先に、prop を後に。同 Depth は座標で安定化して再現性を保つ
	sort.SliceStable(ents, func(i, j int) bool {
		if ents[i].sr.Depth != ents[j].sr.Depth {
			return ents[i].sr.Depth < ents[j].sr.Depth
		}
		if ents[i].g.Y != ents[j].g.Y {
			return ents[i].g.Y < ents[j].g.Y
		}
		return ents[i].g.X < ents[j].g.X
	})

	cache := make(map[string]image.Image)
	for i := range ents {
		spr := loadTestSprite(t, cache, ents[i].sr)
		if spr == nil {
			continue
		}
		dx := (int(ents[i].g.X) - x0) * scatterCellPx
		dy := (int(ents[i].g.Y) - y0) * scatterCellPx
		draw.Draw(img, image.Rect(dx, dy, dx+scatterCellPx, dy+scatterCellPx), spr, image.Point{}, draw.Over)
	}
	return img
}

// loadTestSprite は SpriteRender の指すスプライト PNG を読み、キャッシュする。タイルは tiles、field prop は
// single ディレクトリから引く。ファイルが無ければ nil を返し、描画をスキップする。
func loadTestSprite(t *testing.T, cache map[string]image.Image, sr gc.SpriteRender) image.Image {
	t.Helper()
	var dir string
	switch sr.SpriteSheetName {
	case "tile":
		dir = tileSpriteDir
	case "field":
		dir = fieldSpriteDir
	default:
		return nil
	}
	path := dir + sr.SpriteKey + "_.png"
	if im, ok := cache[path]; ok {
		return im
	}
	f, err := os.Open(path)
	if err != nil {
		cache[path] = nil // 未知スプライトは以後も引かない
		return nil
	}
	defer func() { _ = f.Close() }()
	im, err := png.Decode(f)
	require.NoError(t, err)
	cache[path] = im
	return im
}
