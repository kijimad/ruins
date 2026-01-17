package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// FormatItemName はアイテムエンティティから名前と個数を取得してフォーマットする
// 名前はNameコンポーネントから取得し、見つからない場合は "Unknown Item" を返す
// 個数が1以下の場合は名前のみ、2以上の場合は "名前(個数)" の形式で返す
func FormatItemName(world w.World, itemEntity ecs.Entity) string {
	name := "Unknown Item"
	if nameComp := world.Components.Name.Get(itemEntity); nameComp != nil {
		n := nameComp.(*gc.Name)
		name = n.Name
	}

	count := 1
	if itemComp := world.Components.Item.Get(itemEntity); itemComp != nil {
		item := itemComp.(*gc.Item)
		count = item.Count
	}

	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s(%d個)", name, count)
}
