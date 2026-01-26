// Package mapplanner の橋配置仕様
package mapplanner

// BridgeSpec は橋配置仕様を表す
type BridgeSpec struct {
	X        int    // X座標
	Y        int    // Y座標
	BridgeID string // 橋ID ("A", "B", "C", "D")
}
