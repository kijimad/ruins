package components

// Material はアイテムの材質を表す。可燃性と燃焼熱量の算出根拠で、生成時に raw から付く。
// Kind は raw の Material enum の値をそのまま持つ。金属や石など不燃の材質も保持し、
// 観察メニューで材質を見せて、なぜ燃える/燃えないかを読み取れるようにする。
// 将来クラフトや分解でも使える基礎属性にする。
type Material struct {
	Kind string // raw の Material enum 値。WOOD・METAL・FOOD など
}
