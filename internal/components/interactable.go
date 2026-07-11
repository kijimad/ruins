package components

import (
	"fmt"
)

// Interactable はプレイヤーと相互作用可能なエンティティを示すマーカー。
// 1つのエンティティが複数のインタラクションを持てる（例: 攻撃可能かつ収納を開ける木箱）
type Interactable struct {
	Interactions []InteractionData
}

// InteractionConfig は相互作用の設定
type InteractionConfig struct {
	ActivationRange ActivationRange // 発動範囲
	ActivationWay   ActivationWay   // 発動方式
}

// InteractionKind は相互作用の種類を表す判別子
type InteractionKind string

const (
	// InteractionPortal はポータルを通る相互作用
	InteractionPortal InteractionKind = "PORTAL"
	// InteractionDungeonGate はダンジョン選択門の相互作用（発動でダンジョン選択メニューを開く）
	InteractionDungeonGate InteractionKind = "DUNGEON_GATE"
	// InteractionDoor は扉の相互作用
	InteractionDoor InteractionKind = "DOOR"
	// InteractionDoorLock はプレイヤーが踏むと全扉をロックする相互作用
	InteractionDoorLock InteractionKind = "DOOR_LOCK"
	// InteractionTalk は会話の相互作用
	InteractionTalk InteractionKind = "TALK"
	// InteractionItem はアイテム拾得の相互作用
	InteractionItem InteractionKind = "ITEM"
	// InteractionItemAll は同一タイル上の全アイテム拾得の相互作用
	InteractionItemAll InteractionKind = "ITEM_ALL"
	// InteractionStorage は収納の相互作用
	InteractionStorage InteractionKind = "STORAGE"
	// InteractionMelee は近接攻撃の相互作用
	InteractionMelee InteractionKind = "MELEE"
)

// PortalType はポータルの種類を表す
type PortalType string

const (
	// PortalTypeNext は次の階層へのポータル
	PortalTypeNext PortalType = "NEXT"
	// PortalTypeTown は街への帰還ポータル
	PortalTypeTown PortalType = "TOWN"
)

// InteractionData は相互作用のデータ。
// Kindで種類を判別するタグ付きデータ。serde互換のためinterfaceを排する
type InteractionData struct {
	Kind InteractionKind
	// PortalType は Kind==InteractionPortal のときのみ有効
	PortalType PortalType
}

// Config は種類に応じた相互作用設定を返す。未知の種類はゼロ値を返す
func (d InteractionData) Config() InteractionConfig {
	switch d.Kind {
	case InteractionPortal, InteractionDungeonGate, InteractionItem, InteractionItemAll:
		return InteractionConfig{ActivationRange: ActivationRangeSameTile, ActivationWay: ActivationWayManual}
	case InteractionDoor, InteractionTalk, InteractionMelee:
		return InteractionConfig{ActivationRange: ActivationRangeAdjacent, ActivationWay: ActivationWayOnCollision}
	case InteractionDoorLock:
		return InteractionConfig{ActivationRange: ActivationRangeSameTile, ActivationWay: ActivationWayAuto}
	case InteractionStorage:
		return InteractionConfig{ActivationRange: ActivationRangeAdjacent, ActivationWay: ActivationWayManual}
	default:
		return InteractionConfig{}
	}
}

// ActivationRange は相互作用の発動範囲を表す
type ActivationRange string

const (
	// ActivationRangeSameTile は直上（同じタイル）で発動
	ActivationRangeSameTile ActivationRange = "SAME_TILE"
	// ActivationRangeAdjacent は隣接タイルで発動
	ActivationRangeAdjacent ActivationRange = "ADJACENT"
)

// Valid はActivationRangeの値が有効かを検証する
func (enum ActivationRange) Valid() error {
	switch enum {
	case ActivationRangeSameTile, ActivationRangeAdjacent:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}

// ================

// ActivationWay は相互作用の発動方式を表す
type ActivationWay string

const (
	// ActivationWayAuto は自動発動（範囲内に入ったら即座に発動）
	ActivationWayAuto ActivationWay = "AUTO"
	// ActivationWayManual は手動発動（Enterキーやアクションメニューで発動）
	ActivationWayManual ActivationWay = "MANUAL"
	// ActivationWayOnCollision は移動先衝突時に自動発動（移動先として指定された時に発動）
	ActivationWayOnCollision ActivationWay = "ON_COLLISION"
)

// Valid はActivationWayの値が有効かを検証する
func (enum ActivationWay) Valid() error {
	switch enum {
	case ActivationWayAuto, ActivationWayManual, ActivationWayOnCollision:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}
