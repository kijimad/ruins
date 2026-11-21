package states

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"

	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// throwMode はバックパックの物を投げる狙い方
type throwMode struct {
	item ecs.Entity
	// attack は画面を開いたときに導いた投げの値。画面の間は物が変わらないので毎フレーム導き直さない
	attack gc.ThrowAttack
}

func (throwMode) bindings() []keybind.Binding { return targetingBindings }

func (m throwMode) canTarget(world w.World, player, target ecs.Entity) bool {
	return activity.CanThrowTarget(player, m.item, target, world)
}

func (m throwMode) hitRate(world w.World, player, target ecs.Entity) int {
	return activity.CalculateThrowHitRate(player, target, m.attack, world)
}

func (m throwMode) execute(world w.World, player, target ecs.Entity) error {
	_, err := activity.Execute(activity.NewThrowActivity(m.item, target), player, world)
	return err
}

func (throwMode) handle(_ w.World, _ ecs.Entity, _ inputmapper.ActionID) (bool, error) {
	return false, nil
}

func (throwMode) title(world w.World) string { return query.T(world, "== Throwing Mode ==") }

// drawSource は投げる物の名前と威力を描く
func (m throwMode) drawSource(world w.World, _ ecs.Entity, drawText func(string)) {
	drawText(fmt.Sprintf("%s: %s", query.T(world, "Item"), query.GetEntityName(m.item, world)))
	drawText(fmt.Sprintf("%s: %d", query.T(world, "Attack power"), m.attack.Damage))
}

// unavailable は投げる物を投げられなければ理由を返す
func (m throwMode) unavailable(world w.World, player ecs.Entity) string {
	if !activity.CanThrowItem(player, m.item, world) {
		return query.T(world, "Nothing to throw")
	}
	return ""
}

func (throwMode) noTarget(world w.World) string { return query.T(world, "No throwing target") }

func (throwMode) help(world w.World) string {
	return query.T(world, "Tab: Switch  Enter: Throw  Esc: Back")
}
