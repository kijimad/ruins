package interior

// 内装レシピのカタログ。施設と部屋役割ごとの Content 定義を1箇所へ集める。レシピは Ref と個数だけを言い、
// 置き方は archetype 既定に任せる。以前は施設 main 用が facility.go、役割 room 用が multiroom.go に分かれ、
// 同じ「什器の配合」を探すのに2ファイルを見る必要があった。ここへ集約して「部屋の種類→什器」の一覧にする。
//
// 施設の代表的な部屋(main)のレシピと、奥室の役割別レシピは同じカタログに並ぶ。office と pharmacy は
// 施設まるごと(officeContent/pharmacyContent)と奥室1つ(officeRoomContent/pharmacyRoomContent)で個数が
// 違うので別レシピにし、隣に並べて違いを見えるようにする。

// --- 施設 main のレシピ。facilityVariants が施設種別ごとに1つ以上を持ち、seed で1つ選ぶ。----------------

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

// clinicContent は診療所を模した Content。受付と待合椅子は入口近く、診察ベッドは奥、薬棚は壁際。単室診療所と
// テンプレの無い小さな footprint の main に使う。多部屋テンプレの待合は診察台を持たない waitingContent を使う。
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

// officeContent は事務所まるごとの Content。机と椅子を列に並べたオフィス島、壁際に書棚。事務所施設の main に
// 使う。奥室1つの小さな事務所は officeRoomContent で個数を控えめにする。
func officeContent() Content {
	return Content{
		ID: "office",
		Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "desk", Placement: PlaceRow, Amount: Dice{Bonus: 4}},
				{Kind: KindFurniture, Ref: "chair", Placement: PlaceRow, Amount: Dice{Bonus: 4}},
				{Kind: KindFurniture, Ref: "closet", Amount: Dice{Bonus: 2}},
			}},
			// 事務機の添え物を seed で1つ。ホワイトボードかプリンタでオフィスらしさを足す
			{Style: PickOne, Items: []Stuff{
				{Kind: KindFurniture, Ref: "whiteboard", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "printer", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
			}},
		},
	}
}

// pharmacyContent は薬局まるごとの store 変種。薬棚を壁一面に並べ、レジとゴンドラで店として売る。同じ「店」でも
// 薬局に見える。奥室の薬品庫は register/gondola を持たない pharmacyRoomContent で表す。
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
				// 蝋燭は壁際に寄せる。床の真ん中に散らすと意味不明な位置に浮くので PlaceWall で壁沿いに置く
				{Kind: KindDecor, Ref: "candle", Placement: PlaceWall, Weight: 1, Amount: Dice{Bonus: 1}},
			}},
		},
	}
}

// --- 役割別カタログ。テンプレが付けた役割名で奥室の内装を引く。民家だけでなく店・診療所も部屋を作り分ける。--

// houseRoomContents は民家の部屋役割ごとの content を役割名で引く表。PlanHouse が決めた役割へ中身を
// 対応させる。廊下はほぼ空けて通路とし、玄関は下足入れと観葉、水回りは各機能の什器を置く。狭い部屋が
// 多いので個数は控えめにする。VRT と in-game が同じ表を共有し、見た目と生成の乖離を防ぐ。
func houseRoomContents() map[string]Content {
	bedroom := Content{ID: "bedroom", Groups: []Group{
		{Style: PickEach, Items: []Stuff{
			bedSet(), // 寝床は常設。寝室の署名
		}},
		// 枕元の添え物を seed で1つ。明かり・観葉・物入れ・鏡台・ナイトテーブルで、同じ寝室が続かないようにする
		{Style: PickOne, Items: []Stuff{
			{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 1}},
			{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "dresser", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
			{Kind: KindFurniture, Ref: "bedside", Amount: Dice{Bonus: 1}},
		}},
	}}
	return map[string]Content{
		"genkan": {ID: "genkan", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "getabako", Placement: PlaceWall, Amount: Dice{Bonus: 1}}, // 下駄箱は玄関の署名
			}},
			{Style: PickOne, Items: []Stuff{{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}}}},
		}},
		// 廊下は通路として空ける。幅1の通路に什器や観葉を置くと歩行を塞ぐので何も置かない
		"corridor": {ID: "corridor"},
		"living": {ID: "living", Groups: []Group{
			{Style: PickEach, Items: []Stuff{
				{Kind: KindFurniture, Ref: "lantern", Amount: Dice{Bonus: 2}}, // 明かりは常設
			}},
			// 主座は食卓か寛ぎのソファのどちらか。seed で選び、同じ居間が続かないようにする
			{Style: PickOne, Items: []Stuff{
				diningTable(PlaceCenter),
				loungeSet(),
			}},
			// 添え物を seed で1つ。壁面の棚・観葉・和室の仏壇や神棚・本棚・暖炉・時計。出る家具で居間が変わる
			{Style: PickOne, Items: []Stuff{
				{Kind: KindFurniture, Ref: "closet", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindDecor, Ref: "plant", Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "butsudan", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "kamidana", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "bookshelf", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "fireplace", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "clock", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
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
			// 家電の添え物を seed で1つ。電子レンジかコーヒーメーカーで台所の生活感を足す
			{Style: PickOne, Items: []Stuff{
				{Kind: KindFurniture, Ref: "microwave", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
				{Kind: KindFurniture, Ref: "coffeemaker", Placement: PlaceWall, Amount: Dice{Bonus: 1}},
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

// clinicRoomContents は診療所の各室の役割別 content。待合を診察室と分け、薬局・トイレ・医師室に作り分け、
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

// --- 奥室1つぶんの役割別レシピ。上のカタログと backRoomContent が引く。-----------------------------------

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

// officeRoomContent は奥室1つぶんの小さな事務所。机と椅子を列に並べ、壁際に書類棚。事務所まるごとの
// officeContent より個数を控えめにして、店の奥や診療所の医師室に収める。
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

// pharmacyRoomContent は奥室1つぶんの薬局・薬品庫。薬棚を壁一面に並べ、奥に薬を置く。店として売る
// pharmacyContent と違い register/gondola を持たない。診療所の施錠戦利品の受け皿になる部屋型で、待合の
// 主室に薬棚を積んでいた scope 過大を解く。
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
