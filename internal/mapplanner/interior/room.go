package interior

import "github.com/kijimaD/ruins/internal/consts"

// Vec は建物ローカルのタイル座標。座標軸は footprint の入る位置を原点とする局所系だが、単位は overworld の
// 全域タイル consts.Tile と同じにして、境界での余計な変換を無くす。overworld も chunk 局所座標に consts.Tile を
// 使い offsetX の足し算でフレームを移すので、interior もその流儀に合わせる。
type Vec = consts.Coord[consts.Tile]

// Rect はタイル単位の矩形。X,Y が左上、W,H が幅と高さ。
type Rect struct{ X, Y, W, H consts.Tile }

// Room は分割文法の出力の契約。content システムはこの契約のみに依存し、部屋の形の作り方は知らない。
// layout↔content の契約。ゾーン印は後続 Stage で足す。
type Room struct {
	Rect     Rect
	Doorways []Doorway
}

// Doorway は部屋の戸口タイル。placement はここを塞がない。座標としては Vec と同じ。
type Doorway = Vec

// center は矩形の中心タイル。奇数辺なら真ん中、偶数辺なら中心寄り。
func (r Rect) center() Vec {
	return Vec{X: r.X + r.W/2, Y: r.Y + r.H/2}
}

// interiorTiles は外周の壁を除いた内側の床タイルを、y→x の固定順で返す。map を使わず順序を決定的にする。
func (r Rect) interiorTiles() []Vec {
	out := make([]Vec, 0, (r.W-2)*(r.H-2))
	for y := r.Y + 1; y < r.Y+r.H-1; y++ {
		for x := r.X + 1; x < r.X+r.W-1; x++ {
			out = append(out, Vec{X: x, Y: y})
		}
	}
	return out
}
