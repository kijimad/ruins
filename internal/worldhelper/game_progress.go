package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetGameProgress はシングルトンエンティティからGameProgressを取得する
func GetGameProgress(world w.World) *gc.GameProgress {
	data := world.Components.GameProgress.Get(world.Resources.SingletonEntity)
	if data == nil {
		return nil
	}
	return data.(*gc.GameProgress)
}
