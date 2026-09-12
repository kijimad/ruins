package hud

import (
	"image/color"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGameInfo(t *testing.T) *GameInfo {
	t.Helper()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	return NewGameInfo(res.Text.BodyFace, res.Text.TitleFontFace, nil)
}

// findText は文字列が一致する最後の描画呼び出しを返す。OutlinedText は縁取りと本体をまとめて描くため、
// 本体だけを探すには末尾から一致するものを拾う
func findText(t *testing.T, texts []textCall, s string) textCall {
	t.Helper()
	for _, text := range slices.Backward(texts) {
		if text.str == s {
			return text
		}
	}
	require.Failf(t, "対象テキストが見つからない", "text %q not found in %v", s, texts)
	return textCall{}
}

func TestGameInfo_drawTemperatureArrow(t *testing.T) {
	t.Parallel()
	info := newTestGameInfo(t)

	t.Run("非表示なら何も描かない", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		info.drawTemperatureArrow(cv, TemperatureArrow{Visible: false, Direction: TempDirectionUp})
		assert.Empty(t, cv.texts)
	})

	tests := []struct {
		name      string
		dir       TempDirection
		wantGlyph string
	}{
		{"上向きは上矢印グリフを使う", TempDirectionUp, consts.IconArrowUp},
		{"下向きは下矢印グリフを使う", TempDirectionDown, consts.IconArrowDown},
		{"一定は右矢印グリフを使う", TempDirectionSteady, consts.IconArrowRight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cv := &fakeCanvas{}
			arrowColor := color.RGBA{R: 1, G: 2, B: 3, A: 4}
			info.drawTemperatureArrow(cv, TemperatureArrow{Visible: true, Direction: tt.dir, Color: arrowColor})

			require.NotEmpty(t, cv.texts)
			last := cv.texts[len(cv.texts)-1]
			assert.Equal(t, tt.wantGlyph, last.str, "本体の文字は方向ごとのグリフになる")
			assert.Equal(t, arrowColor, last.color, "本体の色は矢印の色をそのまま使う")
		})
	}
}

func TestGameInfo_drawAmbientTemperature(t *testing.T) {
	t.Parallel()
	info := newTestGameInfo(t)

	t.Run("非表示なら何も描かない", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		info.drawAmbientTemperature(cv, GameInfoData{AmbientTempVisible: false})
		assert.Empty(t, cv.texts)
	})

	t.Run("表示するとラベルと気温を右寄せで描く", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		tempColor := color.RGBA{R: 10, G: 20, B: 30, A: 255}
		data := GameInfoData{
			AmbientTempVisible:  true,
			AmbientTemp:         25,
			AmbientTempColor:    tempColor,
			AmbientShelterLabel: "屋内",
			MessageAreaHeight:   40,
			ScreenDimensions:    ScreenDimensions{Width: 1024, Height: 768},
		}
		info.drawAmbientTemperature(cv, data)

		label := findText(t, cv.texts, "屋内 ")
		temp := findText(t, cv.texts, "25℃")

		assert.Equal(t, "屋内 ", label.str)
		assert.Equal(t, theme.TextPrimary, label.color)
		assert.Equal(t, "25℃", temp.str)
		assert.Equal(t, tempColor, temp.color)
		assert.Equal(t, label.pos.Y, temp.pos.Y, "ラベルと気温は同じ高さに揃える")
		assert.Greater(t, temp.pos.X, label.pos.X, "気温はラベルの右に置く")
	})
}

func TestGameInfo_drawNorthDepth(t *testing.T) {
	t.Parallel()
	info := newTestGameInfo(t)

	t.Run("非表示なら何も描かない", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		info.drawNorthDepth(cv, GameInfoData{ShowNorthDepth: false})
		assert.Empty(t, cv.texts)
	})

	t.Run("表示するとラベルと奥行きを右下へ描く", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		data := GameInfoData{
			ShowNorthDepth:    true,
			NorthLabel:        "North",
			NorthDepth:        3,
			MessageAreaHeight: 40,
			ScreenDimensions:  ScreenDimensions{Width: 1024, Height: 768},
		}
		info.drawNorthDepth(cv, data)

		txt := findText(t, cv.texts, "North 3")
		assert.Equal(t, "North 3", txt.str)
		assert.Equal(t, theme.TextPrimary, txt.color)
	})
}
