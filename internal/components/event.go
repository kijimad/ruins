package components

import (
	ecs "github.com/x-hgg-x/goecs/v2"
)

// StateChangeRequest はステート遷移リクエストを表すマーカーインターフェース。
// 各構造体が実装し、DungeonStateで型スイッチにより処理される
type StateChangeRequest interface {
	stateChangeRequest()
}

// WarpNextEvent は次の階層への移動を表す
type WarpNextEvent struct{}

func (WarpNextEvent) stateChangeRequest() {}

// WarpEscapeEvent は脱出ポータルによる帰還を表す
type WarpEscapeEvent struct{}

func (WarpEscapeEvent) stateChangeRequest() {}

// GameClearEvent はゲームクリアを表す
type GameClearEvent struct{}

func (GameClearEvent) stateChangeRequest() {}

// ShowDialogEvent は会話メッセージの表示を表す
type ShowDialogEvent struct {
	MessageKey    string
	SpeakerEntity ecs.Entity
}

func (ShowDialogEvent) stateChangeRequest() {}

// OpenDungeonSelectEvent はダンジョン選択メニューを開くことを表す
type OpenDungeonSelectEvent struct{}

func (OpenDungeonSelectEvent) stateChangeRequest() {}
