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
	"github.com/kijimaD/ruins/internal/resources"
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
	stageKey   gc.StageKey
	stageLevel gc.Level
	ui         bool
}

// Option は InitTestWorld の初期化オプション。
type Option func(*initConfig)

// WithCurrentStage は初期化時の現ステージキーを指定する。省略時はオーバーワールド。
// ステージ跨ぎのテストで、最初から特定のステージ上で始めたいときに使う。
func WithCurrentStage(key gc.StageKey) Option {
	return func(c *initConfig) { c.stageKey = key }
}

// WithStageLevel は現ステージのフィールド寸法を指定する。省略時は 50x50。
// 実ゲームではフィールド寸法はステージ生成時に一度決まるため、テストも生成相当の初期化時に与える。
func WithStageLevel(level gc.Level) Option {
	return func(c *initConfig) { c.stageLevel = level }
}

// WithUI はフォントフェイス込みの UIResources を積む。UI を描くテストはこれを付ける。
// フェイスはテストが排他所有する。text/v2 は GoTextFaceSource 内に可変キャッシュを持ち、
// 共有フェイスを並行描画すると競合するが、排他所有ならロック無しで並列に実描画できる。
// フルゲームを構築する重い vrt.InitReplayWorld は、実プレイどおりフルフレームを駆動する
// states の golden_replay だけに使う。
func WithUI() Option {
	return func(c *initConfig) { c.ui = true }
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

	cfg := initConfig{stageKey: gc.NewOverworldStage(), stageLevel: gc.Level{TileWidth: 50, TileHeight: 50}}
	for _, opt := range opts {
		opt(&cfg)
	}

	// テスト用configを構築してからWorldを初期化する。シングルトンが config を読むため先に渡す。
	testCfg := &config.Config{Profile: config.ProfileDevelopment}
	testCfg.ApplyProfileDefaults()
	world, err := w.InitWorld(&gc.Components{}, testCfg)
	require.NoError(tb, err)

	world.Resources.Config.LogLevel = "ignore"
	world.Resources.Config.Seed = rand.Uint64()
	world.Resources.Config.RNG = rand.New(rand.NewPCG(world.Resources.Config.Seed, 0))
	world.Resources.SetScreenDimensions(960, 720)

	// RawMasterのみを共有リソースから取得（一度だけ読み込む）
	rawMasterOnce.Do(func() {
		rw, err := loader.LoadRaws()
		require.NoError(tb, err, "failed to load RawMaster")
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
	// 現ステージを cfg.stageKey に確定し、そのキーに束縛した StageField を Level 付きで作る。
	// 実ゲームでも寸法はステージ生成時に一度決まるので、ここで与える。既定は overworld・50x50、
	// WithCurrentStage/WithStageLevel で上書き。
	// オーバーワールド判定は帯データ SeamlessBand の有無で行うので、帯を付けない既定では
	// IsOnOverworld は偽のまま。帯を要するテストは EnsureSeamlessBand で付ける。
	// query の循環 import を避けるため world.Components を直接使う
	d := world.Components.Dungeon.Get(world.Resources.SingletonEntity)
	d.CurrentStage = cfg.stageKey
	fieldEntity := world.ECS.NewEntity()
	world.Components.StageBound.Add(fieldEntity, &gc.StageBound{Key: cfg.stageKey})
	field := gc.NewStageField()
	field.Level = cfg.stageLevel
	world.Components.StageField.Add(fieldEntity, field)

	if cfg.ui {
		world.Resources.UIResources = borrowUIResources(tb)
	}

	return world
}

// uiResPool は WithUI が積む UIResources の置き場。フェイスの構築は重く、描画は
// フェイス内部のキャッシュへ書き込むため同時共有はできない。そこで排他所有で貸し出す。
// 並行するテストは必ず別インスタンスを掴んで独立し、順に走るテストは温まった
// キャッシュごと再利用して構築を省く。ポインタを入れるのは Put のボクシングを避けるため
var uiResPool sync.Pool

// borrowUIResources はプールから UIResources を借り、テスト終了時に返す。
// 空なら独立したフェイス一式を構築する
func borrowUIResources(tb testing.TB) resources.UIResources {
	tb.Helper()
	uir, ok := uiResPool.Get().(*resources.UIResources)
	if !ok {
		fonts, err := loader.LoadFonts()
		require.NoError(tb, err)
		fresh, err := loader.LoadUIResources(fonts)
		require.NoError(tb, err)
		uir = &fresh
	}
	tb.Cleanup(func() { uiResPool.Put(uir) })
	return *uir
}
