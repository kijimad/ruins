package interior

// このファイルは施設種別ごとの内装 content を持ち、overworld の建物外殻を家具で満たす公開入口 Furnish を
// 提供する。content の宣言は幾何から切り離され、Furnish に footprint と入口を渡せば施設種別に応じた配置が
// 決定的に返る。戦利品(KindLoot)は content に含むが、spawn するかは呼び出し側が選ぶ。urban の v1 は
// 家具と装飾だけを置き、施設固有の戦利品はアイテム設計が固まってから足す。

// 施設種別名。overworld の facilityType の文字列と揃える。switch の case で繰り返すので定数にする。
const (
	facHouse  = "house"
	facClinic = "clinic"
	facLab    = "lab"
)

// Furnish は建物の footprint と入口から、施設種別に応じた内装の配置を決定的に返す。overworld の建物外殻を
// 内装で満たすための公開入口で、doc 69 の Feature 層が建物ではこの器を呼ぶ。未知の施設種別は汎用の
// 内装にする。footprint は外周が壁の1部屋とみなし、door はその外周上の入口。
func Furnish(seed uint64, footprint Rect, door Vec, facility string) []Placed {
	room := Room{Rect: footprint, Doorways: []Doorway{{X: door.X, Y: door.Y}}}
	placed := FillRoom(seed, room, facilityContent(facility, seed))
	// 時間の層。経年した建物だけ略奪・生活痕・廃墟化で瓦礫や破片を刻む。手つかずの建物は新品のまま
	if buildingAged(seed) {
		placed = Age(seed, room, placed)
	}
	// 家具の隙間へ flavor machine を1つ置き、戦利品の無い空き箱部屋に character を与える
	return Flavor(seed, room, placed, facilityFlavor(facility))
}

// buildingAged は建物が経年しているかを seed で決める。多くは荒れているが、時々手つかずの建物がある。
// 損傷を建物ごとの独立軸にし、全建物が一律に廃墟化して単調になるのを避ける。
func buildingAged(seed uint64) bool {
	// pristine 1 : aged 2。3棟に1棟は新品同様
	return childSeed(seed, 11_000_000)%3 != 0
}

// scaleDensity は建物の家具量を密度プロファイルで増減する。疎・普通・密を seed で引き、家具の個数を掛ける。
// 個数1の必須什器は1を保ち、詰め物の棚だけが増減する。密度プロファイルを直交軸にして、同じ内装でも
// がらんとした店と品で埋まった店を出し分ける。
func scaleDensity(c Content, seed uint64) Content {
	factors := []int{6, 10, 14} // ×/10。疎・普通・密
	f := factors[int(childSeed(seed, 10_000_000)%uint64(len(factors)))]
	if f == 10 {
		return c
	}
	for gi := range c.Groups {
		for ii := range c.Groups[gi].Items {
			it := &c.Groups[gi].Items[ii]
			if it.Kind != KindFurniture {
				continue // 家具だけ密度を変える。戦利品・装飾はそのまま
			}
			it.Amount.Base = scaleAmount(it.Amount.Base, f)
			it.Amount.Bonus = scaleAmount(it.Amount.Bonus, f)
		}
	}
	return c
}

// scaleAmount は個数を f/10 倍する。元が1以上なら最低1を保ち、必須の1個が密度で消えないようにする。
func scaleAmount(v, f int) int {
	if v <= 0 {
		return v
	}
	if s := v * f / 10; s >= 1 {
		return s
	}
	return 1
}

// facilityFlavor は建物へ足す flavor machine の Content。廃墟に残る生活の痕を PickOne で1つ選ぶので、
// 建物ごとに絨毯か散らばった蝋燭のいずれかが出て単調にならない。蝋燭を輪に組む儀式の scene は宗教施設の
// archetype が来たときに足す。今は宗教施設が無く、民家や店に儀式の輪が出ると意味をなさないので置かない。
func facilityFlavor(facility string) Content {
	_ = facility // 施設別カタログは今後。まずは全施設に共通の廃墟の痕を置く
	return Content{
		ID: "flavor",
		Groups: []Group{
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "carpet", Placement: PlaceFarFromDoor, Weight: 1, Amount: Dice{Bonus: 1}},
				{Kind: KindDecor, Ref: "candle", Placement: PlaceFullArea, Weight: 1, Amount: Dice{Base: 1, Sides: 2}},
			}},
		},
	}
}

// facilityContent は施設種別名から内装 content を1つ引く。同じ施設種別でも複数の変種を持ち、seed で
// 引くことで同じ店が薬局にも食料品店にもなる。doc L694 の最優先「部屋アーキタイプ数」を、既存家具の
// 組み替えだけでデータを足さずに増やす。変種の seed は本体生成と別枠にして相関を避ける。
func facilityContent(facility string, seed uint64) Content {
	variants := facilityVariants(facility)
	c := variants[int(childSeed(seed, 9_000_000)%uint64(len(variants)))]
	return scaleDensity(c, seed)
}

// facilityVariants は施設種別ごとの内装変種の一覧。骨董品店は商店、研究施設は診療所へ寄せ、未知は汎用に
// する。変種を足すときはここへ Content を加えるだけでよい。
func facilityVariants(facility string) []Content {
	switch facility {
	case facHouse:
		return []Content{houseContent(), studioContent()}
	case "store":
		return []Content{storeContent(), pharmacyContent(), groceryContent()}
	case "antique":
		return []Content{storeContent()}
	case facClinic, facLab:
		return []Content{clinicContent()}
	case "office":
		return []Content{officeContent()}
	case "depot":
		return []Content{depotContent()}
	default:
		return []Content{genericContent()}
	}
}

// storeContent はコンビニを模した Content。placement 意味論で「店に見える」配置を宣言する。冷蔵ケースは
// 奥、レジは入口近く、ゴンドラは列。placement は archetype 既定に任せ、レシピは Ref と個数だけを言う。
func storeContent() Content {
	return Content{
		ID: "conv_store",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "walkin_cooler", Amount: Dice{Bonus: 7}},
				{Kind: KindFurniture, Ref: "register", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "gondola", Amount: Dice{Bonus: 10}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "snacks", Weight: 3, Amount: Dice{Base: 2, Sides: 4}},
				{Kind: KindLoot, Ref: "drinks", Weight: 2, Amount: Dice{Base: 1, Sides: 4}},
				{Kind: KindLoot, Ref: "bento", Weight: 1, Amount: Dice{Base: 1, Sides: 3}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "litter", Amount: Dice{Base: 1, Sides: 3, Bonus: 1}},
			}},
		},
	}
}

// clinicContent は診療所を模した Content。受付と待合椅子は入口近く、診察ベッドは奥、薬棚は壁際。
func clinicContent() Content {
	return Content{
		ID: "clinic",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "reception", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "waitchair", Amount: Dice{Bonus: 5}},
				{Kind: KindFurniture, Ref: "exam_bed", Amount: Dice{Bonus: 3}},
				{Kind: KindFurniture, Ref: "medcabinet", Amount: Dice{Bonus: 3}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "meds", Weight: 2, Amount: Dice{Base: 1, Sides: 3}},
				{Kind: KindLoot, Ref: "bandage", Weight: 1, Amount: Dice{Base: 1, Sides: 2}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 2}},
			}},
		},
	}
}

// houseContent は民家を模した Content。ベッドは奥、食卓の机と椅子は中央、棚とランタンは壁際。
func houseContent() Content {
	return Content{
		ID: "house",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "bed", Amount: Dice{Bonus: 1}},
				diningTable(PlaceCenter),
				{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 2}},
				{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 2}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 2}},
			}},
		},
	}
}

// officeContent は事務所を模した Content。机と椅子を列に並べたオフィス島、壁際に書棚。
func officeContent() Content {
	return Content{
		ID: "office",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "desk", Placement: PlaceRow, Amount: Dice{Bonus: 4}},
				{Kind: KindFurniture, Ref: "chair", Placement: PlaceRow, Amount: Dice{Bonus: 4}},
				{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 2}},
			}},
		},
	}
}

// pharmacyContent は薬局の store 変種。薬棚を壁一面に並べ、ゴンドラは控えめ。同じ「店」でも薬局に見える。
func pharmacyContent() Content {
	return Content{
		ID: "pharmacy",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "medcabinet", Amount: Dice{Bonus: 6}},
				{Kind: KindFurniture, Ref: "register", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "gondola", Amount: Dice{Bonus: 4}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "meds", Weight: 3, Amount: Dice{Base: 2, Sides: 4}},
				{Kind: KindLoot, Ref: "bandage", Weight: 1, Amount: Dice{Base: 1, Sides: 3}},
			}},
		},
	}
}

// groceryContent は食料品店の store 変種。ゴンドラを大量に並べ、冷蔵ケースを増やす。売り場が広く見える。
func groceryContent() Content {
	return Content{
		ID: "grocery",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "gondola", Amount: Dice{Bonus: 14}},
				{Kind: KindFurniture, Ref: "walkin_cooler", Amount: Dice{Bonus: 4}},
				{Kind: KindFurniture, Ref: "register", Amount: Dice{Bonus: 2}},
			}},
			{Style: PickN, Pick: 2, Items: []Stuff{
				{Kind: KindLoot, Ref: "snacks", Weight: 2, Amount: Dice{Base: 2, Sides: 4}},
				{Kind: KindLoot, Ref: "drinks", Weight: 2, Amount: Dice{Base: 2, Sides: 4}},
			}},
		},
	}
}

// studioContent は民家の house 変種。食卓を持たず、ベッドと物入れが詰まったワンルーム。狭い暮らしに見える。
func studioContent() Content {
	return Content{
		ID: "studio",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "bed", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 3}},
				{Kind: KindFurniture, Ref: "table", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 1}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}},
			}},
		},
	}
}

// depotContent は倉庫を模した Content。樽を列に積む。
func depotContent() Content {
	return Content{
		ID: "depot",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "barrel", Amount: Dice{Bonus: 8}},
			}},
		},
	}
}

// genericContent は施設種別が未知の建物の汎用内装。空き箱にならないよう樽と観葉だけ置く。
func genericContent() Content {
	return Content{
		ID: "generic",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "barrel", Amount: Dice{Bonus: 3}},
			}},
			{Style: PickOne, Items: []Stuff{
				{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}},
			}},
		},
	}
}

// diningTable は椅子を四辺へ束ねた食卓の Stuff。机を anchor に上下左右へ椅子の衛星を配り、机だけあって
// 椅子が中央に縦並びする散布事故を anchor 相対の束で防ぐ。各椅子は正面の辺を第一候補に、埋まっていれば
// 両隣の斜めへ回り込む。
func diningTable(placement Placement) Stuff {
	chair := func(offs ...Vec) Satellite {
		return Satellite{Kind: KindFurniture, Ref: "chair", Offsets: offs}
	}
	return Stuff{
		Kind: KindFurniture, Ref: "table", Placement: placement, Amount: Dice{Bonus: 1},
		Satellites: []Satellite{
			chair(Vec{X: 0, Y: -1}, Vec{X: -1, Y: -1}, Vec{X: 1, Y: -1}),
			chair(Vec{X: 0, Y: 1}, Vec{X: -1, Y: 1}, Vec{X: 1, Y: 1}),
			chair(Vec{X: -1, Y: 0}, Vec{X: -1, Y: -1}, Vec{X: -1, Y: 1}),
			chair(Vec{X: 1, Y: 0}, Vec{X: 1, Y: -1}, Vec{X: 1, Y: 1}),
		},
	}
}

// bedSet は寝床の一角を束ねる寝室の署名 fixture。ベッドを奥へ置き、脇の空いた隣へクローゼットを寄せる。
// ベッドとクローゼットを個別に散らすと部屋の別々の壁へ離れて寝室に見えないので、束で1つの寝床に見せる。
// クローゼットは4方向を順に試し、壁の向きに依らずベッドの空いた隣へ回り込む。
func bedSet() Stuff {
	return Stuff{
		Kind: KindFurniture, Ref: "bed", Placement: PlaceFarFromDoor, Amount: Dice{Bonus: 1},
		Satellites: []Satellite{
			{Kind: KindFurniture, Ref: "closet", Offsets: []Vec{{X: 1}, {X: -1}, {Y: -1}, {Y: 1}}},
		},
	}
}

// kitchenCounter は調理台の一列を束ねる台所の署名 fixture。流し台を壁際に置き、食器棚を横へ連ねて
// カウンターの列に見せる。流しと棚を別々に散らすと台所と分からないので、束で調理台の連なりに見せる。
// 食器棚は水平の隣を優先し、横壁沿いなら一列に、縦壁沿いなら anchor の内側へ回り込む。
func kitchenCounter() Stuff {
	return Stuff{
		Kind: KindFurniture, Ref: "sink", Placement: PlaceWall, Amount: Dice{Bonus: 1},
		Satellites: []Satellite{
			{Kind: KindFurniture, Ref: "pantry", Offsets: []Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}},
			{Kind: KindFurniture, Ref: "pantry", Offsets: []Vec{{X: 2}, {X: -2}, {Y: 2}, {Y: -2}}},
		},
	}
}
