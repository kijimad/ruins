package states_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
)

// overworldMapSeed は地上の地図の生成の途中経過を撮る seed
const overworldMapSeed = 12345

// overworldMapFixtureDir は地上の地図の途中経過の golden の置き場。マップ生成の途中経過と並べる
const overworldMapFixtureDir = "testdata/TestGolden_MapGenSnapshot"

// TestGolden_OverworldMapSnapshot は地上の地図の生成の全ての段階を、1つの文字の地図で検証する。
// golden はマップ生成の途中経過と同じ置き場に OverworldMap として置く
func TestGolden_OverworldMapSnapshot(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	_, _, cols, rows := dungeon.DungeonOverworld.BandShape()
	spawnRow := gc.SeamlessBand{Rows: rows}.SpawnChunkY()
	snaps := overworld.MapSnapshots(raws, overworldMapSeed, spawnRow, cols, spawnRow)
	require.NotEmpty(t, snaps)
	var b strings.Builder
	for _, snap := range snaps {
		b.WriteString(overworldMapText(snap))
		b.WriteString("\n")
	}
	text := []byte(b.String())
	const name = "OverworldMap"
	g := goldie.New(t, goldie.WithFixtureDir(overworldMapFixtureDir), goldie.WithNameSuffix(".txt"))
	img := goldie.New(t, goldie.WithFixtureDir(overworldMapFixtureDir), goldie.WithNameSuffix(".png"))
	if goldenUpdating() && imgNeedsUpdate(img.GoldenFileName(t, name), g.GoldenFileName(t, name), text) {
		frames := make([][]byte, len(snaps))
		for i := range frames {
			frames[i] = vrt.RenderPNG(t, vrt.States(&gs.OverworldMapVisualizerState{Seed: overworldMapSeed, SnapshotIndex: i}))
		}
		require.NoError(t, img.Update(t, name, phaseStrip(t, frames, 1)))
	}
	g.Assert(t, name, text)
}

// roadLines は道の出入り口の方角の組を罫線の文字にする。北1・南2・東4・西8
var roadLines = map[overworld.RoadDir]rune{
	overworld.RoadN: '╵', overworld.RoadS: '╷', overworld.RoadE: '╶', overworld.RoadW: '╴',
	overworld.RoadN | overworld.RoadS: '│', overworld.RoadE | overworld.RoadW: '─',
	overworld.RoadN | overworld.RoadE: '└', overworld.RoadN | overworld.RoadW: '┘',
	overworld.RoadS | overworld.RoadE: '┌', overworld.RoadS | overworld.RoadW: '┐',
	overworld.RoadN | overworld.RoadS | overworld.RoadE: '├', overworld.RoadN | overworld.RoadS | overworld.RoadW: '┤',
	overworld.RoadN | overworld.RoadE | overworld.RoadW: '┴', overworld.RoadS | overworld.RoadE | overworld.RoadW: '┬',
	overworld.RoadN | overworld.RoadS | overworld.RoadE | overworld.RoadW: '┼',
}

// overworldMapText は段階の俯瞰図を、チャンクの記号の文字の地図にする。道のチャンクは記号の代わりに出入り口の罫線を描く。
// 道のチャンクは他の地物を持たないので、罫線が施設の記号を隠すことはない
func overworldMapText(snap overworld.MapSnapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "label: %s\n", snap.Label)
	for _, row := range snap.View.Cells {
		for _, cell := range row {
			if r, ok := roadLines[cell.Road]; ok {
				b.WriteRune(r)
				continue
			}
			b.WriteRune(cell.Glyph)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func TestOverworldMapVisualizerState_範囲外の段階を開くとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	st := &gs.OverworldMapVisualizerState{Seed: overworldMapSeed, SnapshotIndex: 999}
	require.Error(t, st.OnStart(world))
}

func TestOverworldMapVisualizerState_最初の段階を描ける(t *testing.T) {
	t.Parallel()

	png := vrt.RenderPNG(t, vrt.States(&gs.OverworldMapVisualizerState{Seed: overworldMapSeed}))
	require.NotEmpty(t, png)
}

func TestOverworldMapVisualizerState_地図が画面に収まらなければエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(100, 100)
	st := &gs.OverworldMapVisualizerState{Seed: overworldMapSeed}
	require.Error(t, st.OnStart(world))
}
