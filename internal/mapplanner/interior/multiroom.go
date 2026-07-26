package interior

import "sort"

// 多部屋の建物内装。overworld の大きな建物外殻を部屋へ分割し、主室を施設の顔、奥室を物置・寝室・診察室に
// して埋める。単室では倉庫のように広く間延びする大きな建物に、間仕切りと役割で構造の変化を与える。
// 建物入口は外殻側が持つので分割は addEntrance をしない。小さすぎて割れない footprint は1部屋に収まる。

// SubdivideBuilding は footprint を BSP で複数部屋へ分割し戸口で相互連結する。建物入口を開けない点だけ
// SplitBuilding と違う。返す部屋は相互に連結し、外殻の扉から入った部屋から全室へ到達できる。
func SubdivideBuilding(footprint Rect, seed uint64) []Room {
	rects := bspSplit(footprint, seed, 0)
	rooms := make([]Room, len(rects))
	for i, r := range rects {
		rooms[i] = Room{Rect: r}
	}
	connectRooms(rooms, seed)
	return rooms
}

// FurnishStage は建物内装の加工1段の累積配置。Label は段名、Placed はその段まで適用した全室の配置。
// plan は空の間取り、fill は什器、age は経年、flavor は装飾を足した段を表す。
type FurnishStage struct {
	Label  string
	Placed []Placed
}

// FurnishStages は建物内装を加工ステップごとに分けて返す。plan→fill→age→flavor の各段の累積配置を返し、
// どの段で見た目が壊れたかを段別 VRT で切り分けられるようにする。mapplanner の PlannerChain スナップショットに
// 倣い、最終形だけでなく途中を残す。FurnishBuilding はこの最終段を返す薄い包みで、両者は同じパイプラインを
// 共有するので段別 VRT と実生成は乖離しない。
func FurnishStages(seed uint64, footprint Rect, door Vec, facility string) ([]HouseRoom, []FurnishStage) {
	rooms, roles := planRooms(footprint, seed, facility)
	attachEntrance(rooms, footprint, door)

	aged := buildingAged(seed) // 経年は建物ごとに1つ。全室が揃って新品か廃墟になる
	labeled := make([]HouseRoom, len(rooms))
	var fill, decayed, flavored []Placed
	for i := range rooms {
		roomSeed := childSeed(seed, 300+i)
		f := FillRoom(roomSeed, rooms[i], roleContent(facility, roles[i], seed))
		a := f
		if aged {
			a = Age(roomSeed, rooms[i], f)
		}
		fl := Flavor(roomSeed, rooms[i], a, facilityFlavor(facility))
		fill = append(fill, f...)
		decayed = append(decayed, a...)
		flavored = append(flavored, fl...)
		labeled[i] = HouseRoom{Room: rooms[i], Role: roles[i]}
	}
	return labeled, []FurnishStage{
		{Label: "1 plan", Placed: nil},
		{Label: "2 fill", Placed: fill},
		{Label: "3 age", Placed: decayed},
		{Label: "4 flavor", Placed: flavored},
	}
}

// FurnishBuilding は建物外殻を多部屋へ割り、部屋ごとに内装を敷いて、役割付きの部屋と最終配置を返す。面積最大の
// 部屋を施設の主室、残りを奥室にして、店なら売り場＋倉庫、民家なら居間＋寝室、の構造にする。door は外殻の
// 入口で、それに面する部屋へ戸口として足し、家具が入口を塞がないようにする。各室に時間の層と flavor も掛ける。
// 加工は FurnishStages が持ち、ここはその最終段 flavor を返す。呼び出し側は返す部屋から InternalWalls で
// 間仕切りタイルを導き、配置を spawn する。
func FurnishBuilding(seed uint64, footprint Rect, door Vec, facility string) ([]HouseRoom, []Placed) {
	rooms, stages := FurnishStages(seed, footprint, door, facility)
	return rooms, stages[len(stages)-1].Placed
}

// InternalWalls は役割付き部屋から建物内部の間仕切りタイルを返す。overworld が壁タイルを描くのに使う。
func InternalWalls(footprint Rect, rooms []HouseRoom, door Vec) []Vec {
	plain := make([]Room, len(rooms))
	for i, r := range rooms {
		plain[i] = r.Room
	}
	return internalWalls(footprint, plain, door)
}

// planRooms は施設に応じて部屋群と各部屋の役割を返す。施設ごとに固有の間取りテンプレを持ち、民家は廊下型・
// 店は売場＋バックヤード・診療所は待合＋診察室の列にして、「何の施設か分かる」平面にする。テンプレに足りない
// 小さな footprint と、テンプレの無い施設は BSP へ落として面積最大を主室・残りを奥室にする。実経路のこの分岐が、
// VRT で見た綺麗な間取りを in-game でも出す。
func planRooms(footprint Rect, seed uint64, facility string) ([]Room, []string) {
	if planner, ok := facilityPlanner(facility); ok && footprint.W >= 24 && footprint.H >= 16 {
		plan := planner(footprint, seed)
		rooms := make([]Room, len(plan))
		roles := make([]string, len(plan))
		for i, hr := range plan {
			rooms[i] = hr.Room
			roles[i] = hr.Role
		}
		return rooms, roles
	}
	rooms := SubdivideBuilding(footprint, seed)
	roles := make([]string, len(rooms))
	for rank, ri := range roomOrderByArea(rooms) {
		if rank == 0 {
			roles[ri] = "main"
		} else {
			roles[ri] = "back"
		}
	}
	return rooms, roles
}

// facilityPlanner は施設種別に対応する間取りテンプレを返す。テンプレを持つ施設だけ ok を true にし、無い
// 施設は BSP のフォールバックへ委ねる。骨董品店は店、研究施設は診療所のテンプレを共有する。
func facilityPlanner(facility string) (func(Rect, uint64) []HouseRoom, bool) {
	switch facility {
	case facHouse:
		return PlanHouseAny, true
	case facStore, facAntique:
		return PlanStore, true
	case facClinic, facLab:
		return PlanClinic, true
	default:
		return nil, false
	}
}

// roleContent は役割から content を引く。main は施設の内装、back は施設別の奥室、それ以外は PlanHouse が
// 付ける民家の役割別 content。役割名は planRooms が付ける。
func roleContent(facility, role string, seed uint64) Content {
	switch role {
	case "main":
		return facilityContent(facility, seed)
	case "back":
		return backRoomContent(facility)
	default:
		if c, ok := houseRoomContents()[role]; ok {
			return c
		}
		return backRoomContent(facility)
	}
}

// attachEntrance は外殻の入口 door に面する部屋を見つけ、その部屋へ door を戸口として足す。家具が入口を
// 塞がず、入口近くの配置が入口へ寄るようにする。
func attachEntrance(rooms []Room, footprint Rect, door Vec) {
	inner := doorInner(footprint, door)
	for i := range rooms {
		if rooms[i].Rect.containsInterior(inner) {
			rooms[i].Doorways = append(rooms[i].Doorways, Doorway(door))
			return
		}
	}
}

// doorInner は外壁上の door の1つ内側のタイルを返す。北・南・西・東の壁を判定する。
func doorInner(footprint Rect, door Vec) Vec {
	switch {
	case door.Y == footprint.Y:
		return Vec{X: door.X, Y: door.Y + 1}
	case door.Y == footprint.Y+footprint.H-1:
		return Vec{X: door.X, Y: door.Y - 1}
	case door.X == footprint.X:
		return Vec{X: door.X + 1, Y: door.Y}
	default:
		return Vec{X: door.X - 1, Y: door.Y}
	}
}

// internalWalls は建物内部の間仕切りタイルを返す。footprint の外周は外殻が壁を持つので除き、部屋の内側床
// でも戸口でもない内側タイルを間仕切りとする。入口の1つ内側は床に保ち、扉を開けたら壁、を避ける。
func internalWalls(footprint Rect, rooms []Room, door Vec) []Vec {
	floor := make(map[Vec]bool)
	doorSet := make(map[Vec]bool)
	for _, rm := range rooms {
		for _, v := range rm.Rect.interiorTiles() {
			floor[v] = true
		}
		for _, d := range rm.Doorways {
			doorSet[Vec(d)] = true
		}
	}
	inner := doorInner(footprint, door)

	right, bottom := footprint.X+footprint.W-1, footprint.Y+footprint.H-1
	var walls []Vec
	for y := footprint.Y + 1; y < bottom; y++ {
		for x := footprint.X + 1; x < right; x++ {
			v := Vec{X: x, Y: y}
			if v == inner || floor[v] || doorSet[v] {
				continue
			}
			walls = append(walls, v)
		}
	}
	return walls
}

// roomOrderByArea は部屋を面積降順の添字列で返す。主室に最大の部屋を選ぶための順序。
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

// backRoomContent は奥室の内装。施設ごとに、店は物置、民家は寝室、診療所は診察室にする。既存の家具を
// 使い回すので新しい content 語彙は要らない。物置は樽を控えめにする。depot 本体の樽詰めとは分け、奥室が
// 樽だらけで単調になるのを避ける。
func backRoomContent(facility string) Content {
	switch facility {
	case facHouse:
		return bedroomContent()
	case facClinic, facLab:
		return examRoomContent()
	default:
		return storageRoomContent()
	}
}

// storageRoomContent は奥室の物置。樽を数個だけ置く。倉庫施設 depotContent の樽詰めより疎にする。
func storageRoomContent() Content {
	return Content{ID: "storage", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "barrel", Amount: Dice{Bonus: 3}},
		}},
	}}
}

// bedroomContent は民家の奥室。寝床の一角と物入れ。
func bedroomContent() Content {
	return Content{ID: "bedroom", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			bedSet(), // ベッドとクローゼットを寝床の一角に束ねる
			{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 1}},
		}},
	}}
}

// examRoomContent は診療所の奥室。診察台と薬棚。
func examRoomContent() Content {
	return Content{ID: "exam", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "exam_bed", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "medcabinet", Amount: Dice{Bonus: 1}},
		}},
	}}
}

// houseRoomContents は民家の部屋役割ごとの content を役割名で引く表。PlanHouse が決めた役割へ中身を
// 対応させる。廊下はほぼ空けて通路とし、玄関は下足入れと観葉、水回りは各機能の什器を置く。狭い部屋が
// 多いので個数は控えめにする。VRT の renderHousePlan と in-game の FurnishBuilding が同じ表を共有し、
// 見た目と生成の乖離を防ぐ。
func houseRoomContents() map[string]Content {
	bedroom := Content{ID: "bedroom", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			bedSet(), // 寝床は常設。寝室の署名
		}},
		// 枕元の添え物を seed で1つ。明かり・観葉・物入れのどれかで、同じ寝室が続かないようにする
		{Style: PickOne, Items: []Stuff{
			{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 1}},
			{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
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
				{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 2}}, // 明かりは常設
			}},
			// 主座は食卓か寛ぎのソファのどちらか。seed で選び、同じ居間が続かないようにする
			{Style: PickOne, Items: []Stuff{
				diningTable(PlaceCenter),
				loungeSet(),
			}},
			// 添え物を seed で1つ。壁面の棚か観葉
			{Style: PickOne, Items: []Stuff{
				{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}},
			}},
		}},
		"kitchen": {ID: "kitchen", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				kitchenCounter(), // 調理台は常設。台所の署名
				{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 1}}, // 明かりは常設
			}},
			// 台所の主家具を seed で1つ。食卓か、もう一列の食器棚か
			{Style: PickOne, Items: []Stuff{
				{Kind: KindFurniture, Ref: "table", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "pantry", Placement: PlaceRow, Amount: Dice{Bonus: 2}},
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
