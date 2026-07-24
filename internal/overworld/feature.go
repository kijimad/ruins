package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// FeatureKind は地物の種類。種類の追加は spec のインスタンス追加に還元し、分岐は増やさない。
type FeatureKind int

const (
	// FeatureSettlement は小集落。交易・雇用・合成の補給地で、比較的安全
	FeatureSettlement FeatureKind = iota
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

// FeatureSpec は1種類の地物の「どこに・何を」をまとめた宣言。
// Content は当選チャンクの中心座標を受けて中身を配置する。
type FeatureSpec struct {
	Kind      FeatureKind
	Placement Placement
	Content   func(world w.World, center consts.Coord[consts.Tile]) error
}

// featureSpecs は登録済みの地物一覧。種類を増やすときはここへ spec を足す。
func featureSpecs() []FeatureSpec {
	return []FeatureSpec{settlementSpec}
}

// PlaceFeatures は登録済みの地物 spec を評価し、当選したチャンクへ中身を配置する。
// 判定は (runSeed, 座標) の純関数。start は開始チャンク特例で、新規ゲームの開始点に
// 必須サービスを保証するため、リージョン抽選に関わらず必ず小集落になる。
func PlaceFeatures(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, offsetX, offsetY, chunkW, chunkH consts.Tile) error {
	center := consts.Coord[consts.Tile]{X: offsetX + chunkW/2, Y: offsetY + chunkH/2}
	for _, spec := range featureSpecs() {
		hit := spec.Placement.At(runSeed, c, rows)
		if spec.Kind == FeatureSettlement && c == start {
			hit = true
		}
		if !hit {
			continue
		}
		if err := spec.Content(world, center); err != nil {
			return fmt.Errorf("地物配置失敗 (kind=%d, x=%d, y=%d): %w", spec.Kind, c.X, c.Y, err)
		}
	}
	return nil
}
