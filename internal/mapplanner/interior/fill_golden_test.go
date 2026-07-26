package interior

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"unicode"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// storeContent はコンビニを模した Content。placement 意味論で「店に見える」配置を宣言する。
// 冷蔵ケースは奥、レジは入口近く、ゴンドラは中央、雑貨は壁際。
func storeContent() Content {
	return Content{
		ID: "conv_store",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "walkin_cooler", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 7}},
				{Kind: KindFurniture, Ref: "register", Placement: PlaceNearDoor, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "gondola", Placement: PlaceRow, Amount: Dice{Bonus: 10}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "snacks", Placement: PlaceCenter, Weight: 3, Amount: Dice{Base: 2, Sides: 4}},
				{Kind: KindLoot, Ref: "drinks", Placement: PlaceFarFromDoor, Weight: 2, Amount: Dice{Base: 1, Sides: 4}},
				{Kind: KindLoot, Ref: "bento", Placement: PlaceWall, Weight: 1, Amount: Dice{Base: 1, Sides: 3}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "litter", Placement: PlaceFullArea, Amount: Dice{Base: 1, Sides: 3, Bonus: 1}},
			}},
		},
	}
}

// storeRoom は入口が下辺中央の 15x10 の部屋。
func storeRoom() Room {
	return Room{
		Rect:     Rect{X: 0, Y: 0, W: 15, H: 10},
		Doorways: []Doorway{{X: 7, Y: 9}},
	}
}

// clinicContent は診療所を模した Content。同じ器に別の content を流すだけで別の施設になることを示す。
// 受付と待合椅子は入口近く、診察ベッドは奥、薬棚は壁際、という診療所の定石を placement で宣言する。
func clinicContent() Content {
	return Content{
		ID: "clinic",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "reception", Placement: PlaceNearDoor, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "waitchair", Placement: PlaceNearDoor, Amount: Dice{Bonus: 5}},
				{Kind: KindFurniture, Ref: "exam_bed", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 3}},
				{Kind: KindFurniture, Ref: "medcabinet", Placement: PlaceWall, Amount: Dice{Bonus: 3}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "meds", Placement: PlaceFarFromDoor, Weight: 2, Amount: Dice{Base: 1, Sides: 3}},
				{Kind: KindLoot, Ref: "bandage", Placement: PlaceWall, Weight: 1, Amount: Dice{Base: 1, Sides: 2}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "plant", Placement: PlaceFullArea, Amount: Dice{Bonus: 2}},
			}},
		},
	}
}

// clinicRoom は入口が下辺中央の 16x11 の部屋。
func clinicRoom() Room {
	return Room{
		Rect:     Rect{X: 0, Y: 0, W: 16, H: 11},
		Doorways: []Doorway{{X: 8, Y: 10}},
	}
}

// TestGolden_InteriorRoomLayout は分割文法の出力(部屋レイアウト)の段を目視する中間段 golden。
// content を流し込む前の、壁と戸口だけの器。以降の段は同じ renderInterior にこの器 + その段までの
// 配置を渡すことで、パイプラインの各段を1枚ずつ VRT で押さえられる。
func TestGolden_InteriorRoomLayout(t *testing.T) {
	t.Parallel()

	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderInterior(storeRoom(), nil))
}

// TestGolden_InteriorConvStore は内装生成の目視回帰。FillRoom の結果を PNG に描き golden 化する。
// レジが入口近く・冷蔵ケースが奥・ゴンドラが中央、という「店に見える」配置を人が2枚並べて確認する。
// placement 規則や密度場を変えると golden が変わるので updategolden で焼き直して見比べる。
func TestGolden_InteriorConvStore(t *testing.T) {
	t.Parallel()

	placed := FillRoom(42, storeRoom(), storeContent())
	require.NotEmpty(t, placed, "何か配置される")

	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderInterior(storeRoom(), placed))
}

// TestGolden_InteriorClinic は器の汎用性の目視回帰。store と同じ FillRoom / renderInterior に診療所の
// content を流すだけで、待合が入口・診察ベッドが奥・薬棚が壁、という別の施設が出ることを確認する。
// 幾何と中身の分離、すなわち content を器の追加なしで増やせることの実証。
func TestGolden_InteriorClinic(t *testing.T) {
	t.Parallel()

	placed := FillRoom(7, clinicRoom(), clinicContent())
	require.NotEmpty(t, placed, "何か配置される")

	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderInterior(clinicRoom(), placed))
}

// TestFillRoom_同じseedで完全一致する は配置まで含めた決定性を固定する。再訪一致と serde の前提。
func TestFillRoom_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	room, content := storeRoom(), storeContent()
	first := FillRoom(42, room, content)
	for range 5 {
		require.Equal(t, first, FillRoom(42, room, content), "同じ seed なら配置も完全一致する")
	}
}

// --- 目視用レンダラ。テスト専用のデバッグ描画で、ゲーム本体には持ち込まない ---

const cellPx = 22

// renderInterior はパイプラインの各段を目視するための共通 VRT プリミティブ。部屋の床・壁・戸口と、
// その段までに置かれた stuff を1文字の記号で PNG に描く。placed に段の途中状態を渡せば、レイアウト
// のみ・保証セットまで・全配置・生活痕後・廃墟化後、と段ごとに1枚ずつ golden で押さえられる。
// 記号は Ref の先頭文字、色は StuffKind。素の Go 描画なので ebiten も表示も要らず決定的。
func renderInterior(room Room, placed []Placed) []byte {
	r := room.Rect
	legendH := 3 * cellPx
	img := image.NewRGBA(image.Rect(0, 0, r.W*cellPx, r.H*cellPx+legendH))

	floor := color.RGBA{R: 28, G: 30, B: 36, A: 255}
	wall := color.RGBA{R: 66, G: 70, B: 82, A: 255}
	door := color.RGBA{R: 84, G: 176, B: 108, A: 255}

	// 床と壁と戸口
	for y := range r.H {
		for x := range r.W {
			cell := floor
			if r.isPerimeter(Vec{X: r.X + x, Y: r.Y + y}) {
				cell = wall
			}
			fillCell(img, x, y, cell)
		}
	}
	for _, d := range room.Doorways {
		fillCell(img, d.X-r.X, d.Y-r.Y, door)
	}

	// 置かれた stuff。セルに種別色の淡い下地を敷き、Ref 先頭文字を種別色で重ねる
	for _, p := range placed {
		cx, cy := p.Pos.X-r.X, p.Pos.Y-r.Y
		kc := kindColor(p.Kind)
		fillCell(img, cx, cy, tint(floor, kc, 0.35))
		drawGlyph(img, cx*cellPx, cy*cellPx, glyphOf(p), kc)
	}

	drawLegend(img, r.H*cellPx)

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func fillCell(img *image.RGBA, cx, cy int, c color.RGBA) {
	for y := cy*cellPx + 1; y < (cy+1)*cellPx; y++ {
		for x := cx*cellPx + 1; x < (cx+1)*cellPx; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// glyphOf は置物の記号。Furniture は Ref 先頭を大文字、他は小文字にして種別も文字で読めるようにする。
func glyphOf(p Placed) rune {
	ch := 'x'
	for _, r := range p.Ref {
		ch = r
		break
	}
	if p.Kind == KindFurniture {
		return unicode.ToUpper(ch)
	}
	return unicode.ToLower(ch)
}

func drawGlyph(img *image.RGBA, px, py int, ch rune, c color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(px+cellPx/2-3, py+cellPx/2+5),
	}
	d.DrawString(string(ch))
}

func drawLegend(img *image.RGBA, top int) {
	items := []struct {
		label string
		c     color.RGBA
	}{
		{"Furniture (UPPER glyph)", kindColor(KindFurniture)},
		{"Loot (lower glyph)", kindColor(KindLoot)},
		{"Being", kindColor(KindBeing)},
		{"Decor", kindColor(KindDecor)},
	}
	y := top + 8
	for _, it := range items {
		for dy := range 12 {
			for dx := range 12 {
				img.SetRGBA(8+dx, y+dy, it.c)
			}
		}
		d := &font.Drawer{Dst: img, Src: image.NewUniform(color.RGBA{R: 210, G: 214, B: 222, A: 255}),
			Face: basicfont.Face7x13, Dot: fixed.P(26, y+11)}
		d.DrawString(it.label)
		y += 20
	}
}

func kindColor(k StuffKind) color.RGBA {
	switch k {
	case KindFurniture:
		return color.RGBA{R: 196, G: 150, B: 96, A: 255}
	case KindLoot:
		return color.RGBA{R: 226, G: 188, B: 66, A: 255}
	case KindBeing:
		return color.RGBA{R: 224, G: 84, B: 84, A: 255}
	case KindDecor:
		return color.RGBA{R: 150, G: 154, B: 166, A: 255}
	case KindTrap:
		return color.RGBA{R: 206, G: 96, B: 206, A: 255}
	}
	return color.RGBA{R: 120, G: 120, B: 120, A: 255}
}

// tint は base へ c を t 比率で混ぜる。
func tint(base, c color.RGBA, t float64) color.RGBA {
	mix := func(a, b uint8) uint8 { return uint8(float64(a)*(1-t) + float64(b)*t) }
	return color.RGBA{R: mix(base.R, c.R), G: mix(base.G, c.G), B: mix(base.B, c.B), A: 255}
}
