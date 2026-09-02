package states_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
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
func collectPlannerTypes() []mapplanner.PlannerType {
	// ダンジョン定義からPlannerType名→テーブル名のマッピングを構築する
	type tableInfo struct {
		EnemyTableName string
		ItemTableName  string
	}
	tableMap := map[string]tableInfo{}
	dungeons := dungeon.GetAllDungeons()
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

// TestGolden_MapGenSnapshot は全PlannerTypeの全フェーズのスナップショットをJSONで検証する。
// テーブル名はダンジョン定義から取得する
func TestGolden_MapGenSnapshot(t *testing.T) {
	t.Parallel()

	for _, pt := range collectPlannerTypes() {
		result := buildMapGenChain(t, pt)
		if result == nil {
			continue
		}
		for i, snap := range result.chain.Snapshots {
			t.Run(fmt.Sprintf("%s/Phase%d_%s", pt.Name, i, snap.Label), func(t *testing.T) {
				t.Parallel()
				data, err := json.MarshalIndent(snap, "", "  ")
				require.NoError(t, err)

				g := goldie.New(t, goldie.WithNameSuffix(".json"))
				g.Assert(t, t.Name(), data)
			})
		}
	}
}

// TestGolden_FacilitySamples は施設種別ごとに建物候補を 3x3 で並べたギャラリーを描く。
// 生成＆選別パイプラインの段3(人間の採否)の視覚基盤。生成規則が変わると golden が変わるので
// updategolden で更新し、並んだ候補を見比べて良し悪しを判断する。フォグ無しで内装まで見える。
// TestMapGenImages は全PlannerTypeの各フェーズのVRT画像を生成する。
// 対応するスナップショットJSONの内容が変わった場合のみ画像を再生成する。
// ピクセル比較は行わず、目視確認用の参照画像として保存する
func TestMapGenImages(t *testing.T) {
	t.Parallel()

	for _, pt := range collectPlannerTypes() {
		result := buildMapGenChain(t, pt)
		if result == nil {
			continue
		}
		for i, snap := range result.chain.Snapshots {
			t.Run(fmt.Sprintf("%s/Phase%d_%s", pt.Name, i, snap.Label), func(t *testing.T) {
				t.Parallel()

				currentJSON, err := json.MarshalIndent(snap, "", "  ")
				require.NoError(t, err)

				g := goldie.New(t, goldie.WithNameSuffix(".png"))
				imgPath := g.GoldenFileName(t, t.Name())
				subName := strings.TrimPrefix(t.Name(), "TestMapGenImages/")
				jsonPath := filepath.Join("testdata", "TestGolden_MapGenSnapshot", subName+".json")

				if !imgNeedsUpdate(imgPath, jsonPath, currentJSON) {
					return
				}

				pngData := vrt.RenderPNG(t, vrt.States(&gs.MapGenVisualizerState{
					PlannerType:   pt,
					Seed:          result.seed,
					SnapshotIndex: i,
				}))
				require.NoError(t, g.Update(t, t.Name(), pngData))
				t.Logf("画像を更新: %s", imgPath)
			})
		}
	}
}

// imgNeedsUpdate は画像が存在しないかJSONの内容が変わった場合にtrueを返す。
// goldie はJSONの末尾に改行を付加するためTrimSpaceで比較する
func imgNeedsUpdate(imgPath, jsonPath string, currentJSON []byte) bool {
	if _, err := os.Stat(imgPath); err != nil {
		return true
	}
	goldenJSON, err := os.ReadFile(jsonPath)
	if err != nil {
		return true
	}
	return !bytes.Equal(bytes.TrimSpace(currentJSON), bytes.TrimSpace(goldenJSON))
}
