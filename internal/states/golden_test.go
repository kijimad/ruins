package states_test

import (
	"bytes"
	"errors"
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
// テーブル名はダンジョン定義から取得する。
// 目視用の参照画像も隣に置く。画像はピクセル比較せず、文字の地図が golden から変わったときだけ
// 描き直す。比べるのは golden を書き換える前なので、更新の実行で地図と一緒に画像も新しくなる
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

				if imgNeedsUpdate(img.GoldenFileName(t, t.Name()), g.GoldenFileName(t, t.Name()), text) {
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

// snapshotFixedGlyphs は頻出するタイルの文字。ほかの id は出現した id の名前順に
// snapshotGlyphs から割り当てるので、珍しい id が増減しても頻出タイルの文字はずれない
var snapshotFixedGlyphs = map[string]rune{"void": ' ', "floor": '.', "wall": '#', "dirt": ','}

const snapshotGlyphs = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// snapshotText はスナップショットを文字の地図と1行1件の配置で書き出す。
// タイルは id だけを書く。タイルの定義の中身は raw の検査が受け持ち、ここで写すと
// 無関係な raw の変更で全マスに差分が出る
func snapshotText(snap mapplanner.Snapshot, width int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "label: %s\n", snap.Label)

	tileIDs := make([]string, len(snap.Tiles))
	for i := range snap.Tiles {
		tileIDs[i] = snap.Tiles[i].Id
	}
	writeGrid(&b, "tiles", tileIDs, width, snapshotFixedGlyphs)

	if snap.Environments != nil {
		envs := make([]string, len(snap.Environments))
		for i, e := range snap.Environments {
			envs[i] = fmt.Sprintf("%s/%s/%s", e.Shelter, e.Water, e.Foliage)
		}
		writeGrid(&b, "environments shelter/water/foliage", envs, width, nil)
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

// writeGrid は values を width 列の文字の地図として書き、先に凡例を書く。
// 行の両端を | で囲み、空白の文字が行末で削られないようにする
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

// imgNeedsUpdate は画像が存在しないかスナップショットの内容が変わった場合にtrueを返す。
// 末尾の改行の有無で再生成しないよう TrimSpace で比べる
func imgNeedsUpdate(imgPath, textPath string, current []byte) bool {
	if _, err := os.Stat(imgPath); err != nil {
		return true
	}
	golden, err := os.ReadFile(textPath)
	if err != nil {
		return true
	}
	return !bytes.Equal(bytes.TrimSpace(current), bytes.TrimSpace(golden))
}
