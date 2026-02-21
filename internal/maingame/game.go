package maingame

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/loader"
	gr "github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/screeneffect"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
)

// MainGame はebiten.Game interfaceを満たす
type MainGame struct {
	World          w.World
	StateMachine   es.StateMachine[w.World]
	screenPipeline *screeneffect.Pipeline
}

// NewMainGame はMainGameを初期化する
func NewMainGame(world w.World, stateMachine es.StateMachine[w.World]) (*MainGame, error) {
	// オーバーレイ描画フックを設定
	stateMachine.AfterDrawHook = afterDrawHook

	retroFilter, err := screeneffect.NewRetroFilter()
	if err != nil {
		return nil, fmt.Errorf("レトロフィルタの初期化に失敗: %w", err)
	}

	return &MainGame{
		World:          world,
		StateMachine:   stateMachine,
		screenPipeline: screeneffect.NewPipeline(retroFilter),
	}, nil
}

// Layout はinterface methodのため、シグネチャは変更できない
func (game *MainGame) Layout(_, _ int) (int, int) {
	// TODO: 解像度変更は未実装
	return consts.MinGameWidth, consts.MinGameHeight
}

// Update はゲームの更新処理を行う
func (game *MainGame) Update() error {
	// デバッグ表示をトグルする
	if ebiten.IsKeyPressed(ebiten.KeyShift) && inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		// パフォーマンスモニターは攻略に関係ないのでトグルできてよい
		game.World.Config.ShowMonitor = !game.World.Config.ShowMonitor
	}

	if err := game.StateMachine.Update(game.World); err != nil {
		return err
	}

	// ステートが空になったらゲームを終了
	if game.StateMachine.GetStateCount() == 0 {
		return ebiten.Termination
	}

	return nil
}

// Draw はゲームの描画処理を行う
func (game *MainGame) Draw(screen *ebiten.Image) {
	bounds := screen.Bounds()
	offscreen := game.screenPipeline.Begin(bounds.Dx(), bounds.Dy())

	// パイプラインが未設定の場合は直接screenに描画する
	target := offscreen
	if target == nil {
		target = screen
	}

	if err := game.StateMachine.Draw(game.World, target); err != nil {
		log.Fatal(err)
	}

	if game.World.Config.ShowMonitor {
		msg := getPerformanceInfo()
		ebitenutil.DebugPrint(target, msg)
	}

	game.screenPipeline.End(screen)
}

// getPerformanceInfo はパフォーマンス情報を文字列として返す
func getPerformanceInfo() string {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	// 最後のGCからの経過時間を計算
	var lastGCTime string
	if mem.LastGC > 0 {
		lastGC := time.Unix(0, int64(mem.LastGC))
		elapsed := time.Since(lastGC)
		lastGCTime = fmt.Sprintf("%.2fs", elapsed.Seconds())
	} else {
		lastGCTime = "N/A"
	}

	return fmt.Sprintf(`FPS: %.1f
Alloc: %.2fMB
HeapInuse: %.2fMB
StackInuse: %.2fMB
Sys: %.2fMB
NextGC: %.2fMB
TotalAlloc: %.2fMB
Mallocs: %d
Frees: %d
GC: %d
LastGC: %s
PauseTotalNs: %.2fms
Goroutines: %d
`,
		ebiten.ActualFPS(),
		float64(mem.Alloc/1024/1024),      // 現在割り当てられているメモリ
		float64(mem.HeapInuse/1024/1024),  // ヒープで実際に使用中のメモリ
		float64(mem.StackInuse/1024/1024), // スタックで使用中のメモリ
		float64(mem.Sys/1024/1024),        // OSから取得した総メモリ
		float64(mem.NextGC/1024/1024),     // 次回GC実行予定サイズ
		float64(mem.TotalAlloc/1024/1024), // 起動後から割り当てられたヒープオブジェクトの累計バイト数
		mem.Mallocs,                       // 割り当てられたヒープオブジェクトの回数
		mem.Frees,                         // 解放されたヒープオブジェクトの回数
		mem.NumGC,                         // GC実行回数
		lastGCTime,                        // 最後のGC実行からの経過時間
		float64(mem.PauseTotalNs)/1000000, // GC停止時間の累計（ミリ秒）
		runtime.NumGoroutine(),            // 実行中のGoroutine数
	)
}

// InitWorld はゲームワールドを初期化する
func InitWorld(cfg *config.Config) (w.World, error) {
	world, err := w.InitWorld(&gc.Components{})
	if err != nil {
		return w.World{}, err
	}

	world.Config = cfg
	world.Resources.SetScreenDimensions(cfg.WindowWidth, cfg.WindowHeight)

	// ResourceLoaderを使用してリソースを読み込む
	resourceLoader := loader.NewResourceLoader()

	// Load sprite sheets
	spriteSheets, err := resourceLoader.LoadSpriteSheets()
	if err != nil {
		return w.World{}, err
	}
	world.Resources.SpriteSheets = &spriteSheets

	// load fonts
	fonts, err := resourceLoader.LoadFonts()
	if err != nil {
		return w.World{}, err
	}
	world.Resources.Fonts = &fonts

	dougenzakaFont := (*world.Resources.Fonts)["dougenzaka"]

	// サイズ調整
	dougenzaka := &text.GoTextFace{
		Source: dougenzakaFont.FaceSource,
		Size:   16,
	}

	world.Resources.Faces = &map[string]text.Face{
		"dougenzaka": dougenzaka,
	}

	// load UI resources
	uir, err := gr.NewUIResources(dougenzakaFont.FaceSource)
	if err != nil {
		return w.World{}, err
	}
	world.Resources.UIResources = uir

	// load raws
	rw, err := resourceLoader.LoadRaws()
	if err != nil {
		return w.World{}, err
	}
	world.Resources.RawMaster = rw

	gameResource := &gr.Dungeon{
		ExploredTiles:  make(map[gc.GridElement]bool),
		DefinitionName: dungeon.DungeonDebug.Name,
		MinimapSettings: gr.MinimapSettings{
			Width:   150,
			Height:  150,
			OffsetX: 10,
			OffsetY: 10,
			Scale:   3,
		},
	}
	if err := gameResource.RequestStateChange(gr.NoneEvent{}); err != nil {
		return w.World{}, fmt.Errorf("初期化時の状態変更要求エラー: %w", err)
	}
	world.Resources.Dungeon = gameResource

	// initialize systems
	world.Updaters, world.Renderers = gs.InitializeSystems(world)

	return world, nil
}
