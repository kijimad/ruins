// Package config はアプリケーションの設定管理を提供する
//
// このパッケージは2種類の設定を扱う。責務が異なるため区別する。
//
//   - 起動設定: 開発者が起動時に環境変数で渡す設定。プロファイル、デバッグ
//     フラグ、プロファイリング設定など。github.com/caarlos0/env/v11 で
//     環境変数から読み込む。永続化しない。
//   - ユーザー設定 (UserConfig): プレイヤーがゲーム内で変更し、次回起動時も
//     保持したい設定。解像度など。設定ファイルに永続化する。将来的に音量や
//     キーバインドを追加する。
//
// どちらも Config 構造体に集約され、実行時の設定の真実は Config 単一である。
// 永続化する対象は UserConfig の境界で表現し、この構造体に含めたフィールド
// だけがファイルへ書き出される。デバッグ用フィールドを混ぜないことで、
// 永続化対象を構造で明示し、設定ファイルへの意図しない漏れを防ぐ。
//
// # ユーザー設定の永続化
//
// ユーザー設定は OS 標準の設定ディレクトリ配下 (Linuxでは
// ~/.config/ruins/settings.toml) に TOML で保存する。Steam の
// インストールディレクトリではなくユーザー領域へ置くことで、整合性チェック
// やアップデートによる消失を避ける。
//
//   - 読み込み: Load() 内で自動的に呼ばれる。ファイルが無ければデフォルトで継続する
//   - 書き込み: オプション画面などから UserConfig を変更した後に SaveUserConfig() を呼ぶ
//
// # 設定値の優先順位
//
// ユーザー設定の値は デフォルト < 設定ファイル < 環境変数 の順に上書きされる。
// 環境変数は開発・デバッグ用の明示的な上書き手段として最優先される。
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
//
// ## デバッグ設定
//   - RUINS_DEBUG: デバッグモード (デフォルト: false)
//   - RUINS_LOG_LEVEL: ログレベル (デフォルト: info) "debug", "info", "warn", "error", "fatal", "ignore"
//   - RUINS_LOG_CATEGORIES: カテゴリ別ログレベル設定 (例: "battle=debug,render=warn")
//   - RUINS_DEBUG_PPROF: pprofサーバー起動 (production: false, development: true)
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
//	width := world.Config.User.WindowWidth
//	height := world.Config.User.WindowHeight
//
//	// デバッグモードの確認
//	if world.Config.Debug {
//		log.Println("Debug mode enabled")
//	}
//
// # 設定値の妥当性検証
//
// Load は最後に Validate を呼んで設定値を検証する。不正な値があれば
// エラーを返し、起動を中止する。値の暗黙的な補正は行わない。
// 例えばウィンドウサイズが320x240未満の場合はエラーになる。
package config
