package interior

import "github.com/kijimaD/ruins/internal/rawschema"

// 敷地の類型。1チャンク=1建物なので、街区の短冊分割でなく建物ごとに敷地の性格を1つ引く。区画分割はせず
// 塀・門・前庭の出し分けだけで性格を作る。ここでは前庭の深さ・塀・外構の出し分けで、
// 戸建・商店街・ロードサイドの見えの違いを作る。類型の重みと前庭の奥行きは raw の interiorProfile.sites が持つ。
// 旗竿地・団地は多チャンクの配棟が要るので今後。

// rollSiteType は施設種別と seed から敷地類型を1つ引く。民家と店でそれぞれの重みを raw の sites の行の順に積み、
// 街路に面した店と郊外の駐車場付き店を出し分ける。重みの和が1以上であることは読み込み時に確かめる。
func rollSiteType(t profileTable, fac rawschema.Facility, seed uint64) rawschema.SiteChoice {
	weight := func(c rawschema.SiteChoice) int {
		if fac.IsShop {
			return int(c.ShopWeight)
		}
		return int(c.HomeWeight)
	}
	total := 0
	for _, c := range t.sites {
		total += weight(c)
	}
	roll := int(childSeed(seed, 12_000_000) % uint64(total))
	for _, c := range t.sites {
		roll -= weight(c)
		if roll < 0 {
			return c
		}
	}
	return t.sites[len(t.sites)-1]
}
