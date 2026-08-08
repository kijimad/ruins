package query

import (
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// GetEntityName はエンティティの表示名を取得する
func GetEntityName(entity ecs.Entity, world w.World) string {
	if !world.ECS.Alive(entity) || !world.Components.Name.Has(entity) {
		return "Unknown"
	}
	return world.Components.Name.Get(entity).Name
}

// GetEntityID はエンティティの同定キーを返す。raw 定義やレシピ参照との照合に使う。
// 表示名ではなく英語の id を返す。RawID コンポーネントを持たなければ空文字を返す
func GetEntityID(entity ecs.Entity, world w.World) string {
	if !world.ECS.Alive(entity) || !world.Components.RawID.Has(entity) {
		return ""
	}
	return world.Components.RawID.Get(entity).ID
}

// NameMarkup はエンティティ種別に応じたタグで名前を包んだマークアップ文字列を返す。
// Player=<player>、NPC=<npc>、その他は裸のテキスト。gamelog.Markup へ書式の引数として渡す。
// 名前の色は実行時の種別で決まるため、書式文字列でなくここでタグ付けする
func NameMarkup(entity ecs.Entity, name string, world w.World) string {
	switch {
	case world.Components.Player.Has(entity):
		return gamelog.Tag("player", name)
	case world.Components.SoloAI.Has(entity) || world.Components.SquadAI.Has(entity):
		return gamelog.Tag("npc", name)
	default:
		return name
	}
}
