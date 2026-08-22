package menuframe

import (
	"testing"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNewPanelScreen_タイトルとフッターを含める(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)

	var ui *ebitenui.UI
	vrt.WithUILock(func() {
		ui = NewPanelScreen(res, "タイトル", styled.NewVerticalContainer(), "フッター")
	})

	require.NotNil(t, ui)
	labels := collectLabels(ui.Container)
	assert.Contains(t, labels, "タイトル")
	assert.Contains(t, labels, "フッター")
}

func TestNewPanelScreen_タイトルとフッターが空なら行を置かない(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)

	var ui *ebitenui.UI
	vrt.WithUILock(func() {
		ui = NewPanelScreen(res, "", styled.NewVerticalContainer(), "")
	})

	require.NotNil(t, ui)
	assert.Empty(t, collectLabels(ui.Container))
}

func TestNewTabScreen(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)

	tests := []struct {
		name       string
		screen     TabScreen
		wantLabels []string
		notLabels  []string
	}{
		{
			name: "見出しとタブ帯とフッターをすべて含む",
			screen: TabScreen{
				Header:    "見出し",
				TabLabels: []string{"タブA", "タブB"},
				TabIndex:  0,
				Content:   styled.NewVerticalContainer(),
				Footer:    "フッター",
			},
			wantLabels: []string{"見出し", "タブA", "タブB", "フッター"},
		},
		{
			name: "見出しとタブ帯が空なら行を置かない",
			screen: TabScreen{
				Content: styled.NewVerticalContainer(),
				Footer:  "フッター",
			},
			wantLabels: []string{"フッター"},
			notLabels:  []string{"見出し"},
		},
		{
			name: "フッターが空なら行を置かない",
			screen: TabScreen{
				Header:  "見出し",
				Content: styled.NewVerticalContainer(),
			},
			wantLabels: []string{"見出し"},
			notLabels:  []string{"フッター"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ui *ebitenui.UI
			vrt.WithUILock(func() {
				ui = NewTabScreen(res, tt.screen)
			})

			require.NotNil(t, ui)
			labels := collectLabels(ui.Container)
			for _, want := range tt.wantLabels {
				assert.Contains(t, labels, want)
			}
			for _, not := range tt.notLabels {
				assert.NotContains(t, labels, not)
			}
		})
	}
}
