package components

import (
	"math"

	"github.com/kijimaD/ruins/internal/consts"
)

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
// キャラクター（プレイヤー・敵）には付与しない。キャラクターの通行可否は関係性で判定する
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

// Pushable は押して動かせることを示すマーカー。移動拠点キューブが最初の利用者だが、印は汎用で
// キューブに限らない。BlockPass を持つ物でも、この印があると移動解決は通行不可でなく押しへ分岐する。
type Pushable struct{}

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

// direction8 は8方向の一覧。スナップの走査に使う。
var direction8 = [8]Direction{
	DirectionUp, DirectionUpRight, DirectionRight, DirectionDownRight,
	DirectionDown, DirectionDownLeft, DirectionLeft, DirectionUpLeft,
}

// ScreenIntent は方向を画面基準の意図ベクトルへ写す。su は上向き、sr は右向きの成分で
// どちらも -1/0/1。カメラ相対移動で、押されたキーが指す画面上の向きを表す。
func (d Direction) ScreenIntent() (su, sr float64) {
	switch d {
	case DirectionUp:
		return 1, 0
	case DirectionDown:
		return -1, 0
	case DirectionRight:
		return 0, 1
	case DirectionLeft:
		return 0, -1
	case DirectionUpRight:
		return 1, 1
	case DirectionUpLeft:
		return 1, -1
	case DirectionDownRight:
		return -1, 1
	case DirectionDownLeft:
		return -1, -1
	default:
		return 0, 0
	}
}

// RotateScreenDir は画面基準の方向を、カメラの水平角 yaw で world の8方向へ回す。
// 見下ろしカメラが回転しても、押したキーが画面上で指す向きへ動くようにする。
// 南から北を見下ろすカメラに合わせ、画面奥 forward=(-sin yaw, -cos yaw)、画面右 right=(cos yaw, -sin yaw) とし、
// world = su*forward + sr*right を最寄りの8方向へスナップする。
func RotateScreenDir(base Direction, yaw float64) Direction {
	su, sr := base.ScreenIntent()
	wx := -su*math.Sin(yaw) + sr*math.Cos(yaw)
	wy := -su*math.Cos(yaw) - sr*math.Sin(yaw)
	return SnapWorldVec(wx, wy)
}

// SnapWorldVec は world 平面のベクトルを最寄りの8方向へスナップする。
// カメラ相対移動で、回した向きの world ベクトルから実際の移動方向を決めるのに使う。
// ゼロベクトルは DirectionNone を返す。
func SnapWorldVec(wx, wy float64) Direction {
	best := DirectionNone
	bestDot := 0.0
	for _, d := range direction8 {
		delta := d.GetDelta()
		dx, dy := float64(delta.X), float64(delta.Y)
		norm := math.Hypot(dx, dy)
		if norm == 0 {
			continue
		}
		dot := (wx*dx + wy*dy) / norm
		if dot > bestDot {
			bestDot = dot
			best = d
		}
	}
	return best
}
