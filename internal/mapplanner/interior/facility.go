package interior

// 施設種別の入口とノブ。単室の建物を施設種別で内装する Furnish と、施設ごとの content 変種・密度・経年を
// 決める関数を持つ。content レシピそのものは content_catalog.go、束什器は fixtures.go、多部屋の加工パイプは
// furnish.go にある。ここは「どの施設をどの配合・密度・経年で furnish するか」の施設レベルの判断に絞る。

// 施設種別名。overworld の facilityType の文字列と揃える。switch の case で繰り返すので定数にする。
const (
	facHouse   = "house"
	facStore   = "store"
	facAntique = "antique"
	facClinic  = "clinic"
	facLab     = "lab"
)

// Furnish は建物の footprint と入口から、施設種別に応じた内装の配置を決定的に返す。footprint を外周が壁の
// 1部屋とみなし、door はその外周上の入口。多部屋の敷地計画は FurnishBuilding が担い、Furnish は単室で
// facilityContent の変種・密度・経年・flavor の直交軸を検証する単位になる。未知の施設種別は汎用の内装。
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

// facilityContent は施設種別名から内装 content を1つ引く。同じ施設種別でも複数の変種を持ち、seed で
// 引くことで同じ店が薬局にも食料品店にもなる。doc L694 の最優先「部屋アーキタイプ数」を、既存家具の
// 組み替えだけでデータを足さずに増やす。変種の seed は本体生成と別枠にして相関を避ける。
func facilityContent(facility string, seed uint64) Content {
	variants := facilityVariants(facility)
	c := variants[int(childSeed(seed, 9_000_000)%uint64(len(variants)))]
	return scaleDensity(c, seed)
}

// facilityVariants は施設種別ごとの内装変種の一覧。骨董品店は商店、研究施設は診療所へ寄せ、未知は汎用に
// する。変種を足すときはここへ Content を加えるだけでよい。レシピの実体は content_catalog.go にある。
func facilityVariants(facility string) []Content {
	switch facility {
	case facHouse:
		return []Content{houseContent(), studioContent()}
	case facStore:
		return []Content{storeContent(), pharmacyContent(), groceryContent()}
	case facAntique:
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
