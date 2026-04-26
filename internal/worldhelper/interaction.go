package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
)

// IsInActivationRange はプレイヤーがトリガーの発動範囲内にいるかを判定する
func IsInActivationRange(playerGrid, triggerGrid *gc.GridElement, activationRange gc.ActivationRange) bool {
	switch activationRange {
	case gc.ActivationRangeSameTile:
		return playerGrid.X == triggerGrid.X && playerGrid.Y == triggerGrid.Y
	case gc.ActivationRangeAdjacent:
		return geometry.IsAdjacent(int(playerGrid.X), int(playerGrid.Y), int(triggerGrid.X), int(triggerGrid.Y))
	default:
		return false
	}
}
