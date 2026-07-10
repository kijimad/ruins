package components

import (
	ecs "github.com/x-hgg-x/goecs/v2"
)

// EventKind はステート遷移リクエストの種別を表す
type EventKind string

const (
	// EventWarpNext は次の階層への移動
	EventWarpNext EventKind = "WarpNext"
	// EventWarpEscape は脱出ポータルによる帰還
	EventWarpEscape EventKind = "WarpEscape"
	// EventGameClear はゲームクリア
	EventGameClear EventKind = "GameClear"
	// EventShowDialog は会話メッセージの表示
	EventShowDialog EventKind = "ShowDialog"
	// EventOpenDungeonSelect はダンジョン選択メニューを開く
	EventOpenDungeonSelect EventKind = "OpenDungeonSelect"
	// EventOpenStorage は収納メニューを開く
	EventOpenStorage EventKind = "OpenStorage"
)

// StateChangeRequest はステート遷移リクエストを表すコンポーネント。
// 以前はマーカーインターフェースと各イベント構造体だったが、Ark(archetype格納)と
// serde 互換のため Kind 判別子を持つプレーンデータのタグ付きユニオンに平坦化した。
// DungeonState で Kind により処理される。
type StateChangeRequest struct {
	Kind EventKind

	// EventShowDialog 用
	MessageKey    string
	SpeakerEntity ecs.Entity

	// EventOpenStorage 用
	StorageEntity ecs.Entity // 収納Propのエンティティ
}

// WarpNextEvent は次の階層への移動リクエストを生成する
func WarpNextEvent() StateChangeRequest { return StateChangeRequest{Kind: EventWarpNext} }

// WarpEscapeEvent は脱出ポータルによる帰還リクエストを生成する
func WarpEscapeEvent() StateChangeRequest { return StateChangeRequest{Kind: EventWarpEscape} }

// GameClearEvent はゲームクリアリクエストを生成する
func GameClearEvent() StateChangeRequest { return StateChangeRequest{Kind: EventGameClear} }

// ShowDialogEvent は会話メッセージ表示リクエストを生成する
func ShowDialogEvent(messageKey string, speaker ecs.Entity) StateChangeRequest {
	return StateChangeRequest{Kind: EventShowDialog, MessageKey: messageKey, SpeakerEntity: speaker}
}

// OpenDungeonSelectEvent はダンジョン選択メニューを開くリクエストを生成する
func OpenDungeonSelectEvent() StateChangeRequest {
	return StateChangeRequest{Kind: EventOpenDungeonSelect}
}

// OpenStorageEvent は収納メニューを開くリクエストを生成する
func OpenStorageEvent(storage ecs.Entity) StateChangeRequest {
	return StateChangeRequest{Kind: EventOpenStorage, StorageEntity: storage}
}
