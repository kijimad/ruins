package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// OverworldMapVisualizerState は地上の地図の生成の途中経過を、俯瞰図と同じ描き方で1段階ずつ見せるVRT専用ステート。
// 開始の行を含む計画区間を、地上の定義の列数と開始の行で作り直す
type OverworldMapVisualizerState struct {
	es.BaseState[w.World]
	snapshots []overworld.MapSnapshot
	// Seed は地上の地図の seed
	Seed uint64
	// SnapshotIndex は表示する段階の添字
	SnapshotIndex int
}

var _ es.State[w.World] = &OverworldMapVisualizerState{}

// overworldMapCellPx は1チャンクの辺。計画区間の全ての行が画面に収まる大きさにする
const overworldMapCellPx = 16

// OnPause は一時停止時の処理
func (st *OverworldMapVisualizerState) OnPause(_ w.World) error { return nil }

// OnResume は再開時の処理
func (st *OverworldMapVisualizerState) OnResume(_ w.World) error { return nil }

// OnStart は計画区間の地上の地図を記録しながら作り、表示する段階を確かめる
func (st *OverworldMapVisualizerState) OnStart(world w.World) error {
	_, _, cols, rows := dungeon.DungeonOverworld.BandShape()
	spawnRow := gc.SeamlessBand{Rows: rows}.SpawnChunkY()
	st.snapshots = overworld.MapSnapshots(world.Resources.RawMaster, st.Seed, spawnRow, cols, spawnRow)
	if st.SnapshotIndex < 0 || st.SnapshotIndex >= len(st.snapshots) {
		return fmt.Errorf("SnapshotIndex %d out of range, snapshot count %d", st.SnapshotIndex, len(st.snapshots))
	}
	return nil
}

// OnStop は何もしない。エンティティを生成しない
func (st *OverworldMapVisualizerState) OnStop(_ w.World) error { return nil }

// Update はVRT専用のため何もしない
func (st *OverworldMapVisualizerState) Update(_ w.World) (es.Transition[w.World], error) {
	return st.ConsumeTransition(), nil
}

// Draw は表示する段階を俯瞰図と同じ描き方で左上に描き、段階の名前を出す。段階を横に並べるとき、地図の外の余白を切り詰められる
func (st *OverworldMapVisualizerState) Draw(world w.World, screen *ebiten.Image) error {
	snap := st.snapshots[st.SnapshotIndex]
	hud.DrawMapGrid(uicore.NewEbitenCanvas(screen), snap.View, hud.MapGridStyle{
		OriginX:   4,
		OriginY:   20,
		CellPx:    overworldMapCellPx,
		GlyphFace: world.Resources.UIResources.Text.SmallFace,
	})
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Phase %d/%d: %s", st.SnapshotIndex+1, len(st.snapshots), snap.Label))
	return nil
}
