package mapplanner

import (
	"errors"
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

const (
	// MaxPlanRetries はプランナーチェーンの最大再試行回数
	MaxPlanRetries = 10
)

var (
	// ErrConnectivity は接続性エラーを表す
	ErrConnectivity = errors.New("マップ接続性エラー")
)

// Plan はPlannerChainを初期化してMetaPlanを返す
func Plan(world w.World, width, height consts.Tile, seed uint64, plannerType PlannerType) (*MetaPlan, error) {
	var lastErr error

	// 最大再試行回数まで繰り返す
	for attempt := range MaxPlanRetries {
		// 再試行時は異なるシードを使用
		currentSeed := seed + uint64(attempt*1000)

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
		if !errors.Is(err, ErrConnectivity) {
			return nil, fmt.Errorf("プラン生成失敗 (PlannerType=%s, seed=%d): %w", plannerType.Name, seed, err)
		}
	}

	// 全試行失敗時のエラーメッセージ（最後の試行のみ表示）
	return nil, fmt.Errorf("プラン生成に%d回失敗しました (PlannerType=%s, seed=%d)。最後のエラー: %w",
		MaxPlanRetries, plannerType.Name, seed, lastErr)
}

// BuildChain はPlannerChainを構築して返す。
// チェーンの構築ロジックを共有するために公開している
func BuildChain(world w.World, width, height consts.Tile, seed uint64, plannerType PlannerType) (*PlannerChain, error) {
	var chain *PlannerChain
	var err error
	if plannerType.Name == PlannerTypeRandom.Name {
		chain, err = NewRandomPlanner(width, height, seed)
	} else {
		chain, err = plannerType.PlannerFunc(width, height, seed)
	}
	if err != nil {
		return nil, err
	}

	// RawMasterを設定
	if world.Resources != nil {
		chain.PlanData.RawMaster = &world.Resources.RawMaster
	}

	// 生成中フロアの深度をプランへ焼き込む。深度依存の抽選が世界の現在地に依存しないようにする
	chain.PlanData.Depth = plannerType.Depth

	chain.With(NewHostileNPCPlanner(world, plannerType))
	chain.With(NewItemPlanner(world, plannerType))
	chain.With(NewPortalPlanner(world, plannerType))

	return chain, nil
}

// attemptMetaPlan は単一回のメタプラン生成を試行する
func attemptMetaPlan(world w.World, width, height consts.Tile, seed uint64, plannerType PlannerType) (*MetaPlan, error) {
	chain, err := BuildChain(world, width, height, seed, plannerType)
	if err != nil {
		return nil, err
	}

	if err := chain.Plan(); err != nil {
		return nil, err
	}

	// ポータル到達性検証: プレイヤー開始位置から全ポータルへ到達可能かチェック
	pathFinder := NewPathFinder(&chain.PlanData)
	if err := pathFinder.ValidatePortalReachability(); err != nil {
		return nil, err
	}

	return &chain.PlanData, nil
}
