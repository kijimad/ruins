package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/resources"
)

// AutoTileIndex は16タイルオートタイルのインデックス（0-15）
// 4方向の隣接情報をビットマスクで表現
type AutoTileIndex int

// 16タイル標準パターン定数
// ビットマスク：上(1) 右(2) 下(4) 左(8)
// 各ビットは「その方向に同じタイルがある」ことを示す
const (
	AutoTileIsolated      AutoTileIndex = 0  // 0000: 全方向が異なる（孤立）
	AutoTileUp            AutoTileIndex = 1  // 0001: 上だけ同じ
	AutoTileRight         AutoTileIndex = 2  // 0010: 右だけ同じ
	AutoTileUpRight       AutoTileIndex = 3  // 0011: 上右が同じ
	AutoTileDown          AutoTileIndex = 4  // 0100: 下だけ同じ
	AutoTileVertical      AutoTileIndex = 5  // 0101: 上下が同じ
	AutoTileDownRight     AutoTileIndex = 6  // 0110: 下右が同じ
	AutoTileUpDownRight   AutoTileIndex = 7  // 0111: 上下右が同じ
	AutoTileLeft          AutoTileIndex = 8  // 1000: 左だけ同じ
	AutoTileUpLeft        AutoTileIndex = 9  // 1001: 上左が同じ
	AutoTileHorizontal    AutoTileIndex = 10 // 1010: 左右が同じ
	AutoTileUpLeftRight   AutoTileIndex = 11 // 1011: 上左右が同じ
	AutoTileDownLeft      AutoTileIndex = 12 // 1100: 下左が同じ
	AutoTileUpDownLeft    AutoTileIndex = 13 // 1101: 上下左が同じ
	AutoTileDownLeftRight AutoTileIndex = 14 // 1110: 下左右が同じ
	AutoTileCenter        AutoTileIndex = 15 // 1111: 全方向に同じタイル
)

// String はAutoTileIndexの文字列表現を返す
func (ati AutoTileIndex) String() string {
	switch ati {
	case AutoTileIsolated:
		return "Isolated"
	case AutoTileUp:
		return "Up"
	case AutoTileRight:
		return "Right"
	case AutoTileUpRight:
		return "UpRight"
	case AutoTileDown:
		return "Down"
	case AutoTileVertical:
		return "Vertical"
	case AutoTileDownRight:
		return "DownRight"
	case AutoTileUpDownRight:
		return "UpDownRight"
	case AutoTileLeft:
		return "Left"
	case AutoTileUpLeft:
		return "UpLeft"
	case AutoTileHorizontal:
		return "Horizontal"
	case AutoTileUpLeftRight:
		return "UpLeftRight"
	case AutoTileDownLeft:
		return "DownLeft"
	case AutoTileUpDownLeft:
		return "UpDownLeft"
	case AutoTileDownLeftRight:
		return "DownLeftRight"
	case AutoTileCenter:
		return "Center"
	default:
		return fmt.Sprintf("Unknown(%d)", int(ati))
	}
}

// CalculateAutoTileIndex は4方向の隣接情報からオートタイルインデックスを計算
// 同じタイル名のタイルとのみ接続する
func (mp *MetaPlan) CalculateAutoTileIndex(idx resources.TileIdx, tileType string) AutoTileIndex {
	// 4方向の隣接チェック - 同じタイル名の場合のみ接続
	upTile := mp.UpTile(idx)
	downTile := mp.DownTile(idx)
	leftTile := mp.LeftTile(idx)
	rightTile := mp.RightTile(idx)

	up := upTile.Name == tileType
	down := downTile.Name == tileType
	left := leftTile.Name == tileType
	right := rightTile.Name == tileType

	// ビットマスク計算（標準16タイルパターン）
	bitmask := 0
	if up {
		bitmask |= 1
	} // bit 0: 上
	if right {
		bitmask |= 2
	} // bit 1: 右
	if down {
		bitmask |= 4
	} // bit 2: 下
	if left {
		bitmask |= 8
	} // bit 3: 左

	return AutoTileIndex(bitmask)
}

// IsValidIndex はインデックスが有効範囲内かチェック
func (mp *MetaPlan) IsValidIndex(idx resources.TileIdx) bool {
	return idx >= 0 && int(idx) < len(mp.Tiles)
}
