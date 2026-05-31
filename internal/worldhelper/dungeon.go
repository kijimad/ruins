package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetDungeon はシングルトンエンティティからDungeonを取得する。
// ダンジョン未開始の場合はnilを返す
// TODO: シングルトンに対する操作系は共通化したほうがよさそう
func GetDungeon(world w.World) *gc.Dungeon {
	data := world.Components.DungeonState.Get(world.Resources.SingletonEntity)
	if data == nil {
		return nil
	}
	return data.(*gc.Dungeon)
}

// SetDungeon はシングルトンエンティティにDungeonを設定する
func SetDungeon(world w.World, dungeon *gc.Dungeon) {
	world.Resources.SingletonEntity.AddComponent(world.Components.DungeonState, dungeon)
}
