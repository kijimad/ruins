package interior

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// spriteDir はスプライトのソース PNG(32x32)の場所。テストはパッケージ直下で走るのでリポジトリ根へ遡る。
const spriteDir = "../../../assets/file/textures/single/"

// spriteFileOf は配置の Ref を実スプライトのソース PNG 名へ写す。既存 prop を活かし、無い什器
// (レジ・冷蔵ケース・陳列棚)は同形式のダミーで補った。スプライトの無い装飾・瓦礫は背景描画で表す。
func spriteFileOf(p Placed) string {
	switch p.Ref {
	case "gondola":
		return "prop_gondola_"
	case "register":
		return "prop_register_"
	case "walkin_cooler":
		return "prop_cooler_"
	case "snacks", "drinks", "bento":
		return "prop_goods_"
	default:
		return ""
	}
}

// TestGolden_InteriorConvStoreSprites は実スプライトで描いた内装の目視回帰。文字の模式図では測れない
// 「自然にその施設に見えるか(believability)」を、ゲームと同じ 32px スプライトで確認する。
func TestGolden_InteriorConvStoreSprites(t *testing.T) {
	t.Parallel()

	placed := FillRoom(42, storeRoom(), storeContent())
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderInteriorSprites(t, storeRoom(), placed))
}

// renderInteriorSprites は床・壁・戸口の上に実スプライトを 32px セルへ合成して内装を描く。
func renderInteriorSprites(t *testing.T, room Room, placed []Placed) []byte {
	t.Helper()
	const px = 32
	r := room.Rect
	img := image.NewRGBA(image.Rect(0, 0, r.W*px, r.H*px))

	floor := color.RGBA{R: 40, G: 38, B: 34, A: 255}
	wall := color.RGBA{R: 70, G: 66, B: 60, A: 255}
	door := color.RGBA{R: 80, G: 150, B: 90, A: 255}
	for y := range r.H {
		for x := range r.W {
			c := floor
			if r.isPerimeter(Vec{X: r.X + x, Y: r.Y + y}) {
				c = wall
			}
			fillRect(img, x*px, y*px, px, px, c)
		}
	}
	for _, d := range room.Doorways {
		fillRect(img, (d.X-r.X)*px, (d.Y-r.Y)*px, px, px, door)
	}

	cache := make(map[string]image.Image)
	for _, p := range placed {
		name := spriteFileOf(p)
		if name == "" {
			continue
		}
		sp, ok := cache[name]
		if !ok {
			sp = loadSprite(t, name)
			cache[name] = sp
		}
		dp := image.Pt((p.Pos.X-r.X)*px, (p.Pos.Y-r.Y)*px)
		draw.Draw(img, image.Rectangle{Min: dp, Max: dp.Add(image.Pt(px, px))}, sp, image.Point{}, draw.Over)
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func loadSprite(t *testing.T, name string) image.Image {
	t.Helper()
	f, err := os.Open(spriteDir + name + ".png")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	require.NoError(t, err)
	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			img.SetRGBA(xx, yy, c)
		}
	}
}
