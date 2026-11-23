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

// TestGolden_OverworldMapSnapshot は地上の地図の生成の全ての段階を、1つの文字の地図で検証する。マップ生成の TestGolden_MapGenSnapshot と
// 同じく、目視用の参照画像は比較せず、更新の実行で地図が変わったときだけ、全ての段階を横一列に並べた1枚に描き直す
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
	g := goldie.New(t, goldie.WithNameSuffix(".txt"))
	img := goldie.New(t, goldie.WithNameSuffix(".png"))
	if goldenUpdating() && imgNeedsUpdate(img.GoldenFileName(t, t.Name()), g.GoldenFileName(t, t.Name()), text) {
		frames := make([][]byte, len(snaps))
		for i := range frames {
			frames[i] = vrt.RenderPNG(t, vrt.States(&gs.OverworldMapVisualizerState{Seed: overworldMapSeed, SnapshotIndex: i}))
		}
		require.NoError(t, img.Update(t, t.Name(), phaseStrip(t, frames, 1)))
	}
	g.Assert(t, t.Name(), text)
}

// roadLines は道の出入り口の方角の組を罫線の文字にする。北1・南2・東4・西8
var roadLines = map[overworld.RoadDir]rune{
	overworld.RoadN: '╵', overworld.RoadS: '╷', overworld.RoadE: '╶', overworld.RoadW: '╴',
	overworld.RoadN | overworld.RoadS: '│', overworld.RoadE | overworld.RoadW: '─',
	overworld.RoadN | overworld.RoadE: '└', overworld.RoadN | overworld.RoadW: '┘',
	overworld.RoadS | overworld.RoadE: '┌', overworld.RoadS | overworld.RoadW: '┐',
	overworld.RoadN | overworld.RoadS | overworld.RoadE: '├', overworld.RoadN | overworld.RoadS | overworld.RoadW: '┤',
	overworld.RoadE | overworld.RoadW | overworld.RoadN: '┴', overworld.RoadE | overworld.RoadW | overworld.RoadS: '┬',
	overworld.RoadN | overworld.RoadS | overworld.RoadE | overworld.RoadW: '┼',
}

// overworldMapText は段階の俯瞰図を、チャンクの記号の文字の地図にする。道のチャンクは記号の代わりに出入り口の罫線を描く
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
