// Package mapplanner の橋配置仕様
package mapplanner

import "github.com/kijimaD/ruins/internal/maptemplate"

// BridgeSpec は橋配置仕様を表す
type BridgeSpec struct {
	X        int                  // X座標
	Y        int                  // Y座標
	BridgeID maptemplate.BridgeID // 橋ID
}
