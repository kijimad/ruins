package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

// Placement はリージョン方式の配置ルール。世界を Y 方向 Spacing チャンク高の横帯リージョンに
// 分け、リージョンごとに、地物を置く1チャンクを (seed, リージョン) から決定的に選ぶ。この選ばれた
// 1チャンクを「当選チャンク」と呼ぶ。抽選のメタファーだが乱数でなく座標の純関数で、実行のたびに
// 同じチャンクが当たる。X は帯が有界なので分割せず、リージョンは帯の全列を覆う。よってリージョンは
// 横 cols × 縦 Spacing チャンクの横帯で、そこにちょうど1つ地物が出る。「1リージョンに高々1つ」と
// 最小間隔 Separation を O(1) で保証する。密な散布が要る地物が現れたら Scatter モードをここに拡張する。
type Placement struct {
	Spacing    consts.Chunk // リージョンの Y 高。おおよそ Spacing チャンクに1つ当選する
	Separation consts.Chunk // 隣接リージョンの当選どうしが最低これだけ離れる。Spacing より小さいこと
	Salt       uint64       // 地物の種類ごとに相関を切る
}

// At は c がこの配置の当選チャンクかを返す。runSeed・座標・列数の純関数で、
// 近傍のチャンクを生成せずに判定できる。
//
// Y はリージョンで割り、オフセットを [0, Spacing-Separation) から引く。X は帯が [0, cols) に
// 有界なのでリージョンで割らず、帯の全列から当選列を1つ引く。こうしないと列数より大きい
// オフセットを引いたリージョンの当選が帯の外へ落ち、列数の少ない帯に地物がほぼ出なくなる。
func (p Placement) At(runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) bool {
	return c == p.WinnerOf(runSeed, consts.FloorDiv(c.Y, p.Spacing), cols)
}

// WinnerOf はリージョン ry の当選チャンク座標を返す。生成を伴わない純関数なので、
// 道の結線や情報サービスが近傍の地物位置を「生成せずに算出」する基盤になる。
func (p Placement) WinnerOf(runSeed uint64, ry, cols consts.Chunk) consts.Coord[consts.Chunk] {
	// Spacing <= Separation だと差が負になり uint64 キャストで巨大値へアンダーフローし、
	// オフセット抽選が壊れる。raw のスキーマが separation < spacing を保証するので never として弾く
	if p.Spacing <= p.Separation {
		panic("Placement: Spacing must be greater than Separation")
	}
	span := uint64(p.Spacing - p.Separation)
	h := ChunkSeed2D(runSeed^p.Salt, ry, 0)
	return consts.Coord[consts.Chunk]{
		X: consts.Chunk((h / span) % uint64(cols)),
		Y: ry*p.Spacing + consts.Chunk(h%span),
	}
}

// chunkGeom は生成中チャンクの帯ローカル配置と、地物が共有するタイル索引。座標計算と
// タイル置換に使う。tiles は複数地物が同じ全域スキャンを繰り返さないための遅延共有索引。
type chunkGeom struct {
	offsetX, offsetY consts.Tile
	chunkW, chunkH   consts.Tile
	tiles            *tileIndex
}

// feature は1種類の地物。c がその地物に該当するかを runSeed・座標・cols の純関数で判定し、
// 該当すれば中身を配置する。種類の追加は実装を1つ足すことに還元し、分岐は増やさない。
type feature interface {
	place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error
}

// features は登録済みの地物一覧。種類を増やすときはここへ実装を足す。施設の絵は下地を残す地形のマスに
// 先に置かれたタイルを残すので、施設を置く市街地より前の地物は土の地面以外を置かない
func features() []feature {
	// 点在ランドマークは主役の地物へ譲る判定を持つため後に、道は他の地物の上を舗装しないよう後に
	// 評価する。散布は占有と道の実状態を読むため最後に評価する
	return []feature{
		settlementFeature{}, urbanFeature{}, dungeonEntranceFeature{}, wildernessLandmarkFeature{},
		roadFeature{}, openTerrainFeature{},
	}
}

// 区域と散布のソルト。ハッシュ入力を地物ごとにずらし、配置と抽選を他地物と無相関にするタグ。
// 互いに異なりさえすればよく、ChunkSeed2D の finalizer が1ビット差でも出力を無相関へ散らす。
// 値を変えると同じ RunSeed でも別の世界になる。集落・遺跡の入口・ランドマークのソルトは raw の地物の行が持ち、
// ここの値と重ならないことをテストが確かめる
const (
	districtSalt    uint64 = 2
	scatterSalt     uint64 = 5
	outdoorLootSalt uint64 = 6
)

// pointFeatureOf は地物の種類 kind の raw の行を返す。全ての種類に行があることを raw の検証が保証する
func pointFeatureOf(raws rawschema.Raws, kind rawschema.PointFeatureKind) rawschema.PointFeature {
	for _, f := range raw.PtrSlice(raws.PointFeatures) {
		if f.Id == kind {
			return f
		}
	}
	panic("point feature has no raw row: " + string(kind))
}

// placementOf は地物の種類 kind の配置ルールを、raw の行の間隔とソルトから作る
func placementOf(raws rawschema.Raws, kind rawschema.PointFeatureKind) Placement {
	return placementFrom(pointFeatureOf(raws, kind))
}

// placementFrom は地物の行 f の配置ルールを作る
func placementFrom(f rawschema.PointFeature) Placement {
	return Placement{Spacing: consts.Chunk(f.Spacing), Separation: consts.Chunk(f.Separation), Salt: uint64(f.Salt)}
}

// PlaceFeatures は登録済みの地物を評価し、該当チャンクへ中身を配置する。
// 判定はすべて runSeed・座標・cols の純関数で、開始チャンクの特例は持たない。
func PlaceFeatures(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, offsetX, offsetY, chunkW, chunkH consts.Tile) error {
	g := chunkGeom{
		offsetX: offsetX, offsetY: offsetY, chunkW: chunkW, chunkH: chunkH,
		tiles: &tileIndex{world: world, loX: offsetX, hiX: offsetX + chunkW},
	}
	for _, f := range features() {
		if err := f.place(world, runSeed, c, cols, g); err != nil {
			return err
		}
	}
	return nil
}
