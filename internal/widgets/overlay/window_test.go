package overlay

import (
	"fmt"
	"image"
	"os"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain はebitenの実行状態に依存するテストのためコンテキスト内で全テストを実行する。
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestDetailPageCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		rowCount int
		want     int
	}{
		{"0行は1ページに丸める", 0, 1},
		{"負数は1ページに丸める", -5, 1},
		{"1ページに収まる12行", 12, 1},
		{"1行超えると2ページになる", 13, 2},
		{"24行は2ページに収まる", 24, 2},
		{"25行は3ページになる", 25, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, detailPageCount(tt.rowCount))
		})
	}
}

// TestDetailPageCount_実体の性能行数からページ数を算出する は、公開の DetailPageCount が
// entityspec.SpecRows の行数を detailPageCount に正しく渡していることを固定する。
func TestDetailPageCount_実体の性能行数からページ数を算出する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	noComponent := world.ECS.NewEntity()
	assert.Equal(t, 1, DetailPageCount(world, noComponent), "性能行が無い実体は1ページに丸める")

	withAbilities := world.ECS.NewEntity()
	world.Components.Abilities.Add(withAbilities, &gc.Abilities{
		Vitality: gc.Ability{Base: 10},
	})
	assert.Equal(t, 1, DetailPageCount(world, withAbilities), "Abilitiesの6行は1ページに収まる")
}

// buildPanelUI は uicore のツリーを組むだけでグローバル状態に触れないので、フェイス無し・
// ロック無しで検証できる。Text は参照するので空の実体を渡す。フェイスが nil なら WrapText は
// 測定せず desc を1行で返す。
func TestBuildPanelUI_説明は最終ページにだけ表示する(t *testing.T) {
	t.Parallel()
	rows := make([]entityspec.SpecRow, 15)
	for i := range rows {
		rows[i] = entityspec.SpecRow{Label: fmt.Sprintf("項目%02d", i), Value: fmt.Sprintf("%d", i)}
	}

	firstPage := buildPanelUI(resources.UIResources{Text: &resources.TextResources{}}, image.Rect(0, 0, 400, 400), DetailContent{Name: "名前", Desc: "説明文", Rows: rows}, 0)
	lastPage := buildPanelUI(resources.UIResources{Text: &resources.TextResources{}}, image.Rect(0, 0, 400, 400), DetailContent{Name: "名前", Desc: "説明文", Rows: rows}, 1)

	firstLabels := uicore.CollectLabels(firstPage)
	lastLabels := uicore.CollectLabels(lastPage)
	assert.NotContains(t, firstLabels, "説明文", "最終ページ以外には説明を出さない")
	assert.Contains(t, lastLabels, "説明文", "最終ページには説明を出す")
	assert.Contains(t, firstLabels, "項目00")
	assert.NotContains(t, firstLabels, "項目14", "1ページ目には収まらない行を出さない")
	assert.Contains(t, lastLabels, "項目14")
	assert.Contains(t, firstLabels, "1/2")
	assert.Contains(t, lastLabels, "2/2")
}

func TestBuildPanelUI_ページ番号は範囲外を先頭と末尾にクランプする(t *testing.T) {
	t.Parallel()
	rows := make([]entityspec.SpecRow, 15)
	for i := range rows {
		rows[i] = entityspec.SpecRow{Label: fmt.Sprintf("項目%02d", i), Value: fmt.Sprintf("%d", i)}
	}

	negative := buildPanelUI(resources.UIResources{Text: &resources.TextResources{}}, image.Rect(0, 0, 400, 400), DetailContent{Rows: rows}, -1)
	overflow := buildPanelUI(resources.UIResources{Text: &resources.TextResources{}}, image.Rect(0, 0, 400, 400), DetailContent{Rows: rows}, 99)

	require.NotNil(t, negative)
	require.NotNil(t, overflow)
	assert.Contains(t, uicore.CollectLabels(negative), "1/2", "負のページは先頭にクランプする")
	assert.Contains(t, uicore.CollectLabels(overflow), "2/2", "範囲外の大きいページは末尾にクランプする")
}
