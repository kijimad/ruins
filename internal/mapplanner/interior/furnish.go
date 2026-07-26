package interior

import "sort"

// 建物内装のパイプラインと役割ルーティング。footprint を敷地計画して部屋へ割り、各部屋の役割から content を
// 引いて加工する。overworld の大きな建物外殻を、間仕切りと役割で構造の変化のある内装に変える。
// 加工の各段は FurnishStage で残し、段別 VRT でどの加工が犯人かを切り分けられるようにする。

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
func facilityPlanner(facility string) (func(Rect, uint64) []PlannedRoom, bool) {
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
	if c, ok := roomCatalog(facility)[role]; ok {
		return c
	}
	if c, ok := houseRoomContents()[role]; ok {
		return c
	}
	return backRoomContent(facility)
}

// roomCatalog は施設種別ごとの「役割名→content」表を返す。houseRoomContents を民家以外へ横展開したもので、
// テンプレが付けた役割名で各室の内装を引く。骨董品店は店、研究施設は診療所の表を共有する。
func roomCatalog(facility string) map[string]Content {
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

// backRoomContent は奥室の内装。施設ごとに、店は物置、民家は寝室、診療所は診察室にする。既存の家具を
// 使い回すので新しい content 語彙は要らない。役割カタログに無い役割のフォールバック。
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
