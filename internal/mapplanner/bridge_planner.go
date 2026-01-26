// Package mapplanner の出口・スポーン地点配置仕様
package mapplanner

import "github.com/kijimaD/ruins/internal/maptemplate"

// ExitSpec は出口配置仕様を表す
// 階層間の遷移ポイントを定義する
type ExitSpec struct {
	X      int                // X座標
	Y      int                // Y座標
	ExitID maptemplate.ExitID // 出口ID
}

// SpawnPointSpec はスポーン地点仕様を表す
// プレイヤーの初期配置位置を定義する
type SpawnPointSpec struct {
	X int // X座標
	Y int // Y座標
}
