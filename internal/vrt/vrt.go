package vrt

import (
	"errors"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path"

	"github.com/kijimaD/ruins/internal/config"
	gs "github.com/kijimaD/ruins/internal/systems"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/maingame"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// エラーを返さないと実行終了しないため
var errRegularTermination = errors.New("テスト環境における、想定どおりの終了")

// TestGame はビジュアルリグレッションテスト用のゲーム構造体
type TestGame struct {
	maingame.MainGame
	gameCount  int
	outputPath string
}

// Update はゲームの更新処理を行う
func (g *TestGame) Update() error {
	// テストの前に実行される
	if err := g.StateMachine.Update(g.World); err != nil {
		return err
	}

	// 10フレームだけ実行する。更新→描画の順なので、1度は更新しないと描画されない
	if g.gameCount < 10 {
		g.gameCount++
		return nil
	}

	// エラーを返さないと、実行終了しない
	return errRegularTermination
}

const outputDirName = "vrtimages"
const dirPerm = 0o755

// Draw はゲームの描画処理を行う
func (g *TestGame) Draw(screen *ebiten.Image) {
	g.MainGame.Draw(screen)

	// テストでは保存しない
	if flag.Lookup("test.v") != nil {
		return
	}

	if err := os.Mkdir(outputDirName, dirPerm); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	file, err := os.Create(path.Join(outputDirName, fmt.Sprintf("%s.png", g.outputPath)))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	err = png.Encode(file, screen)
	if err != nil {
		log.Fatal(err)
	}
}

// RunTestGame はテストゲームを実行してスクリーンショットを保存する
// 複数のstateを指定すると、最初のstateを配置した後に残りのstateを順にpushする
func RunTestGame(outputPath string, states ...es.State[w.World]) error {
	if len(states) == 0 {
		return fmt.Errorf("RunTestGame: at least one state is required")
	}

	// VRT用に設定を作成する
	cfg := &config.Config{Profile: config.ProfileTesting}
	cfg.ApplyProfileDefaults()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config.Validate failed: %w", err)
	}

	world, err := maingame.InitWorld(cfg)
	if err != nil {
		return fmt.Errorf("InitWorld failed: %w", err)
	}

	// デバッグデータを初期化
	worldhelper.InitNewGameData(world)

	for _, updater := range []w.Updater{
		&gs.EquipmentChangedSystem{},
		&gs.InventoryChangedSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return err
			}
		}
	}

	// 複数のstateがある場合はラッパーstateを使用
	var state es.State[w.World]
	if len(states) > 1 {
		state = &dummyState{
			states: states,
		}
	} else {
		state = states[0]
	}

	stateMachine, err := es.Init(state, world)
	if err != nil {
		return fmt.Errorf("StateMachine Init failed: %w", err)
	}

	mainGame, err := maingame.NewMainGame(world, stateMachine)
	if err != nil {
		return fmt.Errorf("MainGame Init failed: %w", err)
	}

	g := &TestGame{
		MainGame:   *mainGame,
		gameCount:  0,
		outputPath: outputPath,
	}

	if err := ebiten.RunGame(g); err != nil && err != errRegularTermination {
		return fmt.Errorf("ebiten.RunGame failed: %w", err)
	}

	return nil
}
