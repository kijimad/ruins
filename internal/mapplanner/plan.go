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
	ErrConnectivity = errors.New("マップ接続性エラー: プレイヤーからワープホールに到達できません")
	// ErrPlayerPlacement はプレイヤー配置エラーを表す
	ErrPlayerPlacement = errors.New("プレイヤー配置可能な床タイルが見つかりません")
	// ErrNoWarpPortal はワープポータルが存在しないエラーを表す
	ErrNoWarpPortal = errors.New("マップにワープポータルが配置されていません")
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

	// ワープポータルプランナーを追加する
	if len(chain.PlanData.WarpPortals) == 0 {
		warpPlanner := NewWarpPortalPlanner(world, plannerType)
		chain.With(warpPlanner)
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

	// プランナーチェーンを実行
	if err := chain.Plan(); err != nil {
		return nil, err
	}

	// 基本的な検証: プレイヤー開始位置があるか確認
	playerX, playerY, hasPlayer := chain.PlanData.GetPlayerStartPosition()
	if !hasPlayer {
		return nil, ErrPlayerPlacement
	}

	// MetaPlan用の接続性検証
	pathFinder := NewPathFinder(&chain.PlanData)
	if err := pathFinder.ValidateConnectivity(playerX, playerY); err != nil {
		return nil, err
	}

	return &chain.PlanData, nil
}

// isConnectivityError は接続性エラーかどうかを判定する
func isConnectivityError(err error) bool {
	return errors.Is(err, ErrConnectivity) || errors.Is(err, ErrPlayerPlacement) || errors.Is(err, ErrNoWarpPortal)
}
