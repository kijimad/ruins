package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// Placement はリージョン方式の配置ルール。世界を X 方向 Spacing チャンク幅の縦帯リージョンに
// 分け、リージョンごとに、地物を置く1チャンクを (seed, リージョン) から決定的に選ぶ。この選ばれた
// 1チャンクを「当選チャンク」と呼ぶ。抽選のメタファーだが乱数でなく座標の純関数で、実行のたびに
// 同じチャンクが当たる。Y は帯が有界なので分割せず、リージョンは帯の全行を覆う。よってリージョンは
// Spacing(横) × rows(縦) チャンクの縦帯で、そこにちょうど1つ地物が出る。「1リージョンに高々1つ」と
// 最小間隔 Separation を O(1) で保証する。密な散布が要る地物が現れたら Scatter モードをここに拡張する。
type Placement struct {
	Spacing    consts.Chunk // リージョンの X 幅。おおよそ Spacing チャンクに1つ当選する
	Separation consts.Chunk // 隣接リージョンの当選どうしが最低これだけ離れる。Spacing より小さいこと
	Salt       uint64       // 地物の種類ごとに相関を切る
}

// At は c がこの配置の当選チャンクかを返す。(runSeed, 座標, 帯の行数) の純関数で、
// 近傍のチャンクを生成せずに判定できる。
//
// X はリージョンで割り、オフセットを [0, Spacing-Separation) から引く。Y は帯が [0, rows) に
// 有界なのでリージョンで割らず、帯の全行から当選行を1つ引く。こうしないと行数より大きい
// オフセットを引いたリージョンの当選が帯の外へ落ち、行数の少ない帯に地物がほぼ出なくなる。
func (p Placement) At(runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk) bool {
	return c == p.WinnerOf(runSeed, floorDiv(c.X, p.Spacing), rows)
}

// WinnerOf はリージョン rx の当選チャンク座標を返す。生成を伴わない純関数なので、
// 道の結線や情報サービスが近傍の地物位置を「生成せずに算出」する基盤になる。
func (p Placement) WinnerOf(runSeed uint64, rx, rows consts.Chunk) consts.Coord[consts.Chunk] {
	// Spacing <= Separation だと差が負になり uint64 キャストで巨大値へアンダーフローし、
	// オフセット抽選が壊れる。Placement は内部定数なので設定ミスは never として弾く
	if p.Spacing <= p.Separation {
		panic("Placement: Spacing は Separation より大きいこと")
	}
	span := uint64(p.Spacing - p.Separation)
	h := ChunkSeed2D(runSeed^p.Salt, rx, 0)
	return consts.Coord[consts.Chunk]{
		X: rx*p.Spacing + consts.Chunk(h%span),
		Y: consts.Chunk((h / span) % uint64(rows)),
	}
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

// chunkGeom は生成中チャンクの帯ローカル配置と、地物が共有するタイル索引。座標計算と
// タイル置換に使う。tiles は複数地物が同じ全域スキャンを繰り返さないための遅延共有索引。
type chunkGeom struct {
	offsetX, offsetY consts.Tile
	chunkW, chunkH   consts.Tile
	tiles            *tileIndex
}

// feature は1種類の地物。c がその地物に該当するかを (runSeed, 座標, rows) の純関数で判定し、
// 該当すれば中身を配置する。種類の追加は実装を1つ足すことに還元し、分岐は増やさない。
type feature interface {
	place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error
}

// features は登録済みの地物一覧。種類を増やすときはここへ実装を足す。
func features() []feature {
	// 点在POIは主役の地物へ譲る判定を持つため後に、道は他の地物の上を舗装しないよう最後に評価する
	return []feature{settlementFeature{}, urbanFeature{}, dungeonEntranceFeature{}, wildernessPOIFeature{}, roadFeature{}}
}

// 地物ごとのソルト。ハッシュ入力を地物ごとにずらし、配置と抽選を他地物と無相関にするタグ。
// 互いに異なりさえすればよく、ChunkSeed2D の finalizer が1ビット差でも出力を無相関へ散らす。
// iota で一意性を自動保証する。値を変えると同じ RunSeed でも別の世界になる。
const (
	settlementSalt uint64 = iota + 1
	urbanSalt
	dungeonEntranceSalt
	poiSalt
)

// PlaceFeatures は登録済みの地物を評価し、該当チャンクへ中身を配置する。
// 判定はすべて (runSeed, 座標, rows) の純関数で、開始チャンクの特例は持たない。
func PlaceFeatures(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, offsetX, offsetY, chunkW, chunkH consts.Tile) error {
	g := chunkGeom{
		offsetX: offsetX, offsetY: offsetY, chunkW: chunkW, chunkH: chunkH,
		tiles: &tileIndex{world: world, loX: offsetX, hiX: offsetX + chunkW},
	}
	for _, f := range features() {
		if err := f.place(world, runSeed, c, rows, g); err != nil {
			return err
		}
	}
	return nil
}
