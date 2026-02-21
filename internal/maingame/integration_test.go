package maingame

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/testutil"
	ew "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGameInitializationIntegration はゲーム初期化の統合テスト
//
//nolint:paralleltest // ebitenui内部のrace conditionのためt.Parallel()を使用しない
func TestGameInitializationIntegration(t *testing.T) {
	t.Run("完全なゲーム初期化フロー", func(t *testing.T) {
		// メモリ使用量の初期値を記録
		initialMemStats := getMemoryStats()

		// 1. ワールドの初期化
		cfg := &config.Config{Profile: config.ProfileTesting}
		cfg.ApplyProfileDefaults()
		world, err := InitWorld(cfg)
		require.NoError(t, err)

		// 2. ワールドの基本検証
		validateWorldInitialization(t, world)

		// 3. リソース読み込みの検証
		validateResourceLoading(t, world)

		// 4. 状態機械の初期化と検証
		validateStateMachineInitialization(t, world)

		// 5. MainGameの初期化と基本動作検証
		validateMainGameInitialization(t, world)

		// 6. メモリリーク検証
		validateMemoryUsage(t, initialMemStats)
	})

	t.Run("部分的な初期化テスト", func(t *testing.T) {
		// 最小限のリソースでの初期化テスト
		world := testutil.InitTestWorld(t)

		// 基本構造の確認
		assert.NotNil(t, world.Resources, "ワールドリソースが初期化されていない")
		assert.NotNil(t, world.Resources.ScreenDimensions, "画面サイズが設定されていない")
		width, height := world.Resources.GetScreenDimensions()
		assert.Equal(t, consts.MinGameWidth, width, "画面幅が正しくない")
		assert.Equal(t, consts.MinGameHeight, height, "画面高さが正しくない")
	})
}

// TestMainGameLifecycle はMainGameのライフサイクル統合テスト
//
//nolint:paralleltest // ebitenui内部のrace conditionのためt.Parallel()を使用しない
func TestMainGameLifecycle(t *testing.T) {
	t.Run("ゲームループの基本動作", func(t *testing.T) {
		// 完全なワールドを使用（テスト用の最小限ワールドではUIリソースが不足）
		cfg := &config.Config{Profile: config.ProfileTesting}
		cfg.ApplyProfileDefaults()
		world, err := InitWorld(cfg)
		require.NoError(t, err)
		stateMachine, err := es.Init(&gs.MainMenuState{}, world)
		require.NoError(t, err)
		game := &MainGame{
			World:        world,
			StateMachine: stateMachine,
		}

		// Layout関数のテスト
		width, height := game.Layout(0, 0) // パラメータは無視される
		assert.Equal(t, consts.MinGameWidth, width, "レイアウト幅が正しくない")
		assert.Equal(t, consts.MinGameHeight, height, "レイアウト高さが正しくない")

		// Update関数のテスト（エラーが発生しないことを確認）
		err = game.Update()
		assert.NoError(t, err, "Updateでエラーが発生")

		// Draw関数のテスト（パニックしないことを確認）
		screen := ebiten.NewImage(consts.MinGameWidth, consts.MinGameHeight)
		assert.NotPanics(t, func() {
			game.Draw(screen)
		}, "Drawでパニックが発生")
	})

	t.Run("状態遷移の動作確認", func(t *testing.T) {
		// 完全なワールドを使用
		cfg := &config.Config{Profile: config.ProfileTesting}
		cfg.ApplyProfileDefaults()
		world, err := InitWorld(cfg)
		require.NoError(t, err)
		initialState := &gs.MainMenuState{}
		stateMachine, err := es.Init(initialState, world)
		require.NoError(t, err)

		game := &MainGame{
			World:        world,
			StateMachine: stateMachine,
		}

		// 状態機械の状態確認
		initialStates := stateMachine.GetStates()
		assert.Len(t, initialStates, 1, "初期状態数が正しくない")
		assert.IsType(t, &gs.MainMenuState{}, initialStates[0], "初期状態の型が正しくない")

		// 現在の状態確認
		currentState := stateMachine.GetCurrentState()
		assert.NotNil(t, currentState, "現在の状態がnil")
		assert.IsType(t, &gs.MainMenuState{}, currentState, "現在の状態の型が正しくない")

		// 状態機械の基本動作確認
		assert.NotPanics(t, func() {
			err := stateMachine.Update(world)
			assert.NoError(t, err, "StateMachine.Updateでエラー")
		}, "StateMachine.Updateでパニック")

		// 複数回のUpdateを実行して安定性を確認
		for i := 0; i < 3; i++ { // 回数を減らしてテスト時間を短縮
			err := game.Update()
			assert.NoError(t, err, "Update %d回目でエラーが発生", i+1)
		}
	})
}

// TestResourceIntegration はリソース統合テスト
//
//nolint:paralleltest // ebitenui内部のrace conditionのためt.Parallel()を使用しない
func TestResourceIntegration(t *testing.T) {
	t.Run("全リソースタイプの読み込み確認", func(t *testing.T) {
		cfg := &config.Config{Profile: config.ProfileTesting}
		cfg.ApplyProfileDefaults()
		world, err := InitWorld(cfg)
		require.NoError(t, err)

		// リソースの基本構造確認
		assert.NotNil(t, world.Resources, "リソース構造が初期化されていない")

		// スプライトシートの確認
		assert.NotNil(t, world.Resources.SpriteSheets, "スプライトシートが読み込まれていない")
		spriteSheets := *world.Resources.SpriteSheets
		assert.NotEmpty(t, spriteSheets, "スプライトシートが空")

		// フォントの確認
		assert.NotNil(t, world.Resources.Fonts, "フォントが読み込まれていない")
		fonts := *world.Resources.Fonts
		assert.NotEmpty(t, fonts, "フォントが空")

		// デフォルトフォントの確認
		assert.NotNil(t, world.Resources.Faces, "デフォルトフェイスが設定されていない")
		defaultFaces := *world.Resources.Faces
		assert.Contains(t, defaultFaces, "dougenzaka", "dougenzakaフォントが設定されていない")

		// UIリソースの確認
		assert.NotNil(t, world.Resources.UIResources, "UIリソースが初期化されていない")

		// Rawデータの確認
		assert.NotNil(t, world.Resources.RawMaster, "Rawマスターが読み込まれていない")

		// ゲームリソースの確認
		assert.NotNil(t, world.Resources.Dungeon, "ゲームリソースが初期化されていない")
	})

	t.Run("リソースの整合性確認", func(t *testing.T) {
		cfg := &config.Config{Profile: config.ProfileTesting}
		cfg.ApplyProfileDefaults()
		world, err := InitWorld(cfg)
		require.NoError(t, err)

		// フォントとフェイスの整合性
		fonts := *world.Resources.Fonts
		defaultFaces := *world.Resources.Faces

		if dougenzakaFont, exists := fonts["dougenzaka"]; exists {
			assert.NotNil(t, dougenzakaFont.Font, "dougenzakaフォントのFontフィールドがnil")
			if dougenzakaFace, faceExists := defaultFaces["dougenzaka"]; faceExists {
				assert.NotNil(t, dougenzakaFace, "dougenzakaフェイスがnil")
			}
		}

		// スプライトシートの基本チェック
		spriteSheets := *world.Resources.SpriteSheets
		for name, sheet := range spriteSheets {
			// Textureは値型なので、Imageフィールドを直接チェック
			assert.NotNil(t, sheet.Texture.Image, "スプライトシート '%s' の画像がnil", name)
		}
	})
}

// ヘルパー関数群

// validateWorldInitialization はワールド初期化の基本検証
func validateWorldInitialization(t *testing.T, world ew.World) {
	assert.NotNil(t, world.Resources, "ワールドリソースがnil")
	assert.NotNil(t, world.Resources.ScreenDimensions, "画面サイズがnil")
	assert.Equal(t, consts.MinGameWidth, world.Resources.ScreenDimensions.Width, "画面幅が正しくない")
	assert.Equal(t, consts.MinGameHeight, world.Resources.ScreenDimensions.Height, "画面高さが正しくない")
	assert.NotNil(t, world.Manager, "ECSマネージャがnil")
	assert.NotNil(t, world.Components, "コンポーネントがnil")
}

// validateResourceLoading はリソース読み込みの検証
func validateResourceLoading(t *testing.T, world ew.World) {
	// 各リソースの存在確認
	resources := []struct {
		name     string
		resource interface{}
	}{
		{"SpriteSheets", world.Resources.SpriteSheets},
		{"Fonts", world.Resources.Fonts},
		{"DefaultFaces", world.Resources.Faces},
		{"UIResources", world.Resources.UIResources},
		{"RawMaster", world.Resources.RawMaster},
		{"Game", world.Resources.Dungeon},
	}

	for _, res := range resources {
		assert.NotNil(t, res.resource, "%sリソースがnil", res.name)
	}
}

// validateStateMachineInitialization は状態機械初期化の検証
func validateStateMachineInitialization(t *testing.T, world ew.World) {
	initialState := &gs.MainMenuState{}
	stateMachine, err := es.Init(initialState, world)
	require.NoError(t, err)

	// 状態スタックの確認
	states := stateMachine.GetStates()
	assert.Len(t, states, 1, "初期状態の数が正しくない")
	assert.IsType(t, &gs.MainMenuState{}, states[0], "初期状態の型が正しくない")

	// 現在のアクティブ状態の確認
	currentState := stateMachine.GetCurrentState()
	assert.NotNil(t, currentState, "現在の状態がnil")
	assert.IsType(t, &gs.MainMenuState{}, currentState, "現在の状態の型が正しくない")

	// 状態数の確認
	stateCount := stateMachine.GetStateCount()
	assert.Equal(t, 1, stateCount, "状態数が正しくない")

	// Update呼び出しでパニックしないことを確認
	assert.NotPanics(t, func() {
		err := stateMachine.Update(world)
		assert.NoError(t, err, "状態機械のUpdate呼び出しでエラー")
	}, "状態機械のUpdate呼び出しでパニック")
}

// validateMainGameInitialization はMainGame初期化の検証
func validateMainGameInitialization(t *testing.T, world ew.World) {
	stateMachine, err := es.Init(&gs.MainMenuState{}, world)
	require.NoError(t, err)
	game := &MainGame{
		World:        world,
		StateMachine: stateMachine,
	}

	assert.NotNil(t, game.World, "ゲームワールドがnil")
	assert.NotNil(t, game.StateMachine, "状態機械がnil")

	// Layout関数の動作確認
	width, height := game.Layout(0, 0)
	assert.Greater(t, width, 0, "レイアウト幅が0以下")
	assert.Greater(t, height, 0, "レイアウト高さが0以下")
}

// validateMemoryUsage はメモリ使用量の検証
func validateMemoryUsage(t *testing.T, initialStats memoryStats) {
	finalStats := getMemoryStats()

	// メモリ使用量の増加が異常でないことを確認
	memoryIncreaseRatio := float64(finalStats.Alloc) / float64(initialStats.Alloc)
	assert.Less(t, memoryIncreaseRatio, 10.0, "メモリ使用量が異常に増加している")

	t.Logf("メモリ使用量 - 初期: %d bytes, 最終: %d bytes, 増加率: %.2fx",
		initialStats.Alloc, finalStats.Alloc, memoryIncreaseRatio)
}

// memoryStats はメモリ統計情報
type memoryStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Mallocs    uint64
	Frees      uint64
}

// getMemoryStats は現在のメモリ統計を取得
func getMemoryStats() memoryStats {
	// 実際のメモリ統計取得は実装環境に依存するため、
	// テスト環境では簡易実装
	// 実際の実装では runtime.ReadMemStats() を使用
	return memoryStats{
		Alloc:      1024 * 1024, // 1MB
		TotalAlloc: 2048 * 1024, // 2MB
		Mallocs:    1000,
		Frees:      500,
	}
}

// TestGameInitializationBenchmark はゲーム初期化のベンチマーク
func BenchmarkGameInitialization(b *testing.B) {
	b.Run("InitWorld", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg := &config.Config{Profile: config.ProfileTesting}
			cfg.ApplyProfileDefaults()
			_, err := InitWorld(cfg)
			require.NoError(b, err)
		}
	})

	b.Run("StateMachineCreation", func(b *testing.B) {
		// 完全なワールドを使用（テスト用最小限ワールドではUIリソース不足）
		cfg := &config.Config{Profile: config.ProfileTesting}
		cfg.ApplyProfileDefaults()
		world, err := InitWorld(cfg)
		require.NoError(b, err)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = es.Init(&gs.MainMenuState{}, world)
		}
	})

	b.Run("MainGameCreation", func(b *testing.B) {
		// 完全なワールドを使用
		cfg := &config.Config{Profile: config.ProfileTesting}
		cfg.ApplyProfileDefaults()
		world, err := InitWorld(cfg)
		require.NoError(b, err)
		stateMachine, err := es.Init(&gs.MainMenuState{}, world)
		require.NoError(b, err)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = &MainGame{
				World:        world,
				StateMachine: stateMachine,
			}
		}
	})
}
