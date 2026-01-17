package components

// Position はフィールド上に座標をもって存在する
// スプライトはこの位置に中心を合わせて配置する
// -----
// |   |
// | * |
// |   |
// -----
type Position struct {
	X Pixel
	Y Pixel
}

// Pixel はピクセル単位。計算用にfloat64
type Pixel float64

// GridElement はフィールド上にグリッドに沿って存在する
// スプライトはグリッドに沿って配置する
// *----
// |   |
// |   |
// |   |
// -----
type GridElement struct {
	X Tile
	Y Tile
}

// Tile はタイルの位置。ピクセル数ではない
type Tile int

// Rect は矩形を表す構造体
type Rect struct {
	X1 Tile
	X2 Tile
	Y1 Tile
	Y2 Tile
}

// Center は矩形の中心座標を返す
func (r *Rect) Center() (Tile, Tile) {
	x := (r.X1 + r.X2) / 2
	y := (r.Y1 + r.Y2) / 2
	return x, y
}

// BlockPass はフィールド上で通過できない
type BlockPass struct{}

// BlockView はフィールド上で視界を遮る
type BlockView struct{}

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

// GetDelta は方向から移動量を取得する
func (d Direction) GetDelta() (int, int) {
	switch d {
	case DirectionUp:
		return 0, -1
	case DirectionDown:
		return 0, 1
	case DirectionLeft:
		return -1, 0
	case DirectionRight:
		return 1, 0
	case DirectionUpLeft:
		return -1, -1
	case DirectionUpRight:
		return 1, -1
	case DirectionDownLeft:
		return -1, 1
	case DirectionDownRight:
		return 1, 1
	default:
		return 0, 0
	}
}
