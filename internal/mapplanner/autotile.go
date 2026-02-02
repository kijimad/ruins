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

// CalculateWallAutoTileIndex は壁用のオートタイルインデックスを計算する
// 4方向の隣接タイルが壁かチェックしてビットマスクを生成
// 13タイルの斜め上視点専用タイルセットを使用
func (mp *MetaPlan) CalculateWallAutoTileIndex(idx resources.TileIdx) AutoTileIndex {
	// 4方向の隣接チェック - 壁タイルの場合false（壁と接続していない）
	upTile := mp.UpTile(idx)
	downTile := mp.DownTile(idx)
	leftTile := mp.LeftTile(idx)
	rightTile := mp.RightTile(idx)

	// 壁と接続しているかを判定（壁タイルの場合false）
	up := !mp.isWall(upTile)
	down := !mp.isWall(downTile)
	left := !mp.isWall(leftTile)
	right := !mp.isWall(rightTile)

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

	// 16パターンから13タイルへのマッピング
	// 斜め上視点では下に床がある場合は横縞（壁面）が見える
	mapping := [16]AutoTileIndex{
		0:  9,  // 0000: 全方向壁 → 右下コーナー（上下左右と接続）
		1:  1,  // 0001: 上のみ床 → 上辺（下左右と接続）
		2:  5,  // 0010: 右のみ床 → 左辺（右に部屋がある壁の左側面）
		3:  2,  // 0011: 上右に床 → 右上コーナー（左下に接続）
		4:  1,  // 0100: 下のみ床 → 上辺（左右上と接続）
		5:  1,  // 0101: 上下に床 → 上辺（左右と接続、横棒）
		6:  11, // 0110: 右下に床 → 中央（左上に接続）
		7:  11, // 0111: 右上下に床 → 中央（左とだけ接続）
		8:  5,  // 1000: 左のみ床 → 左辺（上下右と接続、縦棒）
		9:  0,  // 1001: 左上に床 → 左上コーナー
		10: 5,  // 1010: 左右に床 → 左辺（縦棒の壁）
		11: 5,  // 1011: 左上右に床 → 左辺（縦棒の壁、下とだけ接続）
		12: 10, // 1100: 左下に床 → 下辺太い
		13: 10, // 1101: 左上下に床 → 下辺太い（右とだけ接続）
		14: 6,  // 1110: 左右下に床 → 横縞+左右枠
		15: 6,  // 1111: 全方向床 → 下優先
	}

	result := mapping[bitmask]

	// デバッグ出力：左上が壁、右下が壁ではないパターン
	if !up && !left && right && down {
		fmt.Printf("DEBUG 左上接続: idx=%d, bitmask=%d, tile=%d, up=%t(%s), right=%t(%s), down=%t(%s), left=%t(%s)\n",
			idx, bitmask, result, up, upTile.Name, right, rightTile.Name, down, downTile.Name, left, leftTile.Name)
	}

	return result
}
