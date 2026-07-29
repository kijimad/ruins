package interior

// 施設種別の入口とノブ。単室の建物を施設種別で内装する Furnish と、施設ごとの content 変種・密度・経年を
// 決める関数を持つ。content レシピそのものは content_catalog.go、束什器は fixtures.go、多部屋の加工パイプは
// furnish.go にある。ここは「どの施設をどの配合・密度・経年で furnish するか」の施設レベルの判断に絞る。

// FacilityKind は建物の施設種別。overworld の facilityType の文字列と揃える。公開 API は overworld から
// 素の string を受けて境界で FacilityKind へ変換し、内部はこの型で扱う。role など他の文字列と取り違えると
// コンパイルが通らないよう型で区別する。
type FacilityKind string

// 施設種別名。switch の case で繰り返すので定数にする。overworld の facilityType の値と1対1で揃える。
const (
	facHouse   FacilityKind = "house"
	facStore   FacilityKind = "store"
	facAntique FacilityKind = "antique"
	facClinic  FacilityKind = "clinic"
	facLab     FacilityKind = "lab"
	facOffice  FacilityKind = "office"
	facDepot   FacilityKind = "depot"
)

// Furnish は建物の footprint と入口から、施設種別に応じた内装の配置を決定的に返す。footprint を外周が壁の
// 1部屋とみなし、door はその外周上の入口。多部屋の敷地計画は FurnishBuilding が担い、Furnish は単室で
// facilityContent の変種・密度・経年・flavor の直交軸を検証する単位になる。未知の施設種別は汎用の内装。
func Furnish(seed uint64, footprint Rect, door Vec, facility FacilityKind) []Placed {
	prof := rollProfile(seed)
	room := Room{Rect: footprint, Doorways: []Doorway{{X: door.X, Y: door.Y}}}
	placed := FillRoom(seed, room, applyDensity(facilityContent(facility, seed), prof.density))
	// 時間の層。損傷レベルで略奪・生活痕・廃墟化の強度を変える。無傷の建物は新品のまま
	placed = Age(seed, room, placed, prof.damage)
	// 家具の隙間へ flavor machine を1つ置き、戦利品の無い空き箱部屋に character を与える
	placed = Flavor(seed, room, placed, facilityFlavor(facility))
	// 散らかりの小物を家具の隣へ落とし、生活感を足す
	return applyClutter(childSeed(seed, 11_300_000), room, placed, prof.clutter, roleMain)
}

// facilityContent は施設種別名から内装 content を1つ引く。同じ施設種別でも複数の変種を持ち、seed で
// 引くことで同じ店が薬局にも食料品店にもなる。最優先の「部屋アーキタイプ数」を、既存家具の
// 組み替えだけでデータを足さずに増やす。変種の seed は本体生成と別枠にして相関を避ける。
func facilityContent(facility FacilityKind, seed uint64) Content {
	variants := facilityVariants(facility)
	return variants[int(childSeed(seed, 9_000_000)%uint64(len(variants)))]
}

// facilityVariants は施設種別ごとの内装変種の一覧。骨董品店は商店、研究施設は診療所へ寄せ、未知は汎用に
// する。変種を足すときはここへ Content を加えるだけでよい。レシピの実体は content_catalog.go にある。
func facilityVariants(facility FacilityKind) []Content {
	switch facility {
	case facHouse:
		return []Content{houseContent(), studioContent()}
	case facStore:
		return []Content{storeContent(), pharmacyContent(), groceryContent()}
	case facAntique:
		return []Content{storeContent()}
	case facClinic, facLab:
		return []Content{clinicContent()}
	case facOffice:
		return []Content{officeContent()}
	case facDepot:
		return []Content{depotContent()}
	}
	// FacilityKind は raw 由来の文字列なので未知値が来うる。既知の全種別を case で網羅しつつ、未知は末尾で
	// 汎用へ落とす。default を置かないことで、種別を増やして case を足し忘れると exhaustive linter が止める。
	return []Content{genericContent()}
}

// applyDensity は content の家具量を密度係数 factor(×/10)で増減する。個数1の必須什器は1を保ち、詰め物の
// 棚だけが増減する。密度は buildingProfile が建物ごとに引く直交軸で、同じ内装でもがらんとした店と品で
// 埋まった店を出し分ける。factor==10 は等倍で素通し。
func applyDensity(c Content, factor int) Content {
	if factor == 10 {
		return c
	}
	for gi := range c.Groups {
		for ii := range c.Groups[gi].Items {
			it := &c.Groups[gi].Items[ii]
			if it.Kind != KindFurniture {
				continue // 家具だけ密度を変える。戦利品・装飾はそのまま
			}
			it.Amount.Base = scaleAmount(it.Amount.Base, factor)
			it.Amount.Bonus = scaleAmount(it.Amount.Bonus, factor)
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
