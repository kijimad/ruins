// Package config はアプリケーションの設定管理を提供する
//
// このパッケージは github.com/caarlos0/env/v11 を使用して環境変数からの
// 設定読み込みを行う。設定はworld.Configに保持され、
// world経由でアクセスする。
//
// # 使用可能な環境変数
//
// ## プロファイル設定
//   - RUINS_PROFILE: 環境プロファイル (デフォルト: production)
//   - "production": 本番環境 (デバッグ機能無効、軽量設定)
//   - "development": 開発環境 (デバッグ機能有効、開発効率重視)
//
// ## ウィンドウ設定
//   - RUINS_WINDOW_WIDTH: ウィンドウ幅 (デフォルト: 960)
//   - RUINS_WINDOW_HEIGHT: ウィンドウ高さ (デフォルト: 720)
//   - RUINS_FULLSCREEN: フルスクリーンモード (デフォルト: false)
//
// ## デバッグ設定
//   - RUINS_DEBUG: デバッグモード (デフォルト: false)
//   - RUINS_LOG_LEVEL: ログレベル (デフォルト: info) "debug", "info", "warn", "error", "fatal", "ignore"
//   - RUINS_LOG_CATEGORIES: カテゴリ別ログレベル設定 (例: "battle=debug,render=warn")
//   - RUINS_DEBUG_PPROF: pprofサーバー起動 (デフォルト: true)
//   - RUINS_PPROF_PORT: pprofサーバーポート (デフォルト: 6060)
//   - RUINS_SHOW_MONITOR: パフォーマンスモニター表示 (デフォルト: false)
//
// ## ゲーム設定
//   - RUINS_STARTING_STATE: 開始ステート (デフォルト: main_menu)
//   - "main_menu": メインメニュー
//   - "debug_menu": デバッグメニュー
//   - "dungeon": ダンジョン
//   - RUINS_SEED: 乱数シード (デフォルト: ランダム生成)
//   - 指定すると同じシードで再現可能なゲームプレイが可能
//
// ## パフォーマンス設定
//   - RUINS_TARGET_FPS: 目標フレームレート (デフォルト: 60)
//   - RUINS_PROFILE_MEMORY: メモリプロファイル (デフォルト: true)
//   - RUINS_PROFILE_CPU: CPUプロファイル (デフォルト: false)
//   - RUINS_PROFILE_MUTEX: Mutexプロファイル (デフォルト: false)
//   - RUINS_PROFILE_TRACE: トレースプロファイル (デフォルト: false)
//   - RUINS_PROFILE_PATH: プロファイル出力パス (デフォルト: ".")
//
// # 使用例
//
//	// 設定の読み込み（起動時）
//	cfg, err := config.Load()
//	if err != nil {
//		log.Fatal(err)
//	}
//	world, _ := maingame.InitWorld(cfg)
//
//	// world経由での設定アクセス
//	if world.Config.Profile == config.ProfileDevelopment {
//		log.Println("Development mode")
//	}
//
//	// ウィンドウサイズの取得
//	width := world.Config.WindowWidth
//	height := world.Config.WindowHeight
//
//	// デバッグモードの確認
//	if world.Config.Debug {
//		log.Println("Debug mode enabled")
//	}
//
// # 設定値の妥当性検証
//
// 設定値は自動的に妥当性検証され、無効な値は安全なデフォルト値に
// 修正される。例えば、ウィンドウサイズが320x240未満の場合は
// 最小サイズに修正される。
package config
