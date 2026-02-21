// Package testutil はテスト用のユーティリティ関数を提供する
package testutil

import (
	"math/rand/v2"
	"sync"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/loader"
	gr "github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/require"
)

// 共有リソースをキャッシュ（一度だけ読み込む）
var (
	rawMasterOnce sync.Once
	rawMaster     interface{}
)

// InitTestWorld は軽量なテスト用Worldを初期化する
// フォントやスプライトシートなどの重いリソースは読み込まず、
// ECSとRawMasterのみを初期化します。
//
// この関数は以下のようなテストに適しています：
//   - エンティティ操作のテスト
//   - ゲームロジックのテスト
//   - アイテムやレシピのテスト
//   - UIを必要としないテスト
func InitTestWorld(t *testing.T) w.World {
	t.Helper()

	// 基本的なWorld構造を初期化
	world, err := w.InitWorld(&gc.Components{})
	require.NoError(t, err)

	// テスト用configを設定
	world.Config = &config.Config{Profile: config.ProfileDevelopment}
	world.Config.ApplyProfileDefaults()
	world.Config.LogLevel = "ignore"
	world.Config.Seed = rand.Uint64()
	world.Config.RNG = rand.New(rand.NewPCG(world.Config.Seed, 0))
	world.Resources.SetScreenDimensions(960, 720)

	// RawMasterのみを共有リソースから取得（一度だけ読み込む）
	rawMasterOnce.Do(func() {
		resourceLoader := loader.NewResourceLoader()
		rw, err := resourceLoader.LoadRaws()
		require.NoError(t, err, "RawMasterの読み込みに失敗しました")
		rawMaster = rw
	})

	require.NotNil(t, rawMaster, "RawMasterが初期化されていません")
	world.Resources.RawMaster = rawMaster

	// 最低限のゲームリソースを初期化
	gameResource := &gr.Dungeon{
		ExploredTiles: make(map[gc.GridElement]bool),
		MinimapSettings: gr.MinimapSettings{
			Width:   150,
			Height:  150,
			OffsetX: 10,
			OffsetY: 10,
			Scale:   3,
		},
	}
	err = gameResource.RequestStateChange(gr.NoneEvent{})
	require.NoError(t, err, "テスト初期化時の状態変更要求に失敗しました")
	world.Resources.Dungeon = gameResource

	return world
}
