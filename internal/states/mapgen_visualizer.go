package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	"github.com/kijimaD/ruins/internal/oapi"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// MapGenVisualizerState はマップ生成過程を可視化するデバッグ用ステート
type MapGenVisualizerState struct {
	es.BaseState[w.World]
	snapshots  []mapplanner.Snapshot
	currentIdx int
	// PlannerType は使用するプランナータイプ
	PlannerType mapplanner.PlannerType
	// Seed は乱数シード
	Seed uint64
	// PhaseIndex はゴールデンテストで特定フェーズを表示する際に使う
	PhaseIndex int

	mapWidth  consts.Tile
	mapHeight consts.Tile
}

func (st MapGenVisualizerState) String() string {
	return "MapGenVisualizer"
}

var _ es.State[w.World] = &MapGenVisualizerState{}

// OnPause は一時停止時の処理
func (st *MapGenVisualizerState) OnPause(_ w.World) error { return nil }

// OnResume は再開時の処理
func (st *MapGenVisualizerState) OnResume(_ w.World) error { return nil }

// OnStart はスナップショットを生成してエンティティをスポーンする
func (st *MapGenVisualizerState) OnStart(world w.World) error {
	st.mapWidth = consts.MapTileWidth
	st.mapHeight = consts.MapTileHeight

	seed := st.Seed
	if seed == 0 {
		seed = world.Config.RNG.Uint64()
	}

	chain, err := mapplanner.BuildChain(world, st.mapWidth, st.mapHeight, seed, st.PlannerType)
	if err != nil {
		return fmt.Errorf("PlannerChain作成失敗: %w", err)
	}
	chain.Recording = true

	if err := chain.Plan(); err != nil {
		return fmt.Errorf("plan実行失敗: %w", err)
	}

	st.snapshots = chain.Snapshots
	if len(st.snapshots) == 0 {
		return fmt.Errorf("スナップショットが記録されていません")
	}

	// PhaseIndexが指定されていればそのフェーズを表示する
	if st.PhaseIndex >= 0 && st.PhaseIndex < len(st.snapshots) {
		st.currentIdx = st.PhaseIndex
	}

	// カメラをマップ全体が見えるように設定する
	st.setupCamera(world)

	// 最初のスナップショットをスポーンする
	return st.spawnSnapshot(world)
}

// OnStop はエンティティを削除する
func (st *MapGenVisualizerState) OnStop(world w.World) error {
	st.clearEntities(world)
	return nil
}

// Update はキー入力を処理する
func (st *MapGenVisualizerState) Update(_ w.World) (es.Transition[w.World], error) {
	// Escapeで閉じる
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	return st.ConsumeTransition(), nil
}

// Draw はスプライトとHUDを描画する
func (st *MapGenVisualizerState) Draw(world w.World, screen *ebiten.Image) error {
	// RenderSpriteSystemで描画する
	if sys, ok := world.Renderers[(&gs.RenderSpriteSystem{}).String()]; ok {
		if err := sys.Draw(world, screen); err != nil {
			return err
		}
	}

	// HUD: フェーズ情報を表示する
	if st.currentIdx < len(st.snapshots) {
		snap := st.snapshots[st.currentIdx]
		info := fmt.Sprintf("Phase %d/%d: %s", st.currentIdx+1, len(st.snapshots), snap.Label)
		ebitenutil.DebugPrint(screen, info)
	}

	return nil
}

// setupCamera はマップ全体が画面に収まるようにカメラを設定する
func (st *MapGenVisualizerState) setupCamera(world w.World) {
	tileSize := float64(consts.TileSize)
	mapPixelW := float64(st.mapWidth) * tileSize
	mapPixelH := float64(st.mapHeight) * tileSize

	screenW := float64(world.Resources.ScreenDimensions.Width)
	screenH := float64(world.Resources.ScreenDimensions.Height)

	// マップ全体が画面に収まるスケールを計算する
	scaleX := screenW / mapPixelW
	scaleY := screenH / mapPixelH
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	// 少し余白を持たせる
	scale *= 0.9

	centerX := mapPixelW / 2
	centerY := mapPixelH / 2

	world.Manager.Join(
		world.Components.Camera,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		camera := world.Components.Camera.Get(entity).(*gc.Camera)
		camera.Scale = scale
		camera.ScaleTo = scale
		camera.X = centerX
		camera.Y = centerY
		camera.TargetX = centerX
		camera.TargetY = centerY
	}))
}

// spawnSnapshot は現在のスナップショットからエンティティを生成する
func (st *MapGenVisualizerState) spawnSnapshot(world w.World) error {
	snap := st.snapshots[st.currentIdx]

	// SnapshotからMetaPlanを再構築する。
	// 未初期化タイル（Name が空）は void として扱う
	tiles := make([]oapi.Tile, len(snap.Tiles))
	copy(tiles, snap.Tiles)
	for i := range tiles {
		if tiles[i].Name == "" {
			tiles[i].Name = "void"
			tiles[i].BlockPass = true
		}
	}

	plan := &mapplanner.MetaPlan{
		Level: gc.Level{
			TileWidth:  st.mapWidth,
			TileHeight: st.mapHeight,
		},
		Tiles:         tiles,
		Rooms:         snap.Rooms,
		Corridors:     snap.Corridors,
		NPCs:          snap.NPCs,
		Items:         snap.Items,
		Props:         snap.Props,
		Doors:         snap.Doors,
		NextPortals:   snap.NextPortals,
		EscapePortals: snap.EscapePortals,
		SpawnPoints:   snap.SpawnPoints,
	}
	if world.Resources != nil {
		plan.RawMaster = &world.Resources.RawMaster
	}

	if _, err := mapspawner.Spawn(world, plan); err != nil {
		return fmt.Errorf("スナップショット%dのスポーン失敗: %w", st.currentIdx, err)
	}

	// 全タイルを可視にする
	st.revealAllTiles(world)

	// プレイヤーを画面外に移動して非表示にする
	st.hidePlayer(world)

	return nil
}

// revealAllTiles は全タイルを可視状態にする
func (st *MapGenVisualizerState) revealAllTiles(world w.World) {
	d := worldhelper.GetDungeon(world)
	d.VisibleTiles = make(map[gc.GridElement]bool)
	for y := consts.Tile(0); y < st.mapHeight; y++ {
		for x := consts.Tile(0); x < st.mapWidth; x++ {
			d.VisibleTiles[gc.GridElement{X: x, Y: y}] = true
		}
	}
}

// hidePlayer はプレイヤーを画面外に移動して描画されないようにする
func (st *MapGenVisualizerState) hidePlayer(world w.World) {
	world.Manager.Join(
		world.Components.Player,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		ge := world.Components.GridElement.Get(entity).(*gc.GridElement)
		ge.X = -100
		ge.Y = -100
	}))
}

// clearEntities はスポーンしたエンティティを削除する
func (st *MapGenVisualizerState) clearEntities(world w.World) {
	world.Manager.Join(
		world.Components.SpriteRender,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if !entity.HasComponent(world.Components.Player) &&
			!entity.HasComponent(world.Components.LocationInBackpack) &&
			!entity.HasComponent(world.Components.LocationEquipped) {
			world.Manager.DeleteEntity(entity)
		}
	}))
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if !entity.HasComponent(world.Components.Player) {
			world.Manager.DeleteEntity(entity)
		}
	}))

	worldhelper.InvalidateSpatialIndex(world)
}
