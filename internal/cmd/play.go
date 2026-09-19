package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/crashreport"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/maingame"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/steam"
	"github.com/pkg/profile"
	"github.com/urfave/cli/v3"

	_ "net/http/pprof" // pprofのHTTPエンドポイントを登録するためのインポート

	es "github.com/kijimaD/ruins/internal/engine/states"
	gs "github.com/kijimaD/ruins/internal/states"
	w "github.com/kijimaD/ruins/internal/world"
)

// CmdPlay はゲームをプレイするコマンド
var CmdPlay = &cli.Command{
	Name:        "play",
	Usage:       "play",
	Description: "play game",
	Action:      runPlay,
	Flags:       []cli.Flag{},
}

func runPlay(_ context.Context, _ *cli.Command) error {
	// 初期化フェーズの panic をここで受ける。RunGame 中のゲーム panic は MainGame の各コールバックが受ける
	defer crashreport.Guard()

	// Steam APIの初期化。steamタグなしではno-op
	if err := steam.Init(); err != nil {
		return fmt.Errorf("steam initialization failed: %w", err)
	}

	// ユーザー設定ファイルが無ければデフォルト値で作成する。
	// 失敗しても Load はデフォルト値で継続できるため、警告のみで処理を進める
	if err := config.EnsureUserConfigFile(); err != nil {
		logger.New(logger.CategoryLoad).Warn("failed to create user config file", "error", err)
	}

	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// ログ設定を読み込み
	logger.LoadFromConfig(cfg.LogLevel, cfg.LogCategories)

	// デバッグモードの場合は設定を表示
	if cfg.Debug {
		log.Printf("Configuration loaded:\n%s", cfg.String())
	}

	// ウィンドウ設定
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(cfg.User.WindowWidth, cfg.User.WindowHeight)
	ebiten.SetWindowTitle("Coldward")

	// FPS設定
	if cfg.TargetFPS != 60 {
		ebiten.SetTPS(cfg.TargetFPS)
	}

	// プロファイラー設定（WASMは除外）
	if runtime.GOOS != "js" && cfg.DebugPProf {
		var profileOptions []func(*profile.Profile)

		if cfg.ProfileMemory {
			profileOptions = append(profileOptions, profile.MemProfile)
		}
		if cfg.ProfileCPU {
			profileOptions = append(profileOptions, profile.CPUProfile)
		}
		if cfg.ProfileMutex {
			profileOptions = append(profileOptions, profile.MutexProfile)
		}
		if cfg.ProfileTrace {
			profileOptions = append(profileOptions, profile.TraceProfile)
		}

		// デフォルトでメモリプロファイルを有効化
		if len(profileOptions) == 0 {
			profileOptions = append(profileOptions, profile.MemProfile)
		}

		profileOptions = append(profileOptions, profile.ProfilePath(cfg.ProfilePath))
		defer profile.Start(profileOptions...).Stop()

		// pprofサーバー起動
		pprofAddr := fmt.Sprintf("localhost:%d", cfg.PProfPort)
		go func() {
			log.Printf("pprof server starting on http://%s", pprofAddr)
			log.Fatal(http.ListenAndServe(pprofAddr, nil))
		}()
	}

	world, err := maingame.InitWorld(cfg)
	if err != nil {
		return err
	}

	initialState, err := initialPlayState(world, cfg)
	if err != nil {
		return err
	}

	stateMachine, err := es.Init(initialState, world)
	if err != nil {
		return err
	}

	game, err := maingame.NewMainGame(world, stateMachine)
	if err != nil {
		return err
	}

	// クラッシュ諸元へ載せる最上位ステート名の取り出し方を登録する。Save がクラッシュ時に引く。
	// StateMachine は値型で、ゲームループが動かすのは game が持つ実体。登録前のローカル stateMachine を
	// 捕捉するとコピーを掴み遷移を追えないので、game を捕捉して落ちた時点の状態を引く
	crashreport.SetStateProvider(func() string {
		s := game.StateMachine.GetCurrentState()
		if s == nil {
			return ""
		}
		return fmt.Sprintf("%T", s)
	})
	// Linux のタスクバーや Steam Deck は SetWindowTitle でなく X11 の WM_CLASS でアプリを同定する。
	// 既定のままだと "Ebitengine-Application" と表示される。WM_CLASS の class 側は表示・グルーピングに
	// 使われるので表示名の Coldward、instance 側は実行体の同定子なので内部名の ruins にする。
	return ebiten.RunGameWithOptions(game, &ebiten.RunGameOptions{
		X11ClassName:    "Coldward",
		X11InstanceName: "ruins",
	})
}

// initialPlayState は起動時の開始ステートを決める。継続が有効でセーブを読めればその地点から、
// 読めるセーブが無いか継続しない設定ならメインメニューから始める。ロードの失敗は握りつぶさず返す。
func initialPlayState(world w.World, cfg *config.Config) (es.State[w.World], error) {
	if cfg.Continue && cfg.SaveLoadEnabled {
		saveManager, err := save.NewSerializationManager()
		if err != nil {
			return nil, fmt.Errorf("continue: create save manager: %w", err)
		}
		resume, err := gs.ResumeFromLatestSave(world, saveManager)
		if err != nil && !errors.Is(err, gs.ErrNoContinuePoint) {
			return nil, err
		}
		if resume != nil {
			return resume, nil
		}
	}
	return &gs.MainMenuState{}, nil
}
