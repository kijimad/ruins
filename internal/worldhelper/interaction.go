package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
)

// IsInActivationRange はプレイヤーがトリガーの発動範囲内にいるかを判定する
func IsInActivationRange(playerGrid, triggerGrid *gc.GridElement, activationRange gc.ActivationRange) bool {
	switch activationRange {
	case gc.ActivationRangeSameTile:
		// 直上（同じタイル）
		return playerGrid.X == triggerGrid.X && playerGrid.Y == triggerGrid.Y
	case gc.ActivationRangeAdjacent:
		// 隣接タイル（近傍8タイル、同じタイルは含まない）
		diffX := int(playerGrid.X) - int(triggerGrid.X)
		diffY := int(playerGrid.Y) - int(triggerGrid.Y)
		dx := max(diffX, -diffX)
		dy := max(diffY, -diffY)
		return dx <= 1 && dy <= 1 && (dx != 0 || dy != 0)
	default:
		return false
	}
}
