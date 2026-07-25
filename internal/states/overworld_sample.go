package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// OverworldSampleState は施設1種の建物候補を格子状に並べてフォグ無しで描く、生成/選別ツール
// 専用のステート。生成＆選別パイプラインの段3(人間の採否)を担う。同じ施設を異なる seed で
// 複数生成し、1枚に並べて見比べて良いものを選ぶために使う。VisionSystem を回さないので
// 全域が明るく見える。
type OverworldSampleState struct {
	es.BaseState[w.World]

	// Facility は生成する施設種別の添字。
	Facility int
	// Cols/Rows は候補格子の列数・行数。Cols*Rows 個の候補を seed 1.. で生成する。
	Cols, Rows int

	mapW, mapH consts.Tile
}

var _ es.State[w.World] = &OverworldSampleState{}

// サンプル1棟のチャンク寸法と、候補どうしの隙間。隙間は void として黒く残り候補を仕切る。
const (
	sampleChunk = 20
	sampleGap   = 2
)

// OnPause は一時停止時の処理。
func (st *OverworldSampleState) OnPause(_ w.World) error { return nil }

// OnResume は再開時の処理。
func (st *OverworldSampleState) OnResume(_ w.World) error { return nil }

// OnStop はエンティティを削除する。
func (st *OverworldSampleState) OnStop(world w.World) error {
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if !world.Components.Player.Has(e) {
			world.ECS.RemoveEntity(e)
		}
	}
	return nil
}

// OnStart は Cols×Rows 個の候補を格子状に生成し、全体を枠取りして全タイルを可視にする。
func (st *OverworldSampleState) OnStart(world w.World) error {
	if st.Cols <= 0 {
		st.Cols = 3
	}
	if st.Rows <= 0 {
		st.Rows = 3
	}
	pitch := consts.Tile(sampleChunk + sampleGap)
	seed := uint64(1)
	for r := range st.Rows {
		for c := range st.Cols {
			ox := consts.Tile(c) * pitch
			oy := consts.Tile(r) * pitch
			if err := overworld.GenerateSampleBuilding(world, st.Facility, seed, sampleChunk, sampleChunk, ox, oy, mapplanner.PlannerTypeOverworldField); err != nil {
				return fmt.Errorf("候補生成に失敗 (seed=%d): %w", seed, err)
			}
			seed++
		}
	}
	st.mapW = consts.Tile(st.Cols) * pitch
	st.mapH = consts.Tile(st.Rows) * pitch

	st.setupCamera(world)
	st.revealAllTiles(world)
	st.hidePlayer(world)
	return nil
}

// Update はツール専用のため何もしない。VisionSystem を回さず可視状態を保つ。
func (st *OverworldSampleState) Update(_ w.World) (es.Transition[w.World], error) {
	return st.ConsumeTransition(), nil
}

// Draw はスプライトを描き、施設名と seed の凡例を添える。
func (st *OverworldSampleState) Draw(world w.World, screen *ebiten.Image) error {
	if sys, ok := world.Renderers[(&gs.RenderSpriteSystem{}).String()]; ok {
		if err := sys.Draw(world, screen); err != nil {
			return err
		}
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("施設: %s  候補 %dx%d  左上から seed=1,2,3...", overworld.FacilitySampleName(st.Facility), st.Cols, st.Rows))
	return nil
}

// setupCamera は候補格子の全体が画面に収まるようカメラを設定する。
func (st *OverworldSampleState) setupCamera(world w.World) {
	tileSize := float64(consts.TileSize)
	mapPixelW := float64(st.mapW) * tileSize
	mapPixelH := float64(st.mapH) * tileSize
	screenW := float64(world.Resources.ScreenDimensions.Width)
	screenH := float64(world.Resources.ScreenDimensions.Height)
	scale := min(screenW/mapPixelW, screenH/mapPixelH) * 0.95

	q := ecs.NewFilter1[gc.Camera](world.ECS).Query()
	for q.Next() {
		camera := world.Components.Camera.Get(q.Entity())
		camera.Scale, camera.ScaleTo = scale, scale
		camera.Pos.X = consts.WorldPixel(mapPixelW / 2)
		camera.Pos.Y = consts.WorldPixel(mapPixelH / 2)
		camera.Target = camera.Pos
	}
}

// revealAllTiles は全タイルを可視にしてフォグを無効化する。
func (st *OverworldSampleState) revealAllTiles(world w.World) {
	vs := query.GetVisionState(world)
	vs.VisibleTiles = make(map[gc.GridElement]bool)
	for y := consts.Tile(0); y < st.mapH; y++ {
		for x := consts.Tile(0); x < st.mapW; x++ {
			vs.VisibleTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}] = true
		}
	}
}

// hidePlayer はプレイヤーを画面外へ退避して描画されないようにする。
func (st *OverworldSampleState) hidePlayer(world w.World) {
	q := ecs.NewFilter2[gc.Player, gc.GridElement](world.ECS).Query()
	for q.Next() {
		ge := world.Components.GridElement.Get(q.Entity())
		ge.X, ge.Y = -100, -100
	}
}
