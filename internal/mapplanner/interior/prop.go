package interior

// prop の写像。interior の抽象 Ref をゲームの raw prop 名へ写す単一のソース。overworld がこの表で prop を
// spawn し、VRT がこの表で「in-game に出る物だけ」を描く。両者が同じ表を引くので VRT と in-game が乖離しない。
// 表に無い Ref は in-game で spawn されず、VRT でも描かれない。KindLoot の戦利品と raw の無い装飾は含めない。
// urban の v1 は家具と装飾だけを置き、施設固有の戦利品はアイテム設計が固まってから足す。既存の prop へ寄せ、
// raw の無い抽象什器は近い実物へ当てる。
var propRaw = map[string]string{
	"register":      "register",
	"gondola":       "goods_shelf",
	"walkin_cooler": "refrigerator",
	"reception":     "desk",
	"waitchair":     "chair",
	"exam_bed":      "exam_bed",
	"medcabinet":    "medicine_cabinet",
	"bed":           "bed",
	"table":         "table",
	"chair":         "chair",
	"sofa":          "sofa",
	"closet":        "closet",
	"lantern":       "lantern",
	"plant":         "plant",
	"washer":        "washer",
	"pantry":        "dish_shelf",
	"barrel":        "barrel",
	"bathtub":       "bathtub",
	"toilet":        "toilet",
	"sink":          "sink",
	"desk":          "desk",
	"candle":        "candle",
	"carpet":        "carpet",
	"rubble":        "rubble",
	"debris":        "debris",
	// 散らかりの小物。家具の脇に溜まる物。SpawnProp が要るので item でなく実在の prop へ写す。仮画像でない
	// 実スプライトのある prop だけを使う。食器・写真などの拾える item を家具の上へ載せるのは item spawn 経路が
	// 要るので今後
	"laundry": "laundry",
	"crate":   "crate",
	// 外皮 FacadePass。街路側の前壁に付ける窓・シャッターと店の看板。窓とシャッターは仮画像
	"window":  "window",
	"shutter": "shutter",
	"sign":    "wooden_sign",
	// lot pass。敷地を囲う塀と前庭の外構。塀は仮画像、自販機は実物
	"fence":   "fence",
	"vending": "vending_machine",
	// hero 部屋の目玉。稀な1棟の主室に据える landmark。すべて実在の prop
	"chest":      "wood_chest",
	"torii":      "torii_small",
	"phonograph": "phonograph",
	"komainu":    "komainu",
	"pillar":     "stone_pillar",
	"cage":       "cage",
	"poison":     "毒樽",
	// 和家具。民家の believability を上げる。仮画像。raw 名は日本語
	"butsudan": "仏壇",
	"kamidana": "神棚",
	"getabako": "下駄箱",
}

// PropRawName は Ref に対応する raw prop 名と、それが in-game に spawn されるかを返す。overworld の spawn と
// VRT の描画がこの1関数を共有し、片方だけが描く/置くという乖離が構造的に起きないようにする。
func PropRawName(ref string) (string, bool) {
	name, ok := propRaw[ref]
	return name, ok
}

// PropRaws は Ref→raw prop 名の写像を返す。写像先の raw が実在するかを検査するテストが全対を舐めるのに使う。
// 返す map は書き換えない前提。
func PropRaws() map[string]string {
	return propRaw
}
