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

// FurnishStages は敷地計画と内装を加工ステップごとに分けて返す。footprint を建物と庭に分ける planSite を
// 前段に置き、建物内の各室へ plan→fill→age→flavor を掛けた累積配置を返す。どの段で見た目が壊れたかを段別
// VRT で切り分けられるようにする。坪庭の室は家具を置かず観葉だけ置き、経年・flavor を掛けない。FurnishBuilding
// はこの最終段を返す薄い包みで、両者は同じパイプラインを共有するので段別 VRT と実生成は乖離しない。
func FurnishStages(seed uint64, footprint Rect, door Vec, facility string) (Site, []FurnishStage) {
	site := planSite(footprint, seed, door, facility)

	aged := buildingAged(seed) // 経年は建物ごとに1つ。全室が揃って新品か廃墟になる
	var fill, decayed, flavored []Placed
	for i := range site.Rooms {
		hr := site.Rooms[i]
		roomSeed := childSeed(seed, 300+i)
		if hr.Role == roleGarden {
			// 坪庭。観葉だけ置き、経年も flavor も掛けない。囲われた庭を荒らさない
			g := FillRoom(roomSeed, hr.Room, gardenContent())
			fill = append(fill, g...)
			decayed = append(decayed, g...)
			flavored = append(flavored, g...)
			continue
		}
		f := FillRoom(roomSeed, hr.Room, roleContent(facility, hr.Role, seed))
		a := f
		if aged {
			a = Age(roomSeed, hr.Room, f)
		}
		fl := Flavor(roomSeed, hr.Room, a, facilityFlavor(facility))
		fill = append(fill, f...)
		decayed = append(decayed, a...)
		flavored = append(flavored, fl...)
	}
	return site, []FurnishStage{
		{Label: "1 plan", Placed: nil},
		{Label: "2 fill", Placed: fill},
		{Label: "3 age", Placed: decayed},
		{Label: "4 flavor", Placed: flavored},
	}
}

// FurnishBuilding は footprint を敷地計画し、建物内の各室へ内装を敷いて、敷地と最終配置を返す。footprint を
// そのまま埋めず、入口側に前庭を空け、1室を坪庭にし、玄関を凹ませる。加工は FurnishStages が持ち、ここは
// その最終段 flavor を返す。呼び出し側は Site から Walls で壁タイル、Garden で庭タイルを導き、配置を spawn する。
func FurnishBuilding(seed uint64, footprint Rect, door Vec, facility string) (Site, []Placed) {
	site, stages := FurnishStages(seed, footprint, door, facility)
	return site, stages[len(stages)-1].Placed
}

// gardenContent は坪庭の内装。観葉を全域に点在させる。家具は置かない。dirt の地面に緑を散らして庭に見せる。
func gardenContent() Content {
	return Content{ID: roleGarden, Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindDecor, Ref: "plant", Placement: PlaceFullArea, Amount: Dice{Base: 2, Sides: 3}},
		}},
	}}
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

// roleContent は役割から content を引く。main は施設の顔、それ以外はまず施設の room カタログ、無ければ
// 民家の共有役割(corridor 等)、それも無ければ施設別の奥室既定へ落とす。民家だけでなく店・診療所も役割名で
// 部屋を作り分けられるよう、施設カタログを優先して引く。役割名は planRooms とテンプレが付ける。
func roleContent(facility, role string, seed uint64) Content {
	if role == "main" {
		return facilityContent(facility, seed)
	}
	if c, ok := facilityRoomContents(facility)[role]; ok {
		return c
	}
	if c, ok := houseRoomContents()[role]; ok {
		return c
	}
	return backRoomContent(facility)
}

// facilityRoomContents は施設種別ごとの「役割名→content」表を返す。houseRoomContents を民家以外へ横展開
// したもので、テンプレが付けた役割名で各室の内装を引く。骨董品店は店、研究施設は診療所の表を共有する。
func facilityRoomContents(facility string) map[string]Content {
	switch facility {
	case facHouse:
		return houseRoomContents()
	case facStore, facAntique:
		return storeRoomContents()
	case facClinic, facLab:
		return clinicRoomContents()
	default:
		return nil
	}
}

// storeRoomContents は店の奥室の役割別 content。倉庫だけでなく、事務所・従業員トイレ・冷蔵庫室に作り分け、
// 奥室が全部同じ樽の物置になる単調さを解く。什器は既存を流用し新しい語彙は要らない。
func storeRoomContents() map[string]Content {
	return map[string]Content{
		"storeroom": storageRoomContent(),
		"office":    officeRoomContent(),
		"restroom":  restroomContent(),
		"coldroom":  coldroomContent(),
	}
}

// clinicRoomContents は診療所の各室の役割別 content。待合を診察室と分け、薬局・トイレ・供給室に作り分け、
// 奥室が全部同じ診察室になる単調さを解く。
func clinicRoomContents() map[string]Content {
	return map[string]Content{
		"waiting":  waitingContent(),
		"exam":     examRoomContent(),
		"pharmacy": pharmacyRoomContent(),
		"restroom": restroomContent(),
		"office":   officeRoomContent(),
	}
}

// officeRoomContent は事務所の内装。机と椅子を列に並べ、壁際に書類棚。店の奥や診療所の医師室に使う。
func officeRoomContent() Content {
	return Content{ID: "office", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "desk", Placement: PlaceRow, Amount: Dice{Bonus: 2}},
			{Kind: KindFurniture, Ref: "chair", Placement: PlaceRow, Amount: Dice{Bonus: 2}},
			{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
		}},
	}}
}

// restroomContent は水回りの小部屋。便器と流し。店の従業員トイレや診療所のトイレに使う。
func restroomContent() Content {
	return Content{ID: "restroom", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "toilet", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "sink", Amount: Dice{Bonus: 1}},
		}},
	}}
}

// coldroomContent は冷蔵庫室。冷蔵ケースを壁沿いに並べる。店の奥の生鮮の保管に使う。
func coldroomContent() Content {
	return Content{ID: "coldroom", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "walkin_cooler", Placement: PlaceWall, Amount: Dice{Bonus: 4}},
		}},
	}}
}

// pharmacyRoomContent は薬局・薬品庫。薬棚を壁一面に並べ、奥に薬を置く。診療所の施錠戦利品の受け皿になる
// 部屋型で、待合の主室に薬棚を積んでいた scope 過大を解く。
func pharmacyRoomContent() Content {
	return Content{ID: "pharmacy", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "medcabinet", Placement: PlaceWall, Amount: Dice{Bonus: 4}},
		}},
		{Style: PickN, Pick: 1, Items: []Stuff{
			{Kind: KindLoot, Ref: "meds", Placement: PlaceFarFromDoor, Amount: Dice{Base: 1, Sides: 3}},
		}},
	}}
}

// waitingContent は診療所の待合専用。受付を入口近く、長椅子を列に、観葉を添える。診察台は置かない。単室
// 診療所の clinicContent が待合に診察台まで積んで「待合に見えない」問題を、多部屋では待合専用へ切り出して解く。
func waitingContent() Content {
	return Content{ID: "waiting", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			{Kind: KindFurniture, Ref: "reception", Placement: PlaceNearDoor, Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "waitchair", Placement: PlaceRow, Amount: Dice{Bonus: 5}},
		}},
		{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 2}}}},
	}}
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
