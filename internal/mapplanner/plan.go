package mapplanner

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
)

const (
	// MaxPlanRetries はプランナーチェーンの最大再試行回数
	MaxPlanRetries = 10
)

var (
	// ErrConnectivity は接続性エラーを表す
	ErrConnectivity = errors.New("マップ接続性エラー")
	// ErrPlayerPlacement はプレイヤー配置エラーを表す
	ErrPlayerPlacement = errors.New("プレイヤー配置可能な床タイルが見つかりません")
)

// Plan はPlannerChainを初期化してMetaPlanを返す
func Plan(world w.World, width, height int, seed uint64, plannerType PlannerType) (*MetaPlan, error) {
	var lastErr error

	// 最大再試行回数まで繰り返す
	for attempt := 0; attempt < MaxPlanRetries; attempt++ {
		// 再試行時は異なるシードを使用
		currentSeed := seed + uint64(attempt*1000)

		plan, err := attemptMetaPlan(world, width, height, currentSeed, plannerType)
		if err == nil {
			return plan, nil
		}

		lastErr = err
		// 接続性エラー以外は即座に失敗
		if !isConnectivityError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("プラン生成に%d回失敗しました。最後のエラー: %w", MaxPlanRetries, lastErr)
}

// attemptMetaPlan は単一回のメタプラン生成を試行する
func attemptMetaPlan(world w.World, width, height int, seed uint64, plannerType PlannerType) (*MetaPlan, error) {
	// PlannerChainを初期化
	var chain *PlannerChain
	var err error
	if plannerType.Name == PlannerTypeRandom.Name {
		chain, err = NewRandomPlanner(gc.Tile(width), gc.Tile(height), seed)
	} else {
		chain, err = plannerType.PlannerFunc(gc.Tile(width), gc.Tile(height), seed)
	}
	if err != nil {
		return nil, err
	}

	// RawMasterを設定
	if world.Resources != nil && world.Resources.RawMaster != nil {
		if rawMaster, ok := world.Resources.RawMaster.(*raw.Master); ok {
			chain.PlanData.RawMaster = rawMaster
		}
	}

	// 敵NPCプランナーを追加
	if plannerType.SpawnEnemies {
		hostileNPCPlanner := NewHostileNPCPlanner(world, plannerType)
		chain.With(hostileNPCPlanner)
	}

	// アイテムプランナーを追加
	if plannerType.SpawnItems {
		itemPlanner := NewItemPlanner(world, plannerType)
		chain.With(itemPlanner)
	}

	// Propsプランナーを追加
	propsPlanner := NewPropsPlanner(world, plannerType)
	chain.With(propsPlanner)

	// 橋facilityを追加（全ダンジョンタイプに必須）
	// テスト環境ではassetsがない場合があるため、エラー時はスキップする
	bridgeWrapper, err := NewBridgeFacilityWrapper()
	if err == nil {
		chain.With(bridgeWrapper)
	}

	// プランナーチェーンを実行
	if err := chain.Plan(); err != nil {
		return nil, err
	}

	// 基本的な検証: プレイヤー開始位置があるか確認
	_, _, hasPlayer := chain.PlanData.GetPlayerStartPosition()
	if !hasPlayer {
		return nil, ErrPlayerPlacement
	}

	// 橋システムの接続性検証
	// 上部橋エリアと下部橋エリアが接続されているかをチェックする
	pathFinder := NewPathFinder(&chain.PlanData)
	if err := pathFinder.ValidateConnectivity(); err != nil {
		return nil, err
	}

	return &chain.PlanData, nil
}

// isConnectivityError は接続性エラーかどうかを判定する
func isConnectivityError(err error) bool {
	return errors.Is(err, ErrConnectivity) || errors.Is(err, ErrPlayerPlacement)
}
