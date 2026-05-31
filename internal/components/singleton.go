package components

import "github.com/kijimaD/ruins/internal/gamelog"

// GameLog はゲームログストレージを保持するシングルトンコンポーネント
type GameLog struct {
	Store *gamelog.SafeSlice
}
