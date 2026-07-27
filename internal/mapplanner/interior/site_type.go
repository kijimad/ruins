package interior

// 敷地の類型。1チャンク=1建物なので、街区の短冊分割でなく建物ごとに敷地の性格を1つ引く。区画分割はせず
// 塀・門・前庭の出し分けだけで性格を作る。ここでは前庭の深さ・塀・外構の出し分けで、
// 戸建・商店街・ロードサイドの見えの違いを作る。旗竿地・団地は多チャンクの配棟が要るので今後。

// siteType は敷地の類型。前庭の深さと外構の出し分けに使う。
type siteType int

const (
	siteDetached  siteType = iota // 戸建。塀で囲い前庭に観葉。民家の既定
	siteShopfront                 // 商店街。街路に面して建て前庭は浅く、塀を張らず看板と自販機を出す
	siteRoadside                  // ロードサイド。前庭を広く取り前面を駐車場にし、道路際に自販機
)

// rollSiteType は施設種別と seed から敷地類型を1つ引く。民家は戸建。店は商店街かロードサイドを seed で選び、
// 街路に面した店と郊外の駐車場付き店を出し分ける。
func rollSiteType(facility FacilityKind, seed uint64) siteType {
	if !isShop(facility) {
		return siteDetached
	}
	if childSeed(seed, 12_000_000)%2 == 0 {
		return siteShopfront
	}
	return siteRoadside
}

// frontYardOf は敷地類型ごとの前庭の奥行きを返す。商店街は街路に面して浅く、ロードサイドは駐車場ぶん深い。
func frontYardOf(st siteType) int {
	switch st {
	case siteShopfront:
		return 1
	case siteRoadside:
		return 5
	default: // siteDetached
		return frontYard
	}
}
