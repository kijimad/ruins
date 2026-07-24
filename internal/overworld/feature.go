package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// Placement はリージョン方式の配置ルール。世界を Spacing チャンク四方のリージョンに分け、
// リージョンごとに1チャンクを決定的に当選させる。「1リージョンに高々1つ」と最小間隔
// Separation を O(1) で保証する。Minecraft の RandomSpreadStructurePlacement の翻案。
// 密な散布が要る地物が現れたら Scatter モードをここに拡張する。
type Placement struct {
	Spacing    consts.Chunk // リージョンの一辺。おおよそ Spacing チャンクに1つ当選する
	Separation consts.Chunk // 隣接リージョンの当選どうしが最低これだけ離れる。Spacing より小さいこと
	Salt       uint64       // 地物の種類ごとに相関を切る
}

// At は c がこの配置の当選チャンクかを返す。(runSeed, 座標, 帯の行数) の純関数で、
// 近傍のチャンクを生成せずに判定できる。
//
// X はリージョンで割り、オフセットを [0, Spacing-Separation) から引く。Y は帯が [0, rows) に
// 有界なのでリージョンで割らず、帯の全行から当選行を1つ引く。こうしないと行数より大きい
// オフセットを引いたリージョンの当選が帯の外へ落ち、行数の少ない帯に地物がほぼ出なくなる。
func (p Placement) At(runSeed uint64, c worldstream.ChunkCoord, rows consts.Chunk) bool {
	span := uint64(p.Spacing - p.Separation)
	rx := floorDiv(c.X, p.Spacing)
	h := ChunkSeed2D(runSeed^p.Salt, rx, 0)
	ox := consts.Chunk(h % span)
	oy := consts.Chunk((h / span) % uint64(rows))
	return c.X == rx*p.Spacing+ox && c.Y == oy
}

// floorDiv は負の座標でもリージョン割りが連続になる床除算。Go の / はゼロ方向へ丸めるため、
// 負側で境界が二重にならないよう床方向へ丸める。
func floorDiv(a, b consts.Chunk) consts.Chunk {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// chunkGeom は生成中チャンクの帯ローカル配置。地物が中身を置く座標計算に使う。
type chunkGeom struct {
	offsetX, offsetY consts.Tile
	chunkW, chunkH   consts.Tile
}

// feature は1種類の地物。c がその地物に該当するかを (runSeed, 座標, rows) の純関数で判定し、
// 該当すれば中身を配置する。種類の追加は実装を1つ足すことに還元し、分岐は増やさない。
type feature interface {
	place(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error
}

// features は登録済みの地物一覧。種類を増やすときはここへ実装を足す。
func features() []feature {
	return []feature{settlementFeature{}, urbanRuinFeature{}}
}

// PlaceFeatures は登録済みの地物を評価し、該当チャンクへ中身を配置する。
// 判定はすべて (runSeed, 座標, rows) の純関数で、start は開始チャンクの座標。
// 小集落は開始特例で必ず置かれ、市街地は開始チャンクを避ける。
func PlaceFeatures(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, offsetX, offsetY, chunkW, chunkH consts.Tile) error {
	g := chunkGeom{offsetX: offsetX, offsetY: offsetY, chunkW: chunkW, chunkH: chunkH}
	for _, f := range features() {
		if err := f.place(world, runSeed, c, start, rows, g); err != nil {
			return err
		}
	}
	return nil
}
