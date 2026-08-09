package interior

// lootRef は content_catalog が使う抽象 loot 語彙。lootRaw の写像元。
// rawItemGroupID は raw の item group の id。lootRaw の写像先。どちらも実体は string だが、名前で写像の
// 向き、すなわち Ref から group id へ、を型に表す。
type (
	lootRef        = string
	rawItemGroupID = string
)

// loot の写像。interior の抽象 loot Ref をゲームの raw item group へ写す単一のソース。overworld の床 loot が
// この表で item group を引いてアイテムを抽選し、VRT も同じ表で loot の有無を描く。両者が同じ表を引くので
// VRT と in-game が乖離しない。表に無い Ref は spawn されず描かれない。prop の propRaw と対称に保つ。
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

// LootGroups は loot Ref から raw item group 名への写像を返す。写像先の group が実在するかを検査するテストが
// 全対を舐めるのに使う。返す map は書き換えない前提。
func LootGroups() map[string]string {
	return lootRaw
}
