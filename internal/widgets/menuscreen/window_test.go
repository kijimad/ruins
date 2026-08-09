package menuscreen

import (
	"fmt"
	"image"
	"os"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/views"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// UIResources のロードや widget.Window の生成が ebiten の実行状態に依存するため必要
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

// collectLabels はwidget.Containerer以下のwidget.Text.Labelを再帰的に集める
func collectLabels(c widget.Containerer) []string {
	container, ok := c.(*widget.Container)
	if !ok || container == nil {
		return nil
	}
	var labels []string
	for _, child := range container.Children() {
		switch v := child.(type) {
		case *widget.Text:
			labels = append(labels, v.Label)
		case *widget.Container:
			labels = append(labels, collectLabels(v)...)
		}
	}
	return labels
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

func TestDetailPageCount_性能行が無いエンティティは1ページになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()

	got := DetailPageCount(world, e)

	assert.Equal(t, 1, got)
}

func TestBuildDetailFromRows_説明は最終ページにだけ表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)
	rows := make([]views.SpecRow, 15)
	for i := range rows {
		rows[i] = views.SpecRow{Label: fmt.Sprintf("項目%02d", i), Value: fmt.Sprintf("%d", i)}
	}

	// widget 生成は ebitenui のグローバル状態に触れるのでロックで直列化する
	var firstPage, lastPage *widget.Window
	vrt.WithUILock(func() {
		firstPage = buildDetailFromRows(world, image.Rect(0, 0, 400, 400), "名前", "説明文", rows, 0)
		lastPage = buildDetailFromRows(world, image.Rect(0, 0, 400, 400), "名前", "説明文", rows, 1)
	})

	require.NotNil(t, firstPage)
	require.NotNil(t, lastPage)
	firstLabels := collectLabels(firstPage.Contents)
	lastLabels := collectLabels(lastPage.Contents)
	assert.NotContains(t, firstLabels, "説明文", "最終ページ以外には説明を出さない")
	assert.Contains(t, lastLabels, "説明文", "最終ページには説明を出す")
	assert.Contains(t, firstLabels, "項目00")
	assert.NotContains(t, firstLabels, "項目14", "1ページ目には収まらない行を出さない")
	assert.Contains(t, lastLabels, "項目14")
	assert.Contains(t, firstLabels, fmt.Sprintf("%s 1/2 %s", consts.IconArrowLeft, consts.IconArrowRight))
	assert.Contains(t, lastLabels, fmt.Sprintf("%s 2/2 %s", consts.IconArrowLeft, consts.IconArrowRight))
}

func TestBuildDetailFromRows_ページ番号は範囲外を先頭と末尾にクランプする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)
	rows := make([]views.SpecRow, 15)
	for i := range rows {
		rows[i] = views.SpecRow{Label: fmt.Sprintf("項目%02d", i), Value: fmt.Sprintf("%d", i)}
	}

	// widget 生成は ebitenui のグローバル状態に触れるのでロックで直列化する
	var negative, overflow *widget.Window
	vrt.WithUILock(func() {
		negative = buildDetailFromRows(world, image.Rect(0, 0, 400, 400), "", "", rows, -1)
		overflow = buildDetailFromRows(world, image.Rect(0, 0, 400, 400), "", "", rows, 99)
	})

	require.NotNil(t, negative)
	require.NotNil(t, overflow)
	assert.Contains(t, collectLabels(negative.Contents), fmt.Sprintf("%s 1/2 %s", consts.IconArrowLeft, consts.IconArrowRight), "負のページは先頭にクランプする")
	assert.Contains(t, collectLabels(overflow.Contents), fmt.Sprintf("%s 2/2 %s", consts.IconArrowLeft, consts.IconArrowRight), "範囲外の大きいページは末尾にクランプする")
}
