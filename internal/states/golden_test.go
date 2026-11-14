package states_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

const mapGenSeed = uint64(12345)

// collectPlannerTypes は全PlannerTypeにダンジョン定義のテーブル名を設定して返す。
// ダンジョン定義に含まれるPlannerTypeにはそのダンジョンのテーブル名を設定し、
// 含まれないもの（テンプレート系など）はテーブルなしでテストする
func collectPlannerTypes(t *testing.T) []mapplanner.PlannerType {
	t.Helper()
	// ダンジョン定義からPlannerType名→テーブル名のマッピングを構築する
	type tableInfo struct {
		EnemyTableName string
		ItemTableName  string
	}
	tableMap := map[string]tableInfo{}
	dungeons := dungeon.GetAllDungeons(testutil.InitTestWorld(t).Resources.RawMaster)
	for i := range dungeons {
		def := dungeons[i]
		for _, pw := range def.PlannerPool() {
			if _, exists := tableMap[pw.PlannerType.Name]; !exists {
				tableMap[pw.PlannerType.Name] = tableInfo{
					EnemyTableName: def.EnemyTableName(),
					ItemTableName:  def.ItemTableName(),
				}
			}
		}
	}

	result := make([]mapplanner.PlannerType, len(mapplanner.AllPlannerTypes))
	copy(result, mapplanner.AllPlannerTypes)
	for i := range result {
		if info, ok := tableMap[result[i].Name]; ok {
			result[i].EnemyTableName = info.EnemyTableName
			result[i].ItemTableName = info.ItemTableName
		}
		result[i].Danger = 1
	}
	return result
}

// mapGenResult はbuildMapGenChainの結果を保持する
type mapGenResult struct {
	chain *mapplanner.PlannerChain
	seed  uint64
}

// buildMapGenChain はBuildChainを使ってRecording付きチェーンを構築する。
// 実装と同じチェーン構築ロジックを共有する。
// 接続性エラー時は本番と同様にシードを変えてリトライする。
// アセット未作成などでチェーン構築に失敗した場合はスキップしてnilを返す
func buildMapGenChain(t *testing.T, pt mapplanner.PlannerType) *mapGenResult {
	t.Helper()
	world := vrt.InitReplayWorld(t)
	for attempt := range mapplanner.MaxPlanRetries {
		currentSeed := mapGenSeed + uint64(attempt*1000)
		chain, err := mapplanner.BuildChain(world, consts.MapTileWidth, consts.MapTileHeight, currentSeed, pt)
		if err != nil {
			t.Skipf("PlannerType %s のチェーン構築をスキップ: %v", pt.Name, err)
			return nil
		}
		chain.Recording = true
		if err := chain.Plan(); err != nil {
			if errors.Is(err, mapplanner.ErrConnectivity) {
				continue
			}
			t.Skipf("PlannerType %s のプラン生成をスキップ: %v", pt.Name, err)
			return nil
		}
		return &mapGenResult{chain: chain, seed: currentSeed}
	}
	t.Skipf("PlannerType %s のプラン生成が%d回失敗しました", pt.Name, mapplanner.MaxPlanRetries)
	return nil
}

// TestGolden_MapGenSnapshot は全PlannerTypeの全フェーズのスナップショットを文字の地図で検証する。
// 目視用の参照画像は比較せず、更新の実行で地図が変わったときだけ描き直す。
// 変わったかは golden を書き換える前に比べる
func TestGolden_MapGenSnapshot(t *testing.T) {
	t.Parallel()

	for _, pt := range collectPlannerTypes(t) {
		result := buildMapGenChain(t, pt)
		if result == nil {
			continue
		}
		width := int(result.chain.PlanData.Level.TileWidth)
		for i, snap := range result.chain.Snapshots {
			t.Run(fmt.Sprintf("%s/Phase%d_%s", pt.Name, i, snap.Label), func(t *testing.T) {
				t.Parallel()
				text := []byte(snapshotText(snap, width))
				g := goldie.New(t, goldie.WithNameSuffix(".txt"))
				img := goldie.New(t, goldie.WithNameSuffix(".png"))

				if goldenUpdating() && imgNeedsUpdate(img.GoldenFileName(t, t.Name()), g.GoldenFileName(t, t.Name()), text) {
					pngData := vrt.RenderPNG(t, vrt.States(&gs.MapGenVisualizerState{
						PlannerType:   pt,
						Seed:          result.seed,
						SnapshotIndex: i,
					}))
					require.NoError(t, img.Update(t, t.Name(), pngData))
				}
				g.Assert(t, t.Name(), text)
			})
		}
	}
}

func TestMapGenVisualizerState_空のタイルと収納の中身を描ける(t *testing.T) {
	t.Parallel()

	tests := []struct {
		planner string
		label   string
	}{
		{"Small Room", "Initial"},
		{"Small Town", "TemplatePlanner"},
	}
	plannerTypes := collectPlannerTypes(t)
	for _, tt := range tests {
		t.Run(tt.planner+"/"+tt.label, func(t *testing.T) {
			t.Parallel()
			i := slices.IndexFunc(plannerTypes, func(pt mapplanner.PlannerType) bool { return pt.Name == tt.planner })
			require.GreaterOrEqual(t, i, 0)
			pt := plannerTypes[i]
			result := buildMapGenChain(t, pt)
			require.NotNil(t, result)
			index := slices.IndexFunc(result.chain.Snapshots, func(s mapplanner.Snapshot) bool { return s.Label == tt.label })
			require.GreaterOrEqual(t, index, 0)

			png := vrt.RenderPNG(t, vrt.States(&gs.MapGenVisualizerState{
				PlannerType:   pt,
				Seed:          result.seed,
				SnapshotIndex: index,
			}))
			require.NotEmpty(t, png)
		})
	}
}

func TestWriteGrid_固定の文字を先に当て残りを名前順に割り当てる(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	writeGrid(&b, "tiles", []string{"wall", "road", "floor", "", "grass", "wall"}, 3, snapshotTileGlyphs)

	require.Equal(t, `tiles:
  '?' = ""
  '.' = "floor"
  'a' = "grass"
  'b' = "road"
  '#' = "wall"
|#b.|
|?a#|
`, b.String())
}

// goldenUpdating は -update か GOLDIE_UPDATE を付けた実行かを返す
func goldenUpdating() bool {
	f := flag.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

// snapshotTileGlyphs と snapshotEnvGlyphs は頻出する値の文字。残りは snapshotGlyphs から名前順に割り当てる
var (
	snapshotTileGlyphs = map[string]rune{"": '?', "void": ' ', "floor": '.', "wall": '#', "dirt": ','}
	snapshotEnvGlyphs  = map[string]rune{"indoor//": 'i', "outdoor//": 'o', "partial//": 'p'}
)

const snapshotGlyphs = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// snapshotText はスナップショットを文字の地図と1行1件の配置で書き出す
func snapshotText(snap mapplanner.Snapshot, width int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "label: %s\n", snap.Label)

	tileIDs := make([]string, len(snap.Tiles))
	for i := range snap.Tiles {
		tileIDs[i] = snap.Tiles[i].Id
	}
	writeGrid(&b, "tiles", tileIDs, width, snapshotTileGlyphs)

	if snap.Environments != nil {
		envs := make([]string, len(snap.Environments))
		for i, e := range snap.Environments {
			envs[i] = fmt.Sprintf("%s/%s/%s", e.Shelter, e.Water, e.Foliage)
		}
		writeGrid(&b, "environments shelter/water/foliage", envs, width, snapshotEnvGlyphs)
	}

	for _, r := range snap.Rooms {
		fmt.Fprintf(&b, "room: %d,%d-%d,%d\n", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
	}
	for _, c := range snap.Corridors {
		cells := make([]string, len(c))
		for i, idx := range c {
			cells[i] = fmt.Sprintf("%d,%d", int(idx)%width, int(idx)/width)
		}
		fmt.Fprintf(&b, "corridor: %s\n", strings.Join(cells, " "))
	}
	for _, d := range snap.Doors {
		fmt.Fprintf(&b, "door: %d,%d orientation=%d\n", d.X, d.Y, d.Orientation)
	}
	for _, p := range snap.NextPortals {
		fmt.Fprintf(&b, "portal: %d,%d\n", p.X, p.Y)
	}
	for _, p := range snap.SpawnPoints {
		fmt.Fprintf(&b, "spawn: %d,%d\n", p.X, p.Y)
	}
	for _, n := range snap.NPCs {
		fmt.Fprintf(&b, "npc: %d,%d %s\n", n.X, n.Y, n.Name)
	}
	for _, it := range snap.Items {
		fmt.Fprintf(&b, "item: %d,%d %s x%d\n", it.X, it.Y, it.Name, it.Count)
	}
	for _, p := range snap.Props {
		fmt.Fprintf(&b, "prop: %d,%d %s\n", p.X, p.Y, p.Name)
	}
	return b.String()
}

// writeGrid は凡例と width 列の文字の地図を書く。両端の | は行末の空白が削られないようにするため
func writeGrid(b *strings.Builder, name string, values []string, width int, fixed map[string]rune) {
	distinct := map[string]bool{}
	for _, v := range values {
		distinct[v] = true
	}
	keys := make([]string, 0, len(distinct))
	for v := range distinct {
		keys = append(keys, v)
	}
	slices.Sort(keys)

	glyph := map[string]rune{}
	used := map[rune]bool{}
	for _, v := range keys {
		if r, ok := fixed[v]; ok {
			glyph[v], used[r] = r, true
		}
	}
	next := []rune(snapshotGlyphs)
	for _, v := range keys {
		if _, ok := glyph[v]; ok {
			continue
		}
		for used[next[0]] {
			next = next[1:]
		}
		glyph[v], used[next[0]] = next[0], true
	}

	fmt.Fprintf(b, "%s:\n", name)
	for _, v := range keys {
		fmt.Fprintf(b, "  %q = %q\n", glyph[v], v)
	}
	for y := 0; y*width < len(values); y++ {
		row := values[y*width : min((y+1)*width, len(values))]
		b.WriteByte('|')
		for _, v := range row {
			b.WriteRune(glyph[v])
		}
		b.WriteString("|\n")
	}
}

// imgNeedsUpdate は画像が無いか、地図が golden と違うかを返す
func imgNeedsUpdate(imgPath, textPath string, current []byte) bool {
	if _, err := os.Stat(imgPath); err != nil {
		return true
	}
	golden, err := os.ReadFile(textPath)
	if err != nil {
		return true
	}
	return !bytes.Equal(current, golden)
}
