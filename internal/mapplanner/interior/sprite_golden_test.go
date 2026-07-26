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
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// spriteDir はスプライトのソース PNG(32x32)の場所。テストはパッケージ直下で走るのでリポジトリ根へ遡る。
const spriteDir = "../../../assets/file/textures/single/"

// cellPx は1タイルの描画辺長。スプライトが 32x32 なのでセルも 32px にして等倍で置く。
const cellPx = 32

// roomFixtureDir は部屋単位ゴールデンの置き場。建物単位ゴールデンとは検証の関心が違うので別ディレクトリに
// 分ける。建物は overworld が呼ぶ実経路の believability を、部屋は FillRoom 単体の archetype 配置を見る。
const roomFixtureDir = "testdata/rooms"

// spriteFiles は配置の Ref を実スプライトのソース PNG 名へ写す表。ゲーム本体の豊富な既存スプライトから
// 什器の実物を当てるので、ダミーは使わない。什器ごとに別の絵になり、施設の見分けが付く。表に無い装飾は
// 背景描画で表す。仮画像を避ける置換の理由はコメントに残す。
var spriteFiles = map[string]string{
	// 店。gondola_shelf_ / display_cooler_ はゲーム側が未だ仮画像なので、実描画のある代替を当てる
	"gondola":       "goods_shelf_",
	"register":      "register_",
	"walkin_cooler": "refrigerator_",
	"snacks":        "cookie_",
	"drinks":        "bottled_cola_",
	"bento":         "hamburger_",
	// 診療所。reception_counter_ / exam_bed_ / medicine_cabinet_ は仮画像だが、寝室什器で代用すると
	// 診療所に見えなくなるため、意味の合う実資産をそのまま当てる
	"reception":  "reception_counter_",
	"waitchair":  "bench_",
	"exam_bed":   "exam_bed_",
	"medcabinet": "medicine_cabinet_",
	"meds":       "healing_potion_",
	"bandage":    "leather_bandage_",
	// 依存グラフ machine。施錠された戦利品
	"shutter": "door_vertical_closed_",
	"keycard": "violet_card_",
	// 民家・建物
	"bed":     "bed_",
	"table":   "dining_table_",
	"chair":   "chair_",
	"sofa":    "sofa_",
	"closet":  "closet_",
	"lantern": "tall_lamp_",
	"plant":   "houseplants_",
	"pantry":  "dish_shelf_",
	"barrel":  "wood_barrel_",
	"crate":   "wood_barrel_",
	"bathtub": "bathtub_",
	"toilet":  "toilet_",
	"sink":    "sink_",
	"washer":  "wash_machine_empty_",
	// flavor。廃墟に残る生活の痕
	"candle": "candle_",
	"carpet": "carpet_",
	"broom":  "broom_",
	// Age。時間が刻む崩落と散乱。in-game の raw と同じスプライトを当て、VRT と実装を揃える
	"rubble": "rubble_",
	"debris": "debris_",
}

// spriteFileOf は配置の Ref に対応するスプライト名を返す。表に無ければ空を返し、背景描画に委ねる。
func spriteFileOf(p Placed) string {
	return spriteFiles[p.Ref]
}

// --- 建物単位ゴールデン。overworld が実際に呼ぶ FurnishBuilding をそのまま描く目視回帰。--------------
// PlanHouse など VRT 専用経路でなく in-game で生成される建物そのものを写す。VRT と実装が乖離しないための
// 検証で、実プレイの高コストな確認を VRT で代替する。粗い間仕切りや不自然さはここに現れる。

// TestGolden_BuildingHouse は民家1棟。planRooms が PlanHouseAny で玄関・廊下・水回りの動線を保証し、
// 役割ごとの content で充填したうえで、確率的に Age と Flavor を掛けた最終形を写す。
func TestGolden_BuildingHouse(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	door := Vec{X: 14, Y: 0} // 北壁の入口
	rooms, placed := FurnishBuilding(1, footprint, door, facHouse)
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordStage(t, footprint, rooms, placed))
}

// TestGolden_BuildingStore は店舗1棟。民家と同じ FurnishBuilding に facility を替えて流すだけで、
// 主室に商品棚と冷蔵ケース、奥に備品室という別施設の建物が出ることを写す。
func TestGolden_BuildingStore(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	rooms, placed := FurnishBuilding(2, footprint, door, "store")
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordStage(t, footprint, rooms, placed))
}

// TestGolden_BuildingClinic は診療所1棟に、doc の例そのままの「施錠された薬局奥」を重ねた目視回帰。
// FurnishBuilding で待合と診察室・備品室へ割ったうえに、最奥へ薬を施錠して置き、入口寄りにキーカード、
// 奥の戸口にシャッターを足す。鍵と錠を同じ生成が出すので必ず解ける。
func TestGolden_BuildingClinic(t *testing.T) {
	t.Parallel()

	const seed = 4
	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	rooms, placed := FurnishBuilding(seed, footprint, door, facClinic)
	plain := make([]Room, len(rooms))
	for i, r := range rooms {
		plain[i] = r.Room
	}
	placed = append(placed, guardedLoot(seed, footprint, plain, "meds", 1)...)
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordStage(t, footprint, rooms, placed))
}

// --- 部屋単位ゴールデン。FillRoom 単体の archetype 配置を testdata/rooms/ で見る目視回帰。-------------
// 建物のように動線でなく、1部屋の content が「その施設に見える」配置に解決されるかを確認する。

// TestGolden_RoomConvStore は店の内装。文字の模式図では測れない「自然にその施設に見えるか」を、
// ゲームと同じ 32px スプライトで確認する。レジが入口近く・冷蔵ケースが奥・ゴンドラが中央という配置。
func TestGolden_RoomConvStore(t *testing.T) {
	t.Parallel()

	room := storeRoom()
	placed := FillRoom(42, room, storeContent())
	require.NotEmpty(t, placed, "何か配置される")
	assertRoomGolden(t, room, "store", placed)
}

// TestGolden_RoomClinic は器の汎用性。store と同じ FillRoom に診療所の content を流すだけで、待合が入口・
// 診察ベッドが奥・薬棚が壁、という別の施設が出ることを確認する。幾何と中身の分離の実証。
func TestGolden_RoomClinic(t *testing.T) {
	t.Parallel()

	room := clinicRoom()
	placed := FillRoom(7, room, clinicContent())
	require.NotEmpty(t, placed, "何か配置される")
	assertRoomGolden(t, room, "clinic", placed)
}

// TestGolden_RoomHouse は民家の1室。ベッド・机・椅子・棚・ランタンのほぼ既存スプライトで住居が成立し、
// 什器ダミー無しでも自然に見えるかを確認する。
func TestGolden_RoomHouse(t *testing.T) {
	t.Parallel()

	room := houseRoom()
	placed := FillRoom(3, room, houseContent())
	require.NotEmpty(t, placed, "何か配置される")
	assertRoomGolden(t, room, "house", placed)
}

// TestGolden_RoomFlavor は flavor パスの効き目。ベッドと棚だけの生活痕の薄い部屋に蝋燭の輪を足し、
// 戦利品を増やさず character を与えて空き箱部屋を無くす様子を写す。蝋燭の輪は装飾ゆえ通行を阻まない。
func TestGolden_RoomFlavor(t *testing.T) {
	t.Parallel()

	room := Room{Rect: Rect{X: 0, Y: 0, W: 13, H: 10}, Doorways: []Doorway{{X: 6, Y: 9}}}
	base := Content{ID: "sparse", Groups: []Group{{Style: PickEach, Items: []Stuff{
		{Kind: KindFurniture, Ref: "bed", Amount: Dice{Bonus: 1}},
		{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 2}},
	}}}}
	placed := FillRoom(3, room, base)
	placed = Flavor(3, room, placed, abandonedFlavor())
	assertRoomGolden(t, room, "flavor", placed)
}

// assertRoomGolden は単室を「部屋1個の建物」として recordStage に通し、testdata/rooms/ のゴールデンと
// 照合する。role は部屋の役割ラベルで、部屋ゴールデンでも建物と同じく必ず描く。
func assertRoomGolden(t *testing.T, room Room, role string, placed []Placed) {
	t.Helper()
	rooms := []HouseRoom{{Room: room, Role: role}}
	g := goldie.New(t, goldie.WithFixtureDir(roomFixtureDir), goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordStage(t, room.Rect, rooms, placed))
}

// recordStage は建物も単室も共通で描く記録関数。役割付き部屋の集合から下地タイルを導き、実スプライトを
// 重ね、各部屋に役割ラベルを描く。タイルは「戸口→扉 / 部屋の内側→床 / それ以外→壁」で決める。footprint
// 全体を走査するので、部屋どうしが共有する間仕切りも外周壁も、共有壁に開けた戸口も1枚の絵に現れる。
func recordStage(t *testing.T, footprint Rect, rooms []HouseRoom, placed []Placed) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, footprint.W*cellPx, footprint.H*cellPx))

	floorSet := make(map[Vec]bool)
	doorSet := make(map[Vec]bool)
	for _, hr := range rooms {
		for _, v := range hr.Room.Rect.interiorTiles() {
			floorSet[v] = true
		}
		for _, d := range hr.Room.Doorways {
			doorSet[Vec(d)] = true
		}
	}

	for y := range footprint.H {
		for x := range footprint.W {
			v := Vec{X: footprint.X + x, Y: footprint.Y + y}
			isDoor := doorSet[v]
			isWall := !isDoor && !floorSet[v]
			fillCell(img, x*cellPx, y*cellPx, tileColor(isWall, isDoor))
		}
	}

	drawPlaced(t, img, Vec{X: footprint.X, Y: footprint.Y}, placed)

	// 役割ラベルは什器を描いた後に重ね、家具に埋もれず読めるようにする。左上の床側へ寄せて戸口や壁を避ける
	for _, hr := range rooms {
		lx := (hr.Room.Rect.X + 1 - footprint.X) * cellPx
		ly := (hr.Room.Rect.Y + 1 - footprint.Y) * cellPx
		drawLabel(img, lx+2, ly+2, hr.Role)
	}
	return encodePNG(t, img)
}

// drawLabel は px,py を左上に役割名の文字を、可読性のため暗い下地の上へ描く。basicfont は ASCII のみ
// なので役割 ID は英字に保つ。
func drawLabel(img *image.RGBA, px, py int, s string) {
	w := len(s)*7 + 3
	for yy := py; yy < py+14 && yy < img.Bounds().Dy(); yy++ {
		for xx := px; xx < px+w && xx < img.Bounds().Dx(); xx++ {
			img.SetRGBA(xx, yy, color.RGBA{R: 18, G: 18, B: 22, A: 235})
		}
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 236, G: 224, B: 140, A: 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(px+2, py+11),
	}
	d.DrawString(s)
}

// drawPlaced は placed を origin を原点とする 32px セルへ描く。スプライトのある什器は実画像を重ね、
// 表に無い装飾は下地の床のままにする。
func drawPlaced(t *testing.T, img *image.RGBA, origin Vec, placed []Placed) {
	t.Helper()
	cache := make(map[string]image.Image)
	for _, p := range placed {
		name := spriteFileOf(p)
		if name == "" {
			continue // スプライトの無い装飾は下地に委ねる
		}
		dx, dy := (p.Pos.X-origin.X)*cellPx, (p.Pos.Y-origin.Y)*cellPx
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

// fillCell は (x,y) を左上に cellPx 角のセルを c で塗る。下地の床・壁・戸口に使う。
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
