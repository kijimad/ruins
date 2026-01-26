package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/maptemplate"
)

// Interactable はプレイヤーと相互作用可能なエンティティを示すマーカー
type Interactable struct {
	Data InteractionData
}

// InteractionConfig は相互作用の設定
type InteractionConfig struct {
	ActivationRange ActivationRange // 発動範囲
	ActivationWay   ActivationWay   // 発動方式
}

// InteractionData は相互作用のデータインターフェース
type InteractionData interface {
	Config() InteractionConfig
}

// BridgeInteraction は出口を通る相互作用
type BridgeInteraction struct {
	BridgeID maptemplate.ExitID // 出口の識別子
}

// Config は相互作用設定を返す
func (b BridgeInteraction) Config() InteractionConfig {
	return InteractionConfig{
		ActivationRange: ActivationRangeSameTile,
		ActivationWay:   ActivationWayAuto, // 出口に到達したら自動遷移
	}
}

// PlazaWarpInteraction は街広場へのワープ相互作用（5の倍数の階層用）
type PlazaWarpInteraction struct{}

// Config は相互作用設定を返す
func (p PlazaWarpInteraction) Config() InteractionConfig {
	return InteractionConfig{
		ActivationRange: ActivationRangeSameTile,
		ActivationWay:   ActivationWayAuto, // 橋の端に到達したら自動遷移
	}
}

// DoorInteraction はドアの相互作用
type DoorInteraction struct{}

// Config は相互作用設定を返す
func (t DoorInteraction) Config() InteractionConfig {
	return InteractionConfig{
		ActivationRange: ActivationRangeAdjacent,
		ActivationWay:   ActivationWayOnCollision,
	}
}

// TalkInteraction は会話の相互作用
type TalkInteraction struct{}

// Config は相互作用設定を返す
func (t TalkInteraction) Config() InteractionConfig {
	return InteractionConfig{
		ActivationRange: ActivationRangeAdjacent,
		ActivationWay:   ActivationWayOnCollision,
	}
}

// ItemInteraction はアイテム拾得の相互作用
type ItemInteraction struct{}

// Config は相互作用設定を返す
func (t ItemInteraction) Config() InteractionConfig {
	return InteractionConfig{
		ActivationRange: ActivationRangeSameTile,
		ActivationWay:   ActivationWayManual,
	}
}

// MeleeInteraction は近接攻撃の相互作用
type MeleeInteraction struct{}

// Config は相互作用設定を返す
func (t MeleeInteraction) Config() InteractionConfig {
	return InteractionConfig{
		ActivationRange: ActivationRangeAdjacent,
		ActivationWay:   ActivationWayOnCollision,
	}
}

// TestTriggerInteraction はテスト用のトリガー相互作用
// テストで実行されたことを確認するためのフラグを持つ
type TestTriggerInteraction struct {
	Executed *bool // 実行されたかを記録するフラグ
}

// Config は相互作用設定を返す
func (t TestTriggerInteraction) Config() InteractionConfig {
	return InteractionConfig{
		ActivationRange: ActivationRangeSameTile,
		ActivationWay:   ActivationWayAuto,
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
