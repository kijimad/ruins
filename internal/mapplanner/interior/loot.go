package interior

// lootRef と rawItemGroupID は lootRaw のキーと値の型。実体は string だが、名前を付けて意味を持たせる。
type (
	lootRef        = string
	rawItemGroupID = string
)

// lootRaw は loot の対応表。キーは content_catalog が使う抽象 loot Ref、値は raw の item group id。overworld の
// 床 loot がこの表で item group を引いてアイテムを抽選し、VRT も同じ表で loot の有無を描く。両者が同じ表を
// 引くので VRT と in-game が乖離しない。表に無い Ref は spawn されず描かれない。prop の propRaw と対称に保つ。
var lootRaw = map[lootRef]rawItemGroupID{
	"snacks":    "food",
	"drinks":    "food",
	"bento":     "food",
	"meds":      "healing_item",
	"bandage":   "healing_item",
	"documents": "scrap_of_paper", // 事務所の散らばった書類
	"supplies":  "materials",      // 倉庫の保管資材
}

// LootGroupName は loot Ref に対応する raw item group 名と、spawn されるかを返す。overworld の床 loot spawn と
// VRT の描画がこの1関数を共有し、片方だけが置く、描くという乖離が構造的に起きないようにする。
func LootGroupName(ref string) (string, bool) {
	name, ok := lootRaw[ref]
	return name, ok
}

// LootGroups は loot Ref から raw item group id への対応表を返す。値の group が実在するかを検査するテストが
// 全対を舐めるのに使う。返す map は書き換えない前提。
func LootGroups() map[string]string {
	return lootRaw
}
