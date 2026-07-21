package components

import (
	"github.com/mlange-42/ark/ecs"
)

// StatePayload はステート遷移リクエストのペイロード。
// 種別ごとに専用の型を持ち、DungeonState が型スイッチで処理する。
// 非公開メソッドで実装先をこのパッケージ内に限定する。
type StatePayload interface{ isStatePayload() }

// WarpDescend は下り階段による1つ下の階への移動
type WarpDescend struct{}

// WarpAscend は上り階段による1つ上の階への移動
type WarpAscend struct{}

// WarpDungeonEnter は遺跡入口からの遺跡進入
type WarpDungeonEnter struct {
	// DefinitionName は進入する遺跡の定義名
	DefinitionName string
}

// GameClear はゲームクリア
type GameClear struct{}

// ShowDialog は会話メッセージの表示
type ShowDialog struct {
	MessageKey    string
	SpeakerEntity ecs.Entity
}

// OpenStorage は収納メニューを開く
type OpenStorage struct {
	StorageEntity ecs.Entity // 収納Propのエンティティ
}

func (WarpDescend) isStatePayload()      {}
func (WarpAscend) isStatePayload()       {}
func (WarpDungeonEnter) isStatePayload() {}
func (GameClear) isStatePayload()        {}
func (ShowDialog) isStatePayload()       {}
func (OpenStorage) isStatePayload()      {}

// StateChangeRequest はステート遷移リクエストを運ぶコンポーネント。
// Ark は具体型でコンポーネントを格納するため、Payload interface を包む薄いラッパーにする。
// 一時イベントで保存対象外（skipComponents）のため interface フィールドを持てる。
type StateChangeRequest struct {
	Payload StatePayload
}

// WarpDescendEvent は下り階段による移動リクエストを生成する
func WarpDescendEvent() StateChangeRequest { return StateChangeRequest{Payload: WarpDescend{}} }

// WarpAscendEvent は上り階段による移動リクエストを生成する
func WarpAscendEvent() StateChangeRequest { return StateChangeRequest{Payload: WarpAscend{}} }

// WarpDungeonEnterEvent は遺跡進入リクエストを生成する
func WarpDungeonEnterEvent(definitionName string) StateChangeRequest {
	return StateChangeRequest{Payload: WarpDungeonEnter{DefinitionName: definitionName}}
}

// GameClearEvent はゲームクリアリクエストを生成する
func GameClearEvent() StateChangeRequest { return StateChangeRequest{Payload: GameClear{}} }

// ShowDialogEvent は会話メッセージ表示リクエストを生成する
func ShowDialogEvent(messageKey string, speaker ecs.Entity) StateChangeRequest {
	return StateChangeRequest{Payload: ShowDialog{MessageKey: messageKey, SpeakerEntity: speaker}}
}

// OpenStorageEvent は収納メニューを開くリクエストを生成する
func OpenStorageEvent(storage ecs.Entity) StateChangeRequest {
	return StateChangeRequest{Payload: OpenStorage{StorageEntity: storage}}
}
