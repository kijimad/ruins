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
	urbanMaxWidth  consts.Chunk = 3 // 市街地の最大横幅。チャンク数
	urbanBuildings              = 7 // 街区ブロック数の基準
)

// urbanWidthOf は市街地の横幅チャンク数を citySeed から決定的に選ぶ。2..urbanMaxWidth。
// 規模が大きいほど敵も多く、リスクとリターンが規模に比例する。
func urbanWidthOf(citySeed uint64) consts.Chunk {
	return 2 + consts.Chunk((citySeed>>8)%uint64(urbanMaxWidth-1))
}

// urbanPlacement は市街地アンカーのリージョン配置。小集落より疎に置く。
var urbanPlacement = Placement{Spacing: 16, Separation: 4, Salt: urbanSalt}

// urbanRuinFeature は市街地の feature 実装。
type urbanRuinFeature struct{}

// urbanAnchorOf は c を含む市街地のアンカーと横幅を返す。市街地はアンカーから東へ
// width チャンク続く。該当しなければ ok=false。Spacing が最大幅より大きいので、
// 走査範囲に当選アンカーは高々1つしか無い。
func urbanAnchorOf(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk) (anchor worldstream.ChunkCoord, width consts.Chunk, ok bool) {
	for dx := range urbanMaxWidth {
		a := worldstream.ChunkCoord{X: c.X - dx, Y: c.Y}
		if !urbanPlacement.At(runSeed, a, rows) {
			continue
		}
		w := urbanWidthOf(ChunkSeed2D(runSeed^urbanSalt, a.X, a.Y))
		if dx < w {
			return a, w, true
		}
		return worldstream.ChunkCoord{}, 0, false
	}
	return worldstream.ChunkCoord{}, 0, false
}

// cityTile は市街地レイアウト上の1マスの種別。
type cityTile uint8

const (
	cityFloor cityTile = iota + 1 // 舗装跡・屋内の床
	cityWall                      // 廃屋の壁
)

// cityTiles は citySeed から市街地全体の壁・床レイアウトを一括導出する純関数。
// 街区ブロックごとに外周を壁、内側を床にし、南辺に1マスの出入口を開ける。
// 断片描画はこの結果を自チャンク範囲へクリップするだけなので、どのチャンクから
// 生成しても同じ市街地になる。街区数は幅に比例させ、規模が大きいほど密になる。
func cityTiles(citySeed uint64, width consts.Chunk, cityW, cityH consts.Tile) map[consts.Coord[consts.Tile]]cityTile {
	rng := rand.New(rand.NewPCG(citySeed, 0))
	m := map[consts.Coord[consts.Tile]]cityTile{}
	buildings := urbanBuildings * int(width) / 2
	for range buildings {
		rw := consts.Tile(5 + rng.IntN(6))
		rh := consts.Tile(4 + rng.IntN(4))
		x := consts.Tile(1 + rng.IntN(max(1, int(cityW-rw-2))))
		y := consts.Tile(1 + rng.IntN(max(1, int(cityH-rh-2))))
		door := x + 1 + consts.Tile(rng.IntN(max(1, int(rw-2))))
		for ly := y; ly < y+rh; ly++ {
			for lx := x; lx < x+rw; lx++ {
				p := consts.Coord[consts.Tile]{X: lx, Y: ly}
				perimeter := lx == x || lx == x+rw-1 || ly == y || ly == y+rh-1
				switch {
				case perimeter && ly == y+rh-1 && lx == door:
					m[p] = cityFloor // 出入口
				case perimeter:
					if m[p] != cityFloor {
						m[p] = cityWall
					}
				default:
					m[p] = cityFloor
				}
			}
		}
	}
	return m
}

// place は c が市街地の一部なら自分の断片を描く。開始チャンクを含む市街地は丸ごと
// スキップし、新規ゲームの開始点を安全に保つ。
func (urbanRuinFeature) place(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error {
	anchor, width, ok := urbanAnchorOf(runSeed, c, rows)
	if !ok {
		return nil
	}
	for dx := range width {
		if (worldstream.ChunkCoord{X: anchor.X + dx, Y: anchor.Y}) == start {
			return nil
		}
	}

	citySeed := ChunkSeed2D(runSeed^urbanSalt, anchor.X, anchor.Y)
	fragIdx := c.X - anchor.X
	fragOrigin := fragIdx.Tiles(g.chunkW) // 市街地ローカルでの自断片の西端 X

	// 断片範囲に重なる壁・床を置換する。オートタイル添字は置換後にチャンク全域の
	// 再計算が実状態から揃えるため、ここでは仮の 0 で置いてよい
	tiles := tileEntitiesInRange(world, g.offsetX, g.offsetX+g.chunkW)
	layout := cityTiles(citySeed, width, width.Tiles(g.chunkW), g.chunkH)
	for p, kind := range layout {
		if p.X < fragOrigin || p.X >= fragOrigin+g.chunkW {
			continue
		}
		wx := g.offsetX + (p.X - fragOrigin)
		wy := g.offsetY + p.Y
		name := consts.TileNameFloor
		if kind == cityWall {
			name = consts.TileNameDWall
		}
		if err := replaceTile(world, tiles, wx, wy, name); err != nil {
			return fmt.Errorf("市街地の配置に失敗 (x=%d, y=%d): %w", wx, wy, err)
		}
	}

	// 断片ごとの独立ストリームで敵を湧かせる。断片単位のシードなので生成順に依存しない。
	// 敵の数は規模(横幅)に比例し、大きな市街地ほど高リスク高リターンになる
	fragRng := rand.New(rand.NewPCG(citySeed, uint64(fragIdx)+1))
	enemyCount := int(width-1) * (2 + fragRng.IntN(3))
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
// オートタイル添字は仮の 0 で置き、後段の RecalcAutotileInXRange が実状態から揃える。
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
