// Package testutil はテスト用のユーティリティ関数を提供する
package testutil

import (
	"math/rand/v2"
	"sync"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"
)

// SetStageLevel は現ステージのメタにフィールド寸法を設定する。テストで地形寸法を差し替える用。
// フィールド寸法は StageMeta が持つため、Dungeon へ直接書いていた箇所の置き換えに使う。
// query の循環 import を避けるため world.Components を直接使う。
func SetStageLevel(world w.World, level gc.Level) {
	key := world.Components.Dungeon.Get(world.Resources.SingletonEntity).CurrentStage
	var found ecs.Entity
	ok := false
	q := ecs.NewFilter2[gc.StageMeta, gc.StageBound](world.ECS).Query()
	for q.Next() {
		if !ok && world.Components.StageBound.Get(q.Entity()).Key == key {
			found = q.Entity()
			ok = true
		}
	}
	if !ok {
		found = world.ECS.NewEntity()
		world.Components.StageBound.Add(found, &gc.StageBound{Key: key})
		world.Components.StageMeta.Add(found, &gc.StageMeta{})
	}
	world.Components.StageMeta.Get(found).Level = level
}

// 共有リソースをキャッシュ（一度だけ読み込む）
var (
	rawMasterOnce sync.Once
	rawMaster     oapi.Raws
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
func InitTestWorld(tb testing.TB) w.World {
	tb.Helper()

	// 基本的なWorld構造を初期化
	world, err := w.InitWorld(&gc.Components{})
	require.NoError(tb, err)

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
		require.NoError(tb, err, "RawMasterの読み込みに失敗しました")
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
				"red_ball":     {Width: 32, Height: 32},
			},
		},
	}
	world.Resources.SpriteSheets = spriteSheets

	// テスト用の現ステージを用意する。フィールド寸法は現ステージの StageMeta が持つため、
	// 現ステージをオーバーワールドに確定し、そのキーに束縛したメタを Level 付きで作る。
	// オーバーワールド判定は帯データ SeamlessBand の有無で行うので、帯を付けない既定では
	// IsOnOverworld は偽のまま。前線テストは EnsureSeamlessBand で帯を付ける。
	// query の循環 import を避けるため world.Components を直接使う
	d := world.Components.Dungeon.Get(world.Resources.SingletonEntity)
	d.CurrentStage = gc.NewOverworldStage()
	metaEntity := world.ECS.NewEntity()
	world.Components.StageBound.Add(metaEntity, &gc.StageBound{Key: d.CurrentStage})
	world.Components.StageMeta.Add(metaEntity, &gc.StageMeta{
		Level: gc.Level{TileWidth: 50, TileHeight: 50},
	})

	return world
}
