package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDrawers_全DrawerKeyに実装がある は drawer キーの閉集合 DrawerKey enum と Go の drawers 実装が
// 一致することを固定する。tsp に drawer を足して drawers への登録を忘れると、ランドマークがその drawer を
// 指したときに解決できず落ちる。schema が値の閉集合を守り、この被覆テストが実装欠けを守る。
func TestDrawers_全DrawerKeyに実装がある(t *testing.T) {
	t.Parallel()

	for _, key := range []oapi.DrawerKey{oapi.Hut, oapi.Open} {
		_, ok := drawers[key]
		assert.Truef(t, ok, "drawer key %q に Go 実装がある", key)
	}
}

// findLandmarkChunk は点在ランドマークが当選し、他の地物に譲らないチャンクと seed を探す。
func findLandmarkChunk(t *testing.T) (uint64, consts.Coord[consts.Chunk]) {
	t.Helper()
	const rows = 1
	for s := uint64(1); s < 200; s++ {
		for x := range consts.Chunk(30) {
			c := consts.Coord[consts.Chunk]{X: x}
			if !landmarkPlacement.At(s, c, rows) || settlementPlacement.At(s, c, rows) || dungeonEntrancePlacement.At(s, c, rows) {
				continue
			}
			if _, _, _, ok := urbanRegionOf(s, c, rows); ok {
				continue
			}
			return s, c
		}
	}
	require.Fail(t, "前提: ランドマークだけが当選するチャンクが見つかる")
	return 0, consts.Coord[consts.Chunk]{}
}

// landmarkEntities は名前を持つエンティティを 座標 → 名前 の対応で集める。座標を文字列へ畳まず
// consts.Coord をそのまま map のキーにするので、桁や区切りの表記に依存せず整列も要らない。
func landmarkEntities(world w.World) map[consts.Coord[consts.Tile]]string {
	got := map[consts.Coord[consts.Tile]]string{}
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		got[world.Components.GridElement.Get(e).Coord] = world.Components.Name.Get(e).Name
	}
	return got
}

func TestWildernessLandmark_原野の当選チャンクに小構造物が決定的に出る(t *testing.T) {
	t.Parallel()

	seed, c := findLandmarkChunk(t)

	build := func() map[consts.Coord[consts.Tile]]string {
		world := testutil.InitTestWorld(t)
		g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: 50, chunkH: 50, tiles: &tileIndex{world: world, loX: 0, hiX: 50}}
		require.NoError(t, wildernessLandmarkFeature{}.place(world, seed, c, 1, g))
		return landmarkEntities(world)
	}
	a := build()
	b := build()
	assert.NotEmpty(t, a, "ランドマークの prop が置かれる")
	assert.Equal(t, a, b, "ランドマークの配置は決定的で再生成しても一致する")
}

func TestLandmark_各種別が異なる地図記号を持つ(t *testing.T) {
	t.Parallel()

	// 全 landmark id が mapGlyphs に記号を持ち、かつ互いに異なることを確認する。landmark id は
	// mapGlyphs の id と一致するので写像関数を介さず直接引く。記号の重複や欠落をここで弾く。
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	seen := map[rune]bool{}
	for _, l := range raw.PtrSlice(raws.Landmarks) {
		g, ok := glyphByID(raws, l.Id)
		require.Truef(t, ok, "landmark %q に地図記号がある", l.Id)
		assert.Falsef(t, seen[g.Label], "landmark %q の記号 %c が他と重複している", l.Id, g.Label)
		seen[g.Label] = true
	}
}
