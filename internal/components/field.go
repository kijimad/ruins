package components

import "github.com/kijimaD/ruins/internal/consts"

// Position はフィールド上に座標をもって存在する
// 値はカメラ変換前のワールド座標。描画時にはスクリーン座標へ変換する必要がある
// スプライトはこの位置に中心を合わせて配置する
// -----
// |   |
// | * |
// |   |
// -----
type Position struct {
	consts.Coord[consts.WorldPixel]
}

// GridElement はフィールド上にグリッドに沿って存在する
// スプライトはグリッドに沿って配置する
// *----
// |   |
// |   |
// |   |
// -----
type GridElement struct {
	consts.Coord[consts.Tile]
}

// Rect は矩形を表す構造体。Min が左上、Max が右下の隅
type Rect struct {
	Min consts.Coord[consts.Tile]
	Max consts.Coord[consts.Tile]
}

// Center は矩形の中心座標を返す
func (r *Rect) Center() (consts.Tile, consts.Tile) {
	x := (r.Min.X + r.Max.X) / 2
	y := (r.Min.Y + r.Max.Y) / 2
	return x, y
}

// Width は矩形の幅を返す。Max と Min の X 差。
func (r *Rect) Width() consts.Tile { return r.Max.X - r.Min.X }

// Height は矩形の高さを返す。Max と Min の Y 差。
func (r *Rect) Height() consts.Tile { return r.Max.Y - r.Min.Y }

// Tile はタイルエンティティであることを示すタグコンポーネント
type Tile struct{}

// BlockPass は壁やドアなどの静的障害物に付与する。このコンポーネントを持つタイルは常に通行不可になる。
// キャラクター（プレイヤー・敵・隊員）には付与しない。キャラクターの通行可否は関係性で判定する
type BlockPass struct{}

// BlockView はフィールド上で視界を遮る
// TODO: 能動態のほうがわかりやすそう
type BlockView struct{}

// PassCost はフィールド上のタイルの移動コストを修正する。
// ベースコストへの加算値で表現する
type PassCost struct {
	Value int // 移動コスト加算値。0で変化なし、50でコスト+50
}

// Renderable はフィールド上で描画できる
type Renderable struct{}

// Direction はタイルベース移動の方向
type Direction int

const (
	// DirectionNone は移動なし（待機）
	DirectionNone Direction = iota
	// DirectionUp は上方向
	DirectionUp
	// DirectionDown は下方向
	DirectionDown
	// DirectionLeft は左方向
	DirectionLeft
	// DirectionRight は右方向
	DirectionRight
	// DirectionUpLeft は左上方向
	DirectionUpLeft
	// DirectionUpRight は右上方向
	DirectionUpRight
	// DirectionDownLeft は左下方向
	DirectionDownLeft
	// DirectionDownRight は右下方向
	DirectionDownRight
)

// GetDelta は方向から移動量をタイル座標の差分として取得する。各成分は -1/0/1
func (d Direction) GetDelta() consts.Coord[consts.Tile] {
	switch d {
	case DirectionUp:
		return consts.Coord[consts.Tile]{X: 0, Y: -1}
	case DirectionDown:
		return consts.Coord[consts.Tile]{X: 0, Y: 1}
	case DirectionLeft:
		return consts.Coord[consts.Tile]{X: -1, Y: 0}
	case DirectionRight:
		return consts.Coord[consts.Tile]{X: 1, Y: 0}
	case DirectionUpLeft:
		return consts.Coord[consts.Tile]{X: -1, Y: -1}
	case DirectionUpRight:
		return consts.Coord[consts.Tile]{X: 1, Y: -1}
	case DirectionDownLeft:
		return consts.Coord[consts.Tile]{X: -1, Y: 1}
	case DirectionDownRight:
		return consts.Coord[consts.Tile]{X: 1, Y: 1}
	default:
		return consts.Coord[consts.Tile]{X: 0, Y: 0}
	}
}
