package interior

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
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
	// 事務所・受付。desk は raw の spriteKey wood_desk に合わせる
	"desk": "wood_desk_",
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

// TestGolden_BuildingHouse は民家1棟を9 seed 並べた目視回帰。planRooms が PlanHouseAny で玄関・廊下・
// 水回りの動線を保証し、役割ごとの content で充填したうえで、確率的に Age と Flavor を掛けた最終形を写す。
// 1枚では見えない間取りの変種、横型と縦型・部屋数・経年の有無の幅を、9マスの並びで一望する。
func TestGolden_BuildingHouse(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	door := Vec{X: 14, Y: 0} // 北壁の入口
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordSeeds(t, func(seed uint64) (Site, []Placed) {
		return FurnishBuilding(seed, footprint, door, facHouse)
	}))
}

// TestGolden_BuildingStore は店舗1棟を9 seed 並べた目視回帰。民家と同じ FurnishBuilding に facility を
// 替えて流すだけで、主室に商品棚と冷蔵ケース、奥に備品室という別施設の建物が出ることを写す。variant 抽選で
// コンビニ・薬局・八百屋のどれになるかの幅も並びに現れる。
func TestGolden_BuildingStore(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordSeeds(t, func(seed uint64) (Site, []Placed) {
		return FurnishBuilding(seed, footprint, door, "store")
	}))
}

// TestGolden_BuildingClinic は診療所1棟に、doc の例そのままの「施錠された薬局奥」を重ねた9 seed の目視回帰。
// FurnishBuilding で待合と診察室・備品室へ割ったうえに、最奥へ薬を施錠して置き、入口寄りにキーカード、
// 奥の戸口にシャッターを足す。鍵と錠を同じ生成が出すので必ず解ける。seed ごとに錠と鍵の位置が動く幅を見る。
func TestGolden_BuildingClinic(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordSeeds(t, func(seed uint64) (Site, []Placed) {
		site, placed := FurnishBuilding(seed, footprint, door, facClinic)
		plain := make([]Room, len(site.Rooms))
		for i, r := range site.Rooms {
			plain[i] = r.Room
		}
		return site, append(placed, guardedLoot(seed, plain, "meds", 1)...)
	}))
}

// --- 加工段別ゴールデン。1つの建物を plan→fill→age→flavor の各段で並べ、どの加工が犯人かを切り分ける。--
// recordSeeds が最終形の変種を見るのに対し、こちらは1 seed の加工の流れを追う。mapplanner の段別
// スナップショット表示に相当する。seed は経年が乗る個体を選び、age 段で差が出るようにする。

// TestGolden_StagesHouse は民家1棟の加工の流れ。空の間取り(plan)に什器(fill)、経年の略奪と崩落(age)、
// 生活痕の装飾(flavor)が順に乗る様子を並べる。たとえば age で家具が消えすぎる退行は、fill 段と見比べれば
// どの段の仕業か一目で分かる。
func TestGolden_StagesHouse(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	door := Vec{X: 14, Y: 0}
	site, stages := FurnishStages(1, footprint, door, facHouse)
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordStages(t, site, stages))
}

// TestGolden_StagesStore は店舗1棟の加工の流れ。商品棚と冷蔵ケースの fill、略奪で戦利品が減る age、
// 散らばった蝋燭や絨毯の flavor を段ごとに追う。
func TestGolden_StagesStore(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	site, stages := FurnishStages(1, footprint, door, "store")
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordStages(t, site, stages))
}

// TestGolden_StagesClinic は診療所1棟の加工の流れ。診察台と薬棚の fill から、経年と装飾までを追う。
// guardedLoot の施錠戦利品は別の machine なのでここには含めず、加工パイプラインだけを写す。
func TestGolden_StagesClinic(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	door := Vec{X: 13, Y: 0}
	site, stages := FurnishStages(1, footprint, door, facClinic)
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordStages(t, site, stages))
}

// --- 部屋単位ゴールデン。FillRoom 単体の archetype 配置を testdata/rooms/ で見る目視回帰。-------------
// 建物のように動線でなく、1部屋の content が「その施設に見える」配置に解決されるかを確認する。

// TestGolden_RoomConvStore は店の内装を9 seed 並べた目視回帰。文字の模式図では測れない「自然にその施設に
// 見えるか」を、ゲームと同じ 32px スプライトで確認する。レジが入口近く・冷蔵ケースが奥・ゴンドラが中央
// という配置の傾向が、seed をまたいで安定するかを一望する。
func TestGolden_RoomConvStore(t *testing.T) {
	t.Parallel()

	room := storeRoom()
	assertRoomGolden(t, room, "store", func(seed uint64) []Placed {
		return FillRoom(seed, room, storeContent())
	})
}

// TestGolden_RoomClinic は器の汎用性を9 seed で写す。store と同じ FillRoom に診療所の content を流すだけで、
// 待合が入口・診察ベッドが奥・薬棚が壁、という別の施設が出ることを確認する。幾何と中身の分離の実証。
func TestGolden_RoomClinic(t *testing.T) {
	t.Parallel()

	room := clinicRoom()
	assertRoomGolden(t, room, "clinic", func(seed uint64) []Placed {
		return FillRoom(seed, room, clinicContent())
	})
}

// TestGolden_RoomHouse は民家の1室を9 seed で写す。ベッド・机・椅子・棚・ランタンのほぼ既存スプライトで
// 住居が成立し、什器ダミー無しでも自然に見えるかを確認する。
func TestGolden_RoomHouse(t *testing.T) {
	t.Parallel()

	room := houseRoom()
	assertRoomGolden(t, room, "house", func(seed uint64) []Placed {
		return FillRoom(seed, room, houseContent())
	})
}

// TestGolden_RoomFlavor は flavor パスの効き目を9 seed で写す。ベッドと棚だけの生活痕の薄い部屋へ絨毯・
// 箒・散らばった蝋燭を足し、戦利品を増やさず character を与えて空き箱部屋を無くす様子を確認する。flavor は
// 装飾ゆえ通行を阻まない。seed ごとに何がどこへ来るかの幅を見る。
func TestGolden_RoomFlavor(t *testing.T) {
	t.Parallel()

	room := Room{Rect: Rect{X: 0, Y: 0, W: 13, H: 10}, Doorways: []Doorway{{X: 6, Y: 9}}}
	base := Content{ID: "sparse", Groups: []Group{{Style: PickEach, Items: []Stuff{
		{Kind: KindFurniture, Ref: "bed", Amount: Dice{Bonus: 1}},
		{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 2}},
	}}}}
	assertRoomGolden(t, room, "flavor", func(seed uint64) []Placed {
		return Flavor(seed, room, FillRoom(seed, room, base), abandonedFlavor())
	})
}

// assertRoomGolden は単室を「部屋1個の建物」として9 seed 分 recordSeeds に通し、testdata/rooms/ のゴールデンと
// 照合する。role は部屋の役割ラベルで、部屋ゴールデンでも建物と同じく必ず描く。fill は seed から配置を返す。
// 単室は敷地計画を持たないので庭・ポーチの無い素の Site に包む。
func assertRoomGolden(t *testing.T, room Room, role string, fill func(seed uint64) []Placed) {
	t.Helper()
	g := goldie.New(t, goldie.WithFixtureDir(roomFixtureDir), goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), recordSeeds(t, func(seed uint64) (Site, []Placed) {
		return singleRoomSite(room, role), fill(seed)
	}))
}

// singleRoomSite は1部屋を庭・ポーチの無い素の Site に包む。部屋ゴールデンが建物ゴールデンと同じ renderStage
// を共有するための器。footprint と建物を部屋矩形に一致させ、庭を空にする。
func singleRoomSite(room Room, role string) Site {
	door := Vec{}
	if len(room.Doorways) > 0 {
		door = Vec(room.Doorways[0])
	}
	return Site{
		Footprint: room.Rect, Building: room.Rect,
		Garden: map[Vec]bool{}, ExtraWall: map[Vec]bool{},
		Door: door, Rooms: []PlannedRoom{{Room: room, Role: role}},
	}
}

// recordSeeds は seed を 1..9 と変えた9枚を 3x3 のモンタージュに合成する記録関数。1枚では見えない生成の
// 変種を、9マスの並びで一望する。各セルは gen が返す Site と配置を renderStage で描く。全 seed で footprint が
// 同じなのでセルが整列する。
func recordSeeds(t *testing.T, gen func(seed uint64) (Site, []Placed)) []byte {
	t.Helper()
	labels := make([]string, 9)
	cells := make([]*image.RGBA, 9)
	var cellW, cellH int
	for i := range cells {
		seed := uint64(i + 1)
		site, placed := gen(seed)
		labels[i] = "seed " + strconv.FormatUint(seed, 10)
		cells[i] = renderStage(t, site, placed)
		cellW, cellH = site.Footprint.W*cellPx, site.Footprint.H*cellPx
	}
	return montage(t, cellW, cellH, 3, labels, cells)
}

// recordStages は1つの建物を加工ステップごとに横一列へ並べる記録関数。plan→fill→age→flavor の各段を
// 同じ Site の上に描き、どの段で見た目が壊れたかを切り分けられるようにする。mapplanner の段別スナップショット
// 表示に倣った VRT で、最終形だけを見る recordSeeds では特定できない「どの加工が犯人か」に答える。
func recordStages(t *testing.T, site Site, stages []FurnishStage) []byte {
	t.Helper()
	labels := make([]string, len(stages))
	cells := make([]*image.RGBA, len(stages))
	for i, s := range stages {
		labels[i] = s.Label
		cells[i] = renderStage(t, site, s.Placed)
	}
	return montage(t, site.Footprint.W*cellPx, site.Footprint.H*cellPx, len(stages), labels, cells)
}

// montage は同じ大きさのセル画像を cols 列のグリッドへ並べ、各セルの上帯にラベルを添えて1枚に合成する。
// seed 別の変種一覧も加工段別の切り分けも、この1つの並べ方を共有する。セルは全て cellW x cellH で整列する。
func montage(t *testing.T, cellW, cellH, cols int, labels []string, cells []*image.RGBA) []byte {
	t.Helper()
	const gutter, headerH = 6, 16
	rows := (len(cells) + cols - 1) / cols
	img := image.NewRGBA(image.Rect(0, 0, cols*cellW+(cols+1)*gutter, rows*(cellH+headerH)+(rows+1)*gutter))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 20, G: 20, B: 24, A: 255}), image.Point{}, draw.Src)

	for i, cell := range cells {
		x0 := gutter + (i%cols)*(cellW+gutter)
		y0 := gutter + (i/cols)*(cellH+headerH)
		drawLabel(img, x0+2, y0+2, labels[i])
		dp := image.Pt(x0, y0+headerH)
		draw.Draw(img, image.Rectangle{Min: dp, Max: dp.Add(image.Pt(cellW, cellH))}, cell, image.Point{}, draw.Over)
	}
	return encodePNG(t, img)
}

// renderStage は敷地計画 Site を1枚の *image.RGBA に描く。下地タイルを「庭→土 / 戸口→扉 / 部屋の内側→床 /
// それ以外→壁」で決め、実スプライトを重ね、各部屋に役割ラベルを描く。footprint 全体を走査するので、前庭や
// 坪庭・玄関の凹み・間仕切り・共有壁の戸口が1枚に現れる。overworld の描画と同じ分類を使い VRT と実装を揃える。
func renderStage(t *testing.T, site Site, placed []Placed) *image.RGBA {
	t.Helper()
	f := site.Footprint
	img := image.NewRGBA(image.Rect(0, 0, f.W*cellPx, f.H*cellPx))

	floor := site.floorSet()
	door := site.doorSet()
	for y := range f.H {
		for x := range f.W {
			v := Vec{X: f.X + x, Y: f.Y + y}
			var c color.RGBA
			switch {
			case site.Garden[v]:
				c = gardenColor
			case door[v]:
				c = tileColor(false, true)
			case floor[v] && !site.ExtraWall[v]:
				c = tileColor(false, false)
			default:
				c = tileColor(true, false)
			}
			fillCell(img, x*cellPx, y*cellPx, c)
		}
	}

	drawPlaced(t, img, Vec{X: f.X, Y: f.Y}, placed)

	// 役割ラベルは什器を描いた後に重ね、家具に埋もれず読めるようにする。左上の床側へ寄せて戸口や壁を避ける
	for _, hr := range site.Rooms {
		lx := (hr.Room.Rect.X + 1 - f.X) * cellPx
		ly := (hr.Room.Rect.Y + 1 - f.Y) * cellPx
		drawLabel(img, lx+2, ly+2, hr.Role)
	}
	return img
}

// gardenColor は庭・前庭・坪庭の下地色。壁や床と分かれる土のくすんだ緑。
var gardenColor = color.RGBA{R: 54, G: 62, B: 44, A: 255}

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
