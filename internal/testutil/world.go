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
	"github.com/stretchr/testify/require"
)

// 共有リソースをキャッシュ（一度だけ読み込む）
var (
	rawMasterOnce sync.Once
	rawMaster     oapi.Raws
)

// initConfig は InitTestWorld の初期化オプションを集約する。
type initConfig struct {
	stageLevel gc.Level
}

// Option は InitTestWorld の初期化オプション。
type Option func(*initConfig)

// WithStageLevel は現ステージのフィールド寸法を指定する。省略時は 50x50。
// 実ゲームではフィールド寸法はステージ生成時に一度決まるため、テストも生成相当の初期化時に与える。
func WithStageLevel(level gc.Level) Option {
	return func(c *initConfig) { c.stageLevel = level }
}

// InitTestWorld は軽量なテスト用Worldを初期化する
// フォントやスプライトシートなどの重いリソースは読み込まず、
// ECSとRawMasterのみを初期化します。
//
// この関数は以下のようなテストに適しています：
//   - エンティティ操作のテスト
//   - ゲームロジックのテスト
//   - アイテムやレシピのテスト
//   - UIを必要としないテスト
func InitTestWorld(tb testing.TB, opts ...Option) w.World {
	tb.Helper()

	cfg := initConfig{stageLevel: gc.Level{TileWidth: 50, TileHeight: 50}}
	for _, opt := range opts {
		opt(&cfg)
	}

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

	// テスト用の現ステージを用意する。フィールド寸法は現ステージの StageField が持つため、
	// 現ステージをオーバーワールドに確定し、そのキーに束縛した StageField を Level 付きで作る。
	// 実ゲームでも寸法はステージ生成時に一度決まるので、ここで与える。既定は 50x50、WithStageLevel で上書き。
	// オーバーワールド判定は帯データ SeamlessBand の有無で行うので、帯を付けない既定では
	// IsOnOverworld は偽のまま。前線テストは EnsureSeamlessBand で帯を付ける。
	// query の循環 import を避けるため world.Components を直接使う
	d := world.Components.Dungeon.Get(world.Resources.SingletonEntity)
	d.CurrentStage = gc.NewOverworldStage()
	fieldEntity := world.ECS.NewEntity()
	world.Components.StageBound.Add(fieldEntity, &gc.StageBound{Key: d.CurrentStage})
	field := gc.NewStageField()
	field.Level = cfg.stageLevel
	world.Components.StageField.Add(fieldEntity, field)

	return world
}
