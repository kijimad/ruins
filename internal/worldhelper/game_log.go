package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetGameLog はシングルトンエンティティからGameLogストアを取得する
func GetGameLog(world w.World) *gamelog.SafeSlice {
	return world.Components.GameLog.Get(world.Resources.Singleton).(*gc.GameLog).Store
}
