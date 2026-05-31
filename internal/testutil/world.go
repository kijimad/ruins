// Package testutil はテスト用のユーティリティ関数を提供する
package testutil

import (
	"math/rand/v2"
	"sync"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/require"
)

// 共有リソースをキャッシュ（一度だけ読み込む）
var (
	rawMasterOnce sync.Once
	rawMaster     raw.Master
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
		rw, err := loader.LoadRaws()
		require.NoError(t, err, "RawMasterの読み込みに失敗しました")
		rawMaster = rw
	})
	world.Resources.RawMaster = rawMaster

	// テスト用スプライトシートを初期化
	spriteSheets := map[string]gc.SpriteSheet{
		"field": {
			Sprites: map[string]gc.Sprite{
				"void":         {Width: 32, Height: 32},
				"wall_generic": {Width: 32, Height: 32},
				"floor":        {Width: 32, Height: 32},
				"player":       {Width: 32, Height: 32},
				"player_0":     {Width: 32, Height: 32},
				"player_1":     {Width: 32, Height: 32},
				"warp_next":    {Width: 32, Height: 32},
				"warp_escape":  {Width: 32, Height: 32},
				"red_ball":     {Width: 32, Height: 32},
			},
		},
	}
	world.Resources.SpriteSheets = spriteSheets

	// テスト用のLevel設定
	d := world.Components.DungeonState.Get(world.Resources.SingletonEntity).(*gc.Dungeon)
	d.Level = gc.Level{
		TileWidth:  50,
		TileHeight: 50,
	}

	return world
}
