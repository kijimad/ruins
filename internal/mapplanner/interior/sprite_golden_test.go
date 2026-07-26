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
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// spriteDir はスプライトのソース PNG(32x32)の場所。テストはパッケージ直下で走るのでリポジトリ根へ遡る。
const spriteDir = "../../../assets/file/textures/single/"

// cellPx は1タイルの描画辺長。スプライトが 32x32 なのでセルも 32px にして等倍で置く。
const cellPx = 32

// spriteFileOf は配置の Ref を実スプライトのソース PNG 名へ写す。ゲーム本体の豊富な既存スプライトから
// 什器の実物を当てるので、ダミーは使わない。什器ごとに別の絵になり、施設の見分けが付く。スプライトの
// 無い装飾は背景描画で表す。
func spriteFileOf(p Placed) string {
	switch p.Ref {
	// 店。gondola_shelf_ / display_cooler_ はゲーム側が未だ仮画像なので、実描画のある goods_shelf_ /
	// refrigerator_ を当てる
	case "gondola":
		return "goods_shelf_"
	case "register":
		return "register_"
	case "walkin_cooler":
		return "refrigerator_"
	case "snacks":
		return "cookie_"
	case "drinks":
		return "bottled_cola_"
	case "bento":
		return "hamburger_"
	// 診療所。reception_counter_ / exam_bed_ / medicine_cabinet_ はゲーム側が未だ仮画像だが、無理に
	// 寝室の什器で代用すると診療所に見えなくなるため、意味の合う実資産をそのまま当てる
	case "reception":
		return "reception_counter_"
	case "waitchair":
		return "bench_"
	case "exam_bed":
		return "exam_bed_"
	case "medcabinet":
		return "medicine_cabinet_"
	case "meds":
		return "healing_potion_"
	case "bandage":
		return "leather_bandage_"
	// 民家・建物
	case "bed":
		return "bed_"
	case "table":
		return "dining_table_"
	case "chair":
		return "chair_"
	case "sofa":
		return "sofa_"
	case "closet":
		return "closet_"
	case "lantern":
		return "tall_lamp_"
	case "plant":
		return "houseplants_"
	case "pantry":
		return "dish_shelf_"
	case "barrel", "crate":
		return "barrel_brown_"
	case "bathtub":
		return "bathtub_"
	case "toilet":
		return "toilet_"
	case "sink":
		return "sink_"
	case "washer":
		return "wash_machine_empty_"
	// flavor。廃墟に残る生活の痕
	case "candle":
		return "candle_"
	case "carpet":
		return "carpet_"
	case "broom":
		return "broom_"
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

// TestGolden_InteriorFlavorMachine は flavor machine の目視回帰。ベッドと棚だけの生活痕の薄い部屋に
// Flavor パスで蝋燭の輪を足し、戦利品を増やさず character を与えて空き箱部屋を無くす様子を確認する。
// 蝋燭の輪は束の機構で置く1つの scene で、装飾ゆえ通行を阻まない。
func TestGolden_InteriorFlavorMachine(t *testing.T) {
	t.Parallel()

	room := Room{Rect: Rect{X: 0, Y: 0, W: 13, H: 10}, Doorways: []Doorway{{X: 6, Y: 9}}}
	base := Content{ID: "sparse", Groups: []Group{{Style: PickEach, Items: []Stuff{
		{Kind: KindFurniture, Ref: "bed", Amount: Dice{Bonus: 1}},
		{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 2}},
	}}}}
	placed := FillRoom(3, room, base)
	placed = Flavor(3, room, placed, abandonedFlavor())
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderRoomSprites(t, room, placed))
}

// TestGolden_InteriorHouseBuilding は横廊下の民家1棟の目視回帰。玄関から入って廊下に出て、廊下から
// 居間・台所・寝室へ、さらに廊下沿いの脱衣所から浴室へ、という日本家屋の動線を持つ。均一な部屋の BSP
// では作れない、狭い玄関と廊下と水回りの小部屋を PlanHouse が保証する。
func TestGolden_InteriorHouseBuilding(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderHousePlan(t, footprint, PlanHouse(footprint, 1), 1))
}

// TestGolden_InteriorHouseVertical は縦廊下でも小部屋を作れることの目視回帰。縦廊下は左右が横長の帯に
// なりがちだが、右下の水回りを入れ子に再帰分割し、脱衣所だけ廊下に面させ浴室とトイレをその奥へ nest
// することで、横廊下と同じく小さなトイレ・浴室を得る。廊下の向きは制約でなく選択であることを示す。
func TestGolden_InteriorHouseVertical(t *testing.T) {
	t.Parallel()

	footprint := Rect{X: 0, Y: 0, W: 28, H: 20}
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderHousePlan(t, footprint, PlanHouseVertical(footprint, 1), 1))
}

// renderHousePlan は民家プランの各部屋を役割 content で充填し、役割ラベル付きで実スプライトに描く。
// 横型と縦型の目視回帰が同じ経路を共有する。
func renderHousePlan(t *testing.T, footprint Rect, plan []HouseRoom, seed uint64) []byte {
	t.Helper()
	byRole := houseRoomContents()
	rooms := make([]Room, len(plan))
	roles := make([]string, len(plan))
	placed := make([]Placed, 0, len(plan)*8)
	for i, hr := range plan {
		rooms[i] = hr.Room
		roles[i] = hr.Role
		placed = append(placed, FillRoom(childSeed(seed, 500+i), hr.Room, byRole[hr.Role])...)
	}
	return renderBuildingSprites(t, footprint, rooms, roles, placed)
}

// TestGolden_InteriorClinicBuilding は診療所を建物1棟で生成した目視回帰。単室では待合も診察室も同じ
// 部屋に同居して施設に見えないため、複数部屋へ割り、入口の待合と受付・中間の診察室・奥の備品室と、
// ゾーンごとに役割を分ける。動線に沿って部屋を通ると診療所の体をなすかを人が判断する。同じ SplitBuilding /
// fillBuilding に別の content セットを流すだけで別施設の建物になることの実証でもある。
func TestGolden_InteriorClinicBuilding(t *testing.T) {
	t.Parallel()

	const seed = 4
	footprint := Rect{X: 0, Y: 0, W: 26, H: 18}
	rooms := SplitBuilding(footprint, seed)
	require.GreaterOrEqual(t, len(rooms), 3, "footprint が複数部屋に割れる")

	placed := fillBuilding(seed, footprint, rooms, clinicRoleContents())
	roles := buildingRoles(footprint, rooms, clinicRoleContents())
	g := goldie.New(t, goldie.WithNameSuffix(".png"))
	g.Assert(t, t.Name(), renderBuildingSprites(t, footprint, rooms, roles, placed))
}

// assignContents は各部屋へ入口からの距離順で役割 content を割り当て、部屋の添字順に返す。手前を公共、
// 奥を私的の役割に割ることで施設の動線に沿う。contents は入口から奥への順に並べた役割の列で、部屋数が
// 役割数を超えたら末尾を繰り返す。充填とラベル表示がこの1関数を共有し、役割の二重管理を防ぐ。
func assignContents(footprint Rect, rooms []Room, contents []Content) []Content {
	out := make([]Content, len(rooms))
	for rank, ri := range roomOrderByZone(footprint, rooms) {
		out[ri] = contents[min(rank, len(contents)-1)]
	}
	return out
}

// fillBuilding は assignContents の割り当てに従い、部屋ごとに FillRoom を回して全配置を集める。content
// セットを差し替えれば同じ器で別施設の建物になる。
func fillBuilding(seed uint64, footprint Rect, rooms []Room, contents []Content) []Placed {
	all := make([]Placed, 0, len(rooms)*8)
	for ri, c := range assignContents(footprint, rooms, contents) {
		all = append(all, FillRoom(childSeed(seed, 500+ri), rooms[ri], c)...)
	}
	return all
}

// buildingRoles は各部屋に割り当てた役割 ID を部屋の添字順に返す。VRT のラベル表示に使い、ゾーン分類の
// 意図が什器の結果と噛み合っているかを人が照合できるようにする。
func buildingRoles(footprint Rect, rooms []Room, contents []Content) []string {
	assigned := assignContents(footprint, rooms, contents)
	roles := make([]string, len(assigned))
	for i, c := range assigned {
		roles[i] = c.ID
	}
	return roles
}

// roomOrderByZone は入口からの距離を主キー、面積を副キーに部屋の添字を並べる。手前ほど公共、奥ほど
// 私的、という住居の動線に沿って役割を割り当てるための順序。同じ距離の層では大きい部屋を先にして、
// 公共の間ほど広く取る住居の傾向に寄せる。
func roomOrderByZone(footprint Rect, rooms []Room) []int {
	depths := roomDepths(footprint, rooms)
	idx := make([]int, len(rooms))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		if depths[idx[a]] != depths[idx[b]] {
			return depths[idx[a]] < depths[idx[b]]
		}
		ra, rb := rooms[idx[a]].Rect, rooms[idx[b]].Rect
		return ra.W*ra.H > rb.W*rb.H
	})
	return idx
}

// houseRoomContents は民家の部屋役割ごとの content を役割名で引く表。PlanHouse が決めた役割へ中身を
// 対応させる。廊下はほぼ空けて通路とし、玄関は下足入れと観葉、水回りは各機能の什器を置く。狭い部屋が
// 多いので個数は控えめにし、FillRoom が入る分だけ置く。
func houseRoomContents() map[string]Content {
	// 置き方は archetype 既定に任せ、レシピは Ref と個数だけ言う。ベッドは奥・棚やランタンは壁際、と
	// いった家具型の自然な置き方が施設をまたいで揃う。廊下の観葉だけ既定の全域でなく奥へ寄せる例外。
	bedroom := Content{ID: "bedroom", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "bed", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 1}},
		}},
	}}
	return map[string]Content{
		"genkan": {ID: "genkan", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 1}},
			}},
			{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}}}},
		}},
		"corridor": {ID: "corridor", Groups: []Group{
			{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 1}}}},
		}},
		"living": {ID: "living", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "sofa", Amount: Dice{Bonus: 1}},
				diningTable(PlaceCenter),
				{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 2}},
			}},
			{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}}}},
		}},
		"kitchen": {ID: "kitchen", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "table", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "pantry", Amount: Dice{Bonus: 3}},
				{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 1}},
			}},
		}},
		"bedroom": bedroom,
		"dressing": {ID: "dressing", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "washer", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "sink", Amount: Dice{Bonus: 1}},
			}},
		}},
		"bath": {ID: "bath", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "bathtub", Amount: Dice{Bonus: 1}},
			}},
		}},
		"toilet": {ID: "toilet", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "toilet", Amount: Dice{Bonus: 1}},
			}},
		}},
		"storage": {ID: "storage", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "barrel", Amount: Dice{Bonus: 2}},
			}},
		}},
	}
}

// clinicRoleContents は診療所の部屋役割を入口から奥への順で並べた content。待合と受付・診察室・診察室・
// 診察室・備品室の順に対応させる。手前は患者が留まる待合と受付、奥は診察と備品という診療所の動線を
// content 側の並びで表す。部屋が役割数を超えたら末尾の備品室を繰り返す。
func clinicRoleContents() []Content {
	// 置き方は archetype 既定に任せる。診察室の薬だけ既定の奥でなく壁の棚へ、待合の椅子だけ列に並べる例外。
	examRoom := Content{ID: "exam", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "exam_bed", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "medcabinet", Amount: Dice{Bonus: 1}},
		}},
		{Style: PickOne, Items: []Stuff{{Kind: KindLoot, Ref: "meds", Placement: PlaceWall, Amount: Dice{Bonus: 1}}}},
	}}
	return []Content{
		{ID: "waiting", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "reception", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "waitchair", Placement: PlaceRow, Amount: Dice{Bonus: 4}},
			}},
			{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}}}},
		}},
		examRoom,
		examRoom,
		examRoom,
		examRoom,
		{ID: "supply", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "medcabinet", Amount: Dice{Bonus: 2}},
				{Kind: KindLoot, Ref: "bandage", Amount: Dice{Bonus: 2}},
				{Kind: KindLoot, Ref: "meds", Amount: Dice{Bonus: 1}},
			}},
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
// 上に開けた戸口も、1枚の絵として現れる。各部屋の左上に割り当て役割のラベルを重ね、ゾーン分類の意図と
// 什器の結果を人が照合できるようにする。
func renderBuildingSprites(t *testing.T, footprint Rect, rooms []Room, roles []string, placed []Placed) []byte {
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

	for i, rm := range rooms {
		lx := (rm.Rect.X + 1 - footprint.X) * cellPx
		ly := (rm.Rect.Y + 1 - footprint.Y) * cellPx
		drawLabel(img, lx+2, ly+2, roles[i])
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
