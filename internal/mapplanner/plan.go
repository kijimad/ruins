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
	var lastSeed uint64

	// 最大再試行回数まで繰り返す
	for attempt := 0; attempt < MaxPlanRetries; attempt++ {
		// 再試行時は異なるシードを使用
		currentSeed := seed + uint64(attempt*1000)
		lastSeed = currentSeed

		plan, err := attemptMetaPlan(world, width, height, currentSeed, plannerType)
		if err == nil {
			// 成功時にリトライがあった場合は警告ログを出力
			if attempt > 0 {
				fmt.Printf("マップ生成成功（試行回数: %d回, PlannerType: %s, 最終seed: %d）\n", attempt+1, plannerType.Name, currentSeed)
			}
			return plan, nil
		}

		lastErr = err

		// 接続性エラー以外は即座に失敗
		if !isConnectivityError(err) {
			return nil, fmt.Errorf("プラン生成失敗 (PlannerType=%s, seed=%d): %w", plannerType.Name, seed, err)
		}
	}

	// 全試行失敗時のエラーメッセージ（最後の試行のみ表示）
	return nil, fmt.Errorf("プラン生成に%d回失敗しました (PlannerType=%s, baseSeed=%d, 最終試行seed=%d)。最後のエラー: %w",
		MaxPlanRetries, plannerType.Name, seed, lastSeed, lastErr)
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

	// プランナーチェーンを実行（BridgeFacilityWrapper含まず）
	if err := chain.Plan(); err != nil {
		return nil, err
	}

	// 基本的な検証: プレイヤー開始位置があるか確認
	_, _, playerErr := chain.PlanData.GetPlayerStartPosition()
	if playerErr != nil {
		return nil, ErrPlayerPlacement
	}

	// 接続性検証: 最上列と最下列が接続されているかをチェックする
	pathFinder := NewPathFinder(&chain.PlanData)
	if err := pathFinder.ValidateConnectivity(); err != nil {
		return nil, err
	}

	// 橋facilityを追加（接続性検証後に実行）
	// テスト環境ではassetsディレクトリが存在しない場合があるため、
	// NewBridgeFacilityWrapperのエラー（パレット・チャンク登録エラー）のみスキップする
	// PlanMetaのエラー（実際のマップ生成エラー）は返す
	// TODO: Resourcesにもたせて、testWorldで初期化させればよさそう
	bridgeWrapper, err := NewBridgeFacilityWrapper()
	if err == nil {
		if err := bridgeWrapper.PlanMeta(&chain.PlanData); err != nil {
			return nil, err
		}
	}

	return &chain.PlanData, nil
}

// isConnectivityError は接続性エラーかどうかを判定する
func isConnectivityError(err error) bool {
	return errors.Is(err, ErrConnectivity) || errors.Is(err, ErrPlayerPlacement)
}
