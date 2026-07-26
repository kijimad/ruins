package interior

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"sort"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// spriteDir はスプライトのソース PNG(32x32)の場所。テストはパッケージ直下で走るのでリポジトリ根へ遡る。
const spriteDir = "../../../assets/file/textures/single/"

// cellPx は1タイルの描画辺長。スプライトが 32x32 なのでセルも 32px にして等倍で置く。
const cellPx = 32

// spriteFileOf は配置の Ref を実スプライトのソース PNG 名へ写す。既存 prop を活かし、無い什器
// (レジ・冷蔵ケース・陳列棚)は同形式のダミーで補った。スプライトの無い装飾は背景描画で表す。
func spriteFileOf(p Placed) string {
	switch p.Ref {
	case "gondola":
		return "prop_gondola_"
	case "register":
		return "prop_register_"
	case "walkin_cooler":
		return "prop_cooler_"
	case "waitchair", "chair":
		return "prop_chair_"
	case "reception", "table":
		return "prop_table_"
	case "exam_bed", "bed":
		return "prop_bed_"
	case "medcabinet", "closet":
		return "prop_bookshelf_"
	case "lantern":
		return "prop_lantern_"
	case "plant":
		return "prop_plant_"
	case "barrel", "crate", "pantry":
		return "prop_barrel_"
	case "snacks", "drinks", "bento", "meds", "bandage":
		return "prop_goods_"
	default:
		return ""
	}
}

// TestGolden_InteriorConvStore は内装生成の目視回帰。文字の模式図では測れない「自然にその施設に見えるか
// (believability)」を、ゲームと同じ 32px スプライトで確認する。レジが入口近く・冷蔵ケースが奥・
// ゴンドラが中央、という「店に見える」配置を人が見て判断する。
func TestGolden_InteriorConvStore(t *testing.T) {
	t.Parallel()

	placed := FillRoom(42, storeRoom(), storeContent())
	require.NotEmpty(t, placed, "何か配置される")
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderRoomSprites(t, storeRoom(), placed))
}

// TestGolden_InteriorConvStoreAged は時間の層を刻んだ最終形の目視回帰。新品の店に Age を掛け、略奪・
// 生活痕・廃墟化を適用した「打ち捨てられた店」を確認する。ruins の実際の見た目はこれで、新品と2枚
// 並べると戦利品が減り・小物が散り・縁が瓦礫化した差分を言葉にできる。
func TestGolden_InteriorConvStoreAged(t *testing.T) {
	t.Parallel()

	placed := Age(42, storeRoom(), FillRoom(42, storeRoom(), storeContent()))
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderRoomSprites(t, storeRoom(), placed))
}

// TestGolden_InteriorClinic は器の汎用性の目視回帰。store と同じ FillRoom に診療所の content を流すだけで、
// 待合が入口・診察ベッドが奥・薬棚が壁、という別の施設が出ることを確認する。幾何と中身の分離、すなわち
// content を器の追加なしで増やせることの実証。
func TestGolden_InteriorClinic(t *testing.T) {
	t.Parallel()

	placed := FillRoom(7, clinicRoom(), clinicContent())
	require.NotEmpty(t, placed, "何か配置される")
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderRoomSprites(t, clinicRoom(), placed))
}

// TestGolden_InteriorHouse は民家の目視回帰。ベッド・机・椅子・棚・ランタンのほぼ既存スプライトで
// 住居が成立し、什器ダミー無しでも自然に見えるかを確認する。
func TestGolden_InteriorHouse(t *testing.T) {
	t.Parallel()

	placed := FillRoom(3, houseRoom(), houseContent())
	require.NotEmpty(t, placed, "何か配置される")
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderRoomSprites(t, houseRoom(), placed))
}

// TestGolden_InteriorHouseBuilding は分割文法まで含めた建物1棟の目視回帰。1部屋では施設全体を評価
// できないため、footprint を BSP で複数部屋へ割り、面積順に居間・寝室・台所・物置の役割 content を
// 流し込み、戸口で連結した1棟を実スプライトで描く。部屋の連なり・扉の位置・部屋ごとの中身の差が、
// 建物として自然に見えるかを人が判断する。
func TestGolden_InteriorHouseBuilding(t *testing.T) {
	t.Parallel()

	const seed = 1
	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	rooms := SplitBuilding(footprint, seed)
	require.GreaterOrEqual(t, len(rooms), 3, "footprint が複数部屋に割れる")

	placed := fillBuilding(seed, rooms)
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderBuildingSprites(t, footprint, rooms, placed))
}

// fillBuilding は建物の各部屋へ面積順で役割 content を割り当て、部屋ごとに FillRoom を回して全配置を
// 集める。大部屋を居間、次を寝室、と割ることで、器の汎用性を建物スケールでも示す。部屋数が役割数を
// 超えたら末尾の物置を繰り返す。
func fillBuilding(seed uint64, rooms []Room) []Placed {
	contents := houseRoleContents()
	all := make([]Placed, 0, len(rooms)*8)
	for rank, ri := range roomOrderByArea(rooms) {
		c := contents[min(rank, len(contents)-1)]
		all = append(all, FillRoom(childSeed(seed, 500+ri), rooms[ri], c)...)
	}
	return all
}

// roomOrderByArea は部屋を面積降順の添字列で返す。役割割り当てで大部屋を居間などに充てるための代理指標。
func roomOrderByArea(rooms []Room) []int {
	idx := make([]int, len(rooms))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ra, rb := rooms[idx[a]].Rect, rooms[idx[b]].Rect
		return ra.W*ra.H > rb.W*rb.H
	})
	return idx
}

// houseRoleContents は民家の部屋役割ごとの content。面積降順に居間・台所・寝室・寝室・物置・玄関ホールを
// 割り当てる。ゾーン分類は entrance からの BFS 距離で決めるのが doc の後続 Stage で、ここでは面積順を
// 暫定の代理指標に使う。部屋が役割数を超えたら末尾の玄関ホールを繰り返す。
func houseRoleContents() []Content {
	bedroom := Content{ID: "bedroom", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "bed", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "lantern", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
		}},
	}}
	return []Content{
		{ID: "living", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "table", Placement: PlaceCenter, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "chair", Placement: PlaceCenter, Amount: Dice{Bonus: 3}},
				{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "lantern", Placement: PlaceWall, Amount: Dice{Bonus: 2}},
			}},
			{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Placement: PlaceFullArea, Amount: Dice{Bonus: 1}}}},
		}},
		{ID: "kitchen", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "table", Placement: PlaceCenter, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "pantry", Placement: PlaceRow, Amount: Dice{Bonus: 3}},
				{Kind: KindFurniture, Ref: "lantern", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
			}},
		}},
		bedroom,
		bedroom,
		{ID: "storage", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "barrel", Placement: PlaceRow, Amount: Dice{Bonus: 5}},
			}},
		}},
		{ID: "entryhall", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "lantern", Placement: PlaceWall, Amount: Dice{Bonus: 2}},
			}},
			{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Placement: PlaceFullArea, Amount: Dice{Bonus: 1}}}},
		}},
	}
}

// renderRoomSprites は1部屋の床・壁・戸口の上に実スプライトを 32px セルへ合成して内装を描く。
func renderRoomSprites(t *testing.T, room Room, placed []Placed) []byte {
	t.Helper()
	r := room.Rect
	img := image.NewRGBA(image.Rect(0, 0, r.W*cellPx, r.H*cellPx))

	for y := range r.H {
		for x := range r.W {
			v := Vec{X: r.X + x, Y: r.Y + y}
			fillCell(img, x*cellPx, y*cellPx, tileColor(r.isPerimeter(v), false))
		}
	}
	for _, d := range room.Doorways {
		fillCell(img, (d.X-r.X)*cellPx, (d.Y-r.Y)*cellPx, tileColor(true, true))
	}

	drawPlaced(t, img, Vec{X: r.X, Y: r.Y}, placed)
	return encodePNG(t, img)
}

// renderBuildingSprites は建物1棟を描く。footprint 全体を走査し、いずれかの部屋の内側なら床、そうで
// なければ壁、戸口なら扉として塗り、その上に実スプライトを合成する。部屋どうしが共有する壁も、その壁
// 上に開けた戸口も、1枚の絵として現れる。
func renderBuildingSprites(t *testing.T, footprint Rect, rooms []Room, placed []Placed) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, footprint.W*cellPx, footprint.H*cellPx))

	floorSet := make(map[Vec]bool)
	doorSet := make(map[Vec]bool)
	for _, rm := range rooms {
		for _, v := range rm.Rect.interiorTiles() {
			floorSet[v] = true
		}
		for _, d := range rm.Doorways {
			doorSet[Vec(d)] = true
		}
	}

	for y := range footprint.H {
		for x := range footprint.W {
			v := Vec{X: footprint.X + x, Y: footprint.Y + y}
			door := doorSet[v]
			fillCell(img, x*cellPx, y*cellPx, tileColor(!floorSet[v] && !door, door))
		}
	}

	drawPlaced(t, img, Vec{X: footprint.X, Y: footprint.Y}, placed)
	return encodePNG(t, img)
}

// drawPlaced は placed を origin を原点とする 32px セルへ描く。スプライトのある什器は実画像を重ね、
// スプライトの無い瓦礫は崩落を示す塗りつぶしで表す。それ以外の装飾は下地の床のままにする。
func drawPlaced(t *testing.T, img *image.RGBA, origin Vec, placed []Placed) {
	t.Helper()
	cache := make(map[string]image.Image)
	for _, p := range placed {
		dx, dy := (p.Pos.X-origin.X)*cellPx, (p.Pos.Y-origin.Y)*cellPx
		name := spriteFileOf(p)
		if name == "" {
			if p.Ref == "rubble" {
				fillCell(img, dx, dy, color.RGBA{R: 92, G: 84, B: 74, A: 255})
			}
			continue
		}
		sp, ok := cache[name]
		if !ok {
			sp = loadSprite(t, name)
			cache[name] = sp
		}
		dp := image.Pt(dx, dy)
		draw.Draw(img, image.Rectangle{Min: dp, Max: dp.Add(image.Pt(cellPx, cellPx))}, sp, image.Point{}, draw.Over)
	}
}

// tileColor は下地セルの色。壁・床・戸口を落ち着いた土色系で分ける。
func tileColor(isWall, isDoor bool) color.RGBA {
	switch {
	case isDoor:
		return color.RGBA{R: 80, G: 150, B: 90, A: 255}
	case isWall:
		return color.RGBA{R: 70, G: 66, B: 60, A: 255}
	default:
		return color.RGBA{R: 40, G: 38, B: 34, A: 255}
	}
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

// fillCell は (x,y) を左上に cellPx 角のセルを c で塗る。下地の床・壁・戸口と瓦礫に使う。
func fillCell(img *image.RGBA, x, y int, c color.RGBA) {
	for yy := y; yy < y+cellPx; yy++ {
		for xx := x; xx < x+cellPx; xx++ {
			img.SetRGBA(xx, yy, c)
		}
	}
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}
