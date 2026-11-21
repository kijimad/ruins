package activity

import (
	"errors"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// useEffect は使用時に発動する効果1種。アイテムがその効果 component を持つときだけ働く。
// 効果を1つ足すときは useEffect を実装して eatEffects か useEffects へ1行足し、効果の欄を schema.cue の consumable を必須にする一覧にも足す。
// DoTurn/Validate の本体は触らない
type useEffect interface {
	// present はアイテムがこの効果を持つかを返す
	present(world w.World, item ecs.Entity) bool
	// check は使う前の検証。使えない理由があれば *UserError を返す。既定は問題なし
	check(u *ConsumeBehavior, actor, item ecs.Entity, world w.World) error
	// apply は効果を適用する
	apply(u *ConsumeBehavior, comp *gc.Activity, actor, item ecs.Entity, world w.World) error
}

// eatEffects は口に入れて体の内側に働く効果の一覧。順に present を見て、持つ効果だけ適用する
var eatEffects = []useEffect{nutritionEffect{}}

// useEffects は手当てや道具として外から働く効果の一覧。HP の回復は、飲む回復薬も手当てとしてここに置く
var useEffects = []useEffect{healEffect{}, remedyEffect{}, escapeEffect{}, cubeCoreEffect{}}

// healEffect は HP を回復する
type healEffect struct{}

func (healEffect) present(world w.World, item ecs.Entity) bool {
	return world.Components.ProvidesHealing.Has(item)
}

func (healEffect) check(_ *ConsumeBehavior, _, _ ecs.Entity, _ w.World) error { return nil }

func (healEffect) apply(u *ConsumeBehavior, comp *gc.Activity, actor, item ecs.Entity, world w.World) error {
	return u.applyHealing(comp, actor, world, world.Components.ProvidesHealing.Get(item), item)
}

// nutritionEffect は空腹度を回復する
type nutritionEffect struct{}

func (nutritionEffect) present(world w.World, item ecs.Entity) bool {
	return world.Components.ProvidesNutrition.Has(item)
}

// check は満腹度が上限なら食べさせない
func (nutritionEffect) check(_ *ConsumeBehavior, actor, _ ecs.Entity, world w.World) error {
	if !world.Components.Hunger.Has(actor) {
		return nil
	}
	if h := world.Components.Hunger.Get(actor); h.Current >= h.Max {
		return &UserError{Msg: query.T(world, "Too full to eat")}
	}
	return nil
}

func (nutritionEffect) apply(u *ConsumeBehavior, comp *gc.Activity, actor, item ecs.Entity, world w.World) error {
	return u.applyNutrition(comp, actor, world, world.Components.ProvidesNutrition.Get(item).Amount, item)
}

// remedyEffect は不調を治療する
type remedyEffect struct{}

func (remedyEffect) present(world w.World, item ecs.Entity) bool {
	return world.Components.Remedy.Has(item)
}

// check は治療専用アイテムが治せる不調を今持たないなら使用を弾く。空振りを使う前に止める。
// 併用効果を持つアイテムは他の効果で使えるので弾かない
func (remedyEffect) check(u *ConsumeBehavior, actor, item ecs.Entity, world w.World) error {
	if !u.isRemedyOnly(world, item) {
		return nil
	}
	remedy := world.Components.Remedy.Get(item)
	if !world.Components.HealthStatus.Has(actor) {
		return &UserError{Msg: query.T(world, "Nothing to treat.")}
	}
	if _, c := remedyTarget(world.Components.HealthStatus.Get(actor), remedy); c == nil {
		return &UserError{Msg: query.T(world, "Nothing to treat.")}
	}
	return nil
}

func (remedyEffect) apply(u *ConsumeBehavior, _ *gc.Activity, actor, item ecs.Entity, world w.World) error {
	u.applyRemedy(actor, world, world.Components.Remedy.Get(item), item)
	return nil
}

// escapeEffect は惑星を脱出してクリアの画面へ移る
type escapeEffect struct{}

func (escapeEffect) present(world w.World, item ecs.Entity) bool {
	return world.Components.EscapeDevice.Has(item)
}

func (escapeEffect) check(_ *ConsumeBehavior, _, _ ecs.Entity, _ w.World) error { return nil }

func (escapeEffect) apply(_ *ConsumeBehavior, _ *gc.Activity, _, _ ecs.Entity, world w.World) error {
	return lifecycle.RequestStateChange(world, gc.EscapeEvent())
}

// cubeCoreEffect は使用者の隣に新しいキューブを組み上げる。キューブはいくつあってもよい
type cubeCoreEffect struct{}

func (cubeCoreEffect) present(world w.World, item ecs.Entity) bool {
	return world.Components.CubeCore.Has(item)
}

func (cubeCoreEffect) check(_ *ConsumeBehavior, actor, _ ecs.Entity, world w.World) error {
	if !query.IsOnOverworld(world) {
		return &UserError{Msg: query.T(world, "The cube can only be assembled outdoors.")}
	}
	if _, ok := cubeAssemblyTile(world, actor); !ok {
		return &UserError{Msg: query.T(world, "There is no room to assemble the cube.")}
	}
	return nil
}

func (cubeCoreEffect) apply(_ *ConsumeBehavior, _ *gc.Activity, actor, _ ecs.Entity, world w.World) error {
	pos, ok := cubeAssemblyTile(world, actor)
	if !ok {
		return errors.New("no tile to assemble the cube")
	}
	if _, err := lifecycle.SpawnCube(world, pos); err != nil {
		return err
	}
	query.InvalidateSpatialIndex(world)
	gamelog.New(query.GetGameLog(world)).Markup(query.T(world, "A new cube has been assembled.")).Log()
	return nil
}

// cubeAssemblyOffsets は組み上げるマスを探す順。上下左右を先に見て、斜めを後に見る
var cubeAssemblyOffsets = []consts.Coord[consts.Tile]{
	{X: 0, Y: -1}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: -1, Y: 0},
	{X: -1, Y: -1}, {X: 1, Y: -1}, {X: 1, Y: 1}, {X: -1, Y: 1},
}

// cubeAssemblyTile は actor の周りの8マスから、歩いて入れて置物もアイテムも無い最初のマスを返す。
// 遺跡の入口や落ちている物の上に組み上げると、キューブに塞がれて触れなくなる
func cubeAssemblyTile(world w.World, actor ecs.Entity) (consts.Coord[consts.Tile], bool) {
	occupied := map[consts.Coord[consts.Tile]]bool{}
	q := query.ActiveFilter2[gc.GridElement, gc.LocationOnField](world).Query()
	for q.Next() {
		occupied[world.Components.GridElement.Get(q.Entity()).Coord] = true
	}
	from := world.Components.GridElement.Get(actor).Coord
	for _, d := range cubeAssemblyOffsets {
		to := from.Add(d)
		if !occupied[to] && CanMoveTo(world, to, from, actor) {
			return to, true
		}
	}
	return consts.Coord[consts.Tile]{}, false
}
