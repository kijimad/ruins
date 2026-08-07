package mapplanner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
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
		panic(fmt.Sprintf("不正なAutoTileIndex値: %d", int(ati)))
	}
}

// AutoTileBits は4方向の接続の有無からオートタイルのビットマスクを組む。上1・右2・下4・左8。
// プランナ格子を見る CalculateAutoTileIndex と、生成後の ECS 実体を見る overworld の継ぎ目再計算が
// この規則を1箇所から共有し、両者のビット割り当てがずれないようにする。純関数なので単体で検証できる。
func AutoTileBits(up, right, down, left bool) AutoTileIndex {
	bitmask := 0
	if up {
		bitmask |= 1
	}
	if right {
		bitmask |= 2
	}
	if down {
		bitmask |= 4
	}
	if left {
		bitmask |= 8
	}
	return AutoTileIndex(bitmask)
}

// CalculateAutoTileIndex は4方向の隣接情報からオートタイルインデックスを計算
// 同じタイル名のタイルとのみ接続する
func (mp *MetaPlan) CalculateAutoTileIndex(idx gc.TileIdx, tileType string) AutoTileIndex {
	// 4方向の隣接チェック - 同じタイル名の場合のみ接続
	up := mp.UpTile(idx).Name == tileType
	right := mp.RightTile(idx).Name == tileType
	down := mp.DownTile(idx).Name == tileType
	left := mp.LeftTile(idx).Name == tileType
	return AutoTileBits(up, right, down, left)
}

// IsValidIndex はインデックスが有効範囲内かチェック
func (mp *MetaPlan) IsValidIndex(idx gc.TileIdx) bool {
	return idx >= 0 && int(idx) < len(mp.Tiles)
}
