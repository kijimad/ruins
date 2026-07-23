package components

import (
	"fmt"
)

// Interactable はプレイヤーと相互作用可能なエンティティを示すマーカー。
// 1つのエンティティが複数のインタラクションを持てる（例: 攻撃可能かつ収納を開ける木箱）
//
// 相互作用をここに束ねる基準: フィールドを持たないマーカーで、エンティティ単位で
// 全種別をまとめて列挙して使い（アクションメニュー生成など）、種別ごとのグローバル
// クエリを必要としないもの。この条件を外れる相互作用は独立コンポーネントにする:
//   - システムがその種別だけをグローバルにフィルタする
//   - 固有のフィールド/データを持つ（例: 行き先IDを持つポータル）
//   - 独立したシステムを単独で駆動する
type Interactable struct {
	Interactions []InteractionKind
}

// InteractionConfig は相互作用の設定
type InteractionConfig struct {
	ActivationRange ActivationRange // 発動範囲
	ActivationWay   ActivationWay   // 発動方式
}

// InteractionKind は相互作用の種類を表す。
// 種別ごとに Config() で発動プロトコル（発動範囲・発動方式）を持つ点が本質で、
// これは domain コンポーネント（Door の開閉状態、Dialog のメッセージキー等）とは
// 別レイヤの情報。例えば「扉の状態」は Door コンポーネントが、「扉としてどう発動
// する相互作用か」は InteractionDoor が担うため、両者は冗長ではない。
type InteractionKind string

const (
	// InteractionPortalNext は次の階層へ進むポータル
	InteractionPortalNext InteractionKind = "PORTAL_NEXT"
	// InteractionPortalPrev は1つ上の階層へ戻るポータル
	InteractionPortalPrev InteractionKind = "PORTAL_PREV"
	// InteractionDungeonEnter は遺跡入口の相互作用（発動でオーバーワールドから遺跡へ入る）
	InteractionDungeonEnter InteractionKind = "DUNGEON_ENTER"
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

// Config は種類に応じた相互作用設定を返す。未知の種類はゼロ値の無効な Config を返す。
// switch に default を置かず、既知種別の網羅漏れを exhaustive linter に検知させる。
// 未知入力は raw/save 由来でありうるので panic せず末尾のゼロ値へ graceful に落とす
func (k InteractionKind) Config() InteractionConfig {
	switch k {
	case InteractionPortalNext, InteractionPortalPrev, InteractionDungeonEnter, InteractionItem, InteractionItemAll:
		return InteractionConfig{ActivationRange: ActivationRangeSameTile, ActivationWay: ActivationWayManual}
	case InteractionDoor, InteractionTalk, InteractionMelee:
		return InteractionConfig{ActivationRange: ActivationRangeAdjacent, ActivationWay: ActivationWayOnCollision}
	case InteractionDoorLock:
		return InteractionConfig{ActivationRange: ActivationRangeSameTile, ActivationWay: ActivationWayAuto}
	case InteractionStorage:
		return InteractionConfig{ActivationRange: ActivationRangeAdjacent, ActivationWay: ActivationWayManual}
	}
	return InteractionConfig{}
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
