package overworld

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
)

// 市街地は複数チャンクに及ぶ地表の危険地帯。補給の場ではなく、変質した群れが徘徊する
// 攻略対象で、v1 は舗装跡の床ブロックと敵で表す。
//
// レイアウトは citySeed から市街地全体を一括導出し、各チャンクは自分の断片だけを描く。
// 断片の導出は (runSeed, アンカー座標) の純関数なので、隣接チャンクを生成せずに断片どうしが
// 一致する。帯ストリーミングで市街地の一部だけが先に生成されても矛盾しない。
const (
	urbanSalt                   = 0x0b17
	urbanWidth     consts.Chunk = 2 // 市街地の横幅。チャンク数
	urbanBuildings              = 7 // 街区ブロック数の基準
)

// urbanPlacement は市街地アンカーのリージョン配置。小集落より疎に置く。
var urbanPlacement = Placement{Spacing: 16, Separation: 4, Salt: urbanSalt}

// urbanRuinFeature は市街地の feature 実装。
type urbanRuinFeature struct{}

// urbanAnchorOf は c を含む市街地のアンカーを返す。市街地はアンカーから東へ urbanWidth
// チャンク続く。該当しなければ ok=false。
func urbanAnchorOf(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk) (worldstream.ChunkCoord, bool) {
	for dx := consts.Chunk(0); dx < urbanWidth; dx++ {
		a := worldstream.ChunkCoord{X: c.X - dx, Y: c.Y}
		if urbanPlacement.At(runSeed, a, rows) {
			return a, true
		}
	}
	return worldstream.ChunkCoord{}, false
}

// cityRect は市街地ローカル座標での街区ブロック。
type cityRect struct {
	x, y, w, h consts.Tile
}

// cityLayout は citySeed から市街地全体の街区ブロックを一括導出する純関数。
// 断片描画はこの結果を自チャンク範囲へクリップするだけなので、どのチャンクから
// 生成しても同じ市街地になる。
func cityLayout(citySeed uint64, cityW, cityH consts.Tile) []cityRect {
	rng := rand.New(rand.NewPCG(citySeed, 0))
	rects := make([]cityRect, 0, urbanBuildings)
	for range urbanBuildings {
		rw := consts.Tile(5 + rng.IntN(6))
		rh := consts.Tile(4 + rng.IntN(4))
		x := consts.Tile(1 + rng.IntN(max(1, int(cityW-rw-2))))
		y := consts.Tile(1 + rng.IntN(max(1, int(cityH-rh-2))))
		rects = append(rects, cityRect{x: x, y: y, w: rw, h: rh})
	}
	return rects
}

// place は c が市街地の一部なら自分の断片を描く。開始チャンクを含む市街地は丸ごと
// スキップし、新規ゲームの開始点を安全に保つ。
func (urbanRuinFeature) place(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error {
	anchor, ok := urbanAnchorOf(runSeed, c, rows)
	if !ok {
		return nil
	}
	for dx := consts.Chunk(0); dx < urbanWidth; dx++ {
		if (worldstream.ChunkCoord{X: anchor.X + dx, Y: anchor.Y}) == start {
			return nil
		}
	}

	citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
	fragIdx := c.X - anchor.X
	fragOrigin := fragIdx.Tiles(g.chunkW) // 市街地ローカルでの自断片の西端 X

	// 断片範囲に重なる街区ブロックを床タイルへ置換する
	tiles := tileEntitiesInRange(world, g.offsetX, g.offsetX+g.chunkW)
	for _, r := range cityLayout(citySeed, urbanWidth.Tiles(g.chunkW), g.chunkH) {
		for ly := r.y; ly < r.y+r.h; ly++ {
			for lx := r.x; lx < r.x+r.w; lx++ {
				if lx < fragOrigin || lx >= fragOrigin+g.chunkW {
					continue
				}
				wx := g.offsetX + (lx - fragOrigin)
				wy := g.offsetY + ly
				if err := replaceTile(world, tiles, wx, wy, consts.TileNameFloor); err != nil {
					return fmt.Errorf("市街地の舗装配置に失敗 (x=%d, y=%d): %w", wx, wy, err)
				}
			}
		}
	}

	// 断片ごとの独立ストリームで敵を湧かせる。断片単位のシードなので生成順に依存しない
	fragRng := rand.New(rand.NewPCG(citySeed, uint64(fragIdx)+1))
	enemyCount := 2 + fragRng.IntN(3)
	for range enemyCount {
		pos := consts.Coord[consts.Tile]{
			X: g.offsetX + consts.Tile(fragRng.IntN(int(g.chunkW))),
			Y: g.offsetY + consts.Tile(fragRng.IntN(int(g.chunkH))),
		}
		if _, err := lifecycle.SpawnEnemy(world, pos, "火の玉"); err != nil {
			return fmt.Errorf("市街地の敵配置に失敗: %w", err)
		}
	}
	return nil
}

// tileEntitiesInRange は X 範囲 [loX, hiX) のタイルエンティティを座標引きできるよう集める。
func tileEntitiesInRange(world w.World, loX, hiX consts.Tile) map[gc.GridElement]ecs.Entity {
	tiles := make(map[gc.GridElement]ecs.Entity)
	q := query.ActiveFilter2[gc.GridElement, gc.Tile](world).Query()
	for q.Next() {
		e := q.Entity()
		g := *world.Components.GridElement.Get(e)
		if g.X >= loX && g.X < hiX {
			tiles[g] = e
		}
	}
	return tiles
}

// replaceTile は座標のタイルを取り除き、tileName のタイルへ置き換える。
func replaceTile(world w.World, tiles map[gc.GridElement]ecs.Entity, x, y consts.Tile, tileName string) error {
	g := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
	if e, ok := tiles[g]; ok && world.ECS.Alive(e) {
		world.ECS.RemoveEntity(e)
		delete(tiles, g)
	}
	zero := 0
	if _, err := lifecycle.SpawnTile(world, tileName, x, y, &zero); err != nil {
		return err
	}
	return nil
}
