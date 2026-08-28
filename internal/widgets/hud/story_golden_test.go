// Package hud_test は HUD 部品のカタログ。部品を単体で撮り、意匠と幾何をピクセルで固定する。
//
// 画面まるごとの golden は HUD も一緒に写すが、変化があったとき部品が変わったのか
// 画面の使い方が変わったのかを切り分けられない。ここで部品だけを撮っておくと切り分く。
//
// 外部テストパッケージにするのは、撮影を担う vrt が systems を経由して hud を取り込むため。
// 同じパッケージからは循環になる。
package hud_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/hud"
)

func TestMain(m *testing.M) {
	vrt.RunTestMain(m)
}

// storyRes はカタログ用のフォントと素材を読む。world は要らないので loader から直に読む。
func storyRes(t *testing.T) resources.UIResources {
	t.Helper()
	fonts, err := loader.LoadFonts()
	require.NoError(t, err)
	res, err := loader.LoadUIResources(fonts)
	require.NoError(t, err)
	return res
}

// storyScreen はカタログの撮影範囲。HUD は画面全体を基準に位置を決めるので、
// 実際の論理解像度で撮る。
var storyScreen = hud.ScreenDimensions{Width: consts.GameWidth, Height: consts.GameHeight}

func TestGolden_Story_GameInfoGauges(t *testing.T) {
	t.Parallel()
	res := storyRes(t)
	vrt.AssertScreenGolden(t, func() func(screen *ebiten.Image) {
		info := hud.NewGameInfo(res.Text.SmallFace, res.Text.TitleFontFace, res.GaugeFill)
		return func(screen *ebiten.Image) {
			info.Draw(screen, hud.GameInfoData{
				FloorNumber:       3,
				PlayerHP:          42,
				PlayerMaxHP:       80,
				PlayerWeight:      consts.Milligram(4_400_000),
				PlayerMaxWeight:   consts.Milligram(21_000_000),
				BodyTempRatio:     0.3,
				BodyTempVisible:   true,
				MessageAreaHeight: hud.DefaultMessageAreaConfig.Height(),
				ScreenDimensions:  storyScreen,
			})
		}
	}, storyScreen.Width, storyScreen.Height)
}

func TestGolden_Story_StatusBadges(t *testing.T) {
	t.Parallel()
	res := storyRes(t)
	vrt.AssertScreenGolden(t, func() func(screen *ebiten.Image) {
		badges := hud.NewStatusBadges(res.Text.SmallFace)
		return func(screen *ebiten.Image) {
			badges.Draw(screen, hud.StatusBadgesData{
				Badges: []hud.StatusBadge{
					{Text: "Hungry", Color: color.RGBA{R: 150, G: 90, B: 30, A: 255}},
					{Text: "Cold", Color: color.RGBA{R: 40, G: 90, B: 160, A: 255}},
					{Text: "Bleeding", Color: color.RGBA{R: 160, G: 40, B: 40, A: 255}},
				},
				MessageAreaHeight: hud.DefaultMessageAreaConfig.Height(),
				ScreenDimensions:  storyScreen,
			})
		}
	}, storyScreen.Width, storyScreen.Height)
}

func TestGolden_Story_Currency(t *testing.T) {
	t.Parallel()
	res := storyRes(t)
	vrt.AssertScreenGolden(t, func() func(screen *ebiten.Image) {
		cur := hud.NewCurrencyDisplay(res.Text.SmallFace)
		return func(screen *ebiten.Image) {
			cur.Draw(screen, hud.CurrencyData{
				Currency:         consts.Currency(10000),
				ScreenDimensions: storyScreen,
				Config:           hud.DefaultMessageAreaConfig,
			})
		}
	}, storyScreen.Width, storyScreen.Height)
}
