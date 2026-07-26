package interior

// このファイルは施設種別ごとの内装 content を持ち、overworld の建物外殻を家具で満たす公開入口 Furnish を
// 提供する。content の宣言は幾何から切り離され、Furnish に footprint と入口を渡せば施設種別に応じた配置が
// 決定的に返る。戦利品(KindLoot)は content に含むが、spawn するかは呼び出し側が選ぶ。urban の v1 は
// 家具と装飾だけを置き、施設固有の戦利品はアイテム設計が固まってから足す。

// Furnish は建物の footprint と入口から、施設種別に応じた内装の配置を決定的に返す。overworld の建物外殻を
// 内装で満たすための公開入口で、doc 69 の Feature 層が建物ではこの器を呼ぶ。未知の施設種別は汎用の
// 内装にする。footprint は外周が壁の1部屋とみなし、door はその外周上の入口。
func Furnish(seed uint64, footprint Rect, door Vec, facility string) []Placed {
	room := Room{Rect: footprint, Doorways: []Doorway{{X: door.X, Y: door.Y}}}
	placed := FillRoom(seed, room, facilityContent(facility, seed))
	// 時間の層。略奪・生活痕・廃墟化で瓦礫や破片を刻み、新品でなく打ち捨てられた見た目にする
	placed = Age(seed, room, placed)
	// 家具の隙間へ flavor machine を1つ置き、戦利品の無い空き箱部屋に character を与える
	return Flavor(seed, room, placed, facilityFlavor(facility))
}

// facilityFlavor は建物へ足す flavor machine の Content。廃墟に残る生活の痕を PickOne で1つ選ぶので、
// 建物ごとに蝋燭の輪・絨毯・散らばった蝋燭のいずれかが出て単調にならない。施設別の flavor は今後。
func facilityFlavor(facility string) Content {
	_ = facility // 施設別カタログは今後。まずは全施設に共通の廃墟の痕を置く
	return Content{
		ID: "flavor",
		Groups: []Group{
			{Style: PickOne, Items: []Stuff{
				candleCircle(), // 儀式の輪は稀。重み1で、ありふれた痕の陰から時々現れる
				{Kind: KindDecor, Ref: "carpet", Placement: PlaceFarFromDoor, Weight: 3, Amount: Dice{Bonus: 1}},
				{Kind: KindDecor, Ref: "candle", Placement: PlaceFullArea, Weight: 3, Amount: Dice{Base: 1, Sides: 2}},
			}},
		},
	}
}

// candleCircle は蝋燭の輪の flavor machine。中心を空け、その周囲8マスへ蝋燭を輪に並べる。廃墟で誰かが
// 儀式めいた円を組んだ痕を、束の機構で1つの scene として置く。anchor は輪の中心だが raw を持たない印に
// して spawn されず、中心が空くことで塊でなく輪に見える。装飾なので通行を阻まない。
func candleCircle() Stuff {
	offs := []Vec{{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 0}, {X: 1, Y: 0}, {X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1}}
	ring := make([]Satellite, 0, len(offs))
	for _, o := range offs {
		ring = append(ring, Satellite{Kind: KindDecor, Ref: "candle", Offsets: []Vec{o}})
	}
	return Stuff{Kind: KindDecor, Ref: "ritual_center", Placement: PlaceCenter, Amount: Dice{Bonus: 1}, Satellites: ring}
}

// facilityContent は施設種別名から内装 content を1つ引く。同じ施設種別でも複数の変種を持ち、seed で
// 引くことで同じ店が薬局にも食料品店にもなる。doc L694 の最優先「部屋アーキタイプ数」を、既存家具の
// 組み替えだけでデータを足さずに増やす。変種の seed は本体生成と別枠にして相関を避ける。
func facilityContent(facility string, seed uint64) Content {
	variants := facilityVariants(facility)
	return variants[int(childSeed(seed, 9_000_000)%uint64(len(variants)))]
}

// facilityVariants は施設種別ごとの内装変種の一覧。骨董品店は商店、研究施設は診療所へ寄せ、未知は汎用に
// する。変種を足すときはここへ Content を加えるだけでよい。
func facilityVariants(facility string) []Content {
	switch facility {
	case "house":
		return []Content{houseContent(), studioContent()}
	case "store":
		return []Content{storeContent(), pharmacyContent(), groceryContent()}
	case "antique":
		return []Content{storeContent()}
	case "clinic", "lab":
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
