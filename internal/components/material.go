package components

import (
	"github.com/kijimaD/ruins/internal/oapi"
)

// Material はアイテムの材質を表す。可燃性と燃焼熱量の算出根拠で、生成時に raw から付く。
// Kind は raw の Material enum の値。金属や石など不燃の材質も保持し、
// 観察メニューで材質を見せて、なぜ燃える/燃えないかを読み取れるようにする。
type Material struct {
	Kind oapi.Material // 材質の種類。WOOD・METAL・FOOD など。enum で比較のタイプミスを型が弾く
}
