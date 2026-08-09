package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// PushBehavior は BehaviorPush の実装。プレイヤーが隣接する Pushable キューブを押し方向へ
// 1タイル動かす。分解などと同じ多ターン行動で、専用の進捗コンポーネントを持たず
// gc.Activity の Progress にパーティの押し力を累積して進捗を表す。重いキューブほど
// 必要な総コストが増え、パーティAPが多いほど速く満ちる。
//
// 着手時のパラメータである押す対象キューブと押し向きは NewPushActivity が受け取り、
// 押し先を求めて gc.Activity の PlaceParams へ書き込む。継続処理は状態を持たない
// ゼロ値インスタンスで回るので、着手後は gc.Activity の PlaceParams だけを読む。
type PushBehavior struct{}

// Info はBehaviorの実装
func (pb *PushBehavior) Info() Info {
	return Info{
		Name:            "Push",
		Description:     "Push an adjacent cube to move it",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		// 必要総APはキューブの総重量に応じて buildCubeMove が算出するため、Info では持たず 0 とする
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (pb *PushBehavior) Name() gc.BehaviorName {
	return gc.BehaviorPush
}

// NewPushActivity は押すキューブと向きを指定して押しアクティビティを組む。
func NewPushActivity(cube ecs.Entity, dir gc.Direction, world w.World) (*gc.Activity, error) {
	if !world.Components.GridElement.Has(cube) {
		return nil, fmt.Errorf("push target has no position")
	}
	cubeCoord := world.Components.GridElement.Get(cube).Coord
	dest := cubeCoord.Add(dir.GetDelta())
	return buildCubeMove(gc.BehaviorPush, cube, dest, world)
}

// Validate はBehaviorの実装
func (pb *PushBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) (string, error) {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return "", fmt.Errorf("push target is not set")
	}
	if !world.ECS.Alive(p.Target) {
		return query.T(world, "push target does not exist"), nil
	}
	if !world.Components.Pushable.Has(p.Target) {
		return query.T(world, "target cannot be pushed"), nil
	}
	if !world.Components.GridElement.Has(p.Target) {
		return query.T(world, "push target has no position"), nil
	}
	if !world.Components.GridElement.Has(actor) {
		// 押し手の位置欠落は不変条件違反
		return "", fmt.Errorf("pusher has no position")
	}
	// 押せる先はプレイヤーが行ける先に一致させる。CanMoveTo が寒波前線の破棄域や
	// 壁を弾くので、押し専用の前線チェックは持たない
	cubeCoord := world.Components.GridElement.Get(p.Target).Coord
	if !CanMoveTo(world, p.Destination.Coord, cubeCoord, p.Target) {
		return query.T(world, "cannot push in that direction"), nil
	}
	return "", nil
}

// Start はBehaviorの実装
func (pb *PushBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("push started", "actor", actor)
	return nil
}

// DoTurn はBehaviorの実装。毎ターン対象の生存と押し先の通行可否を確かめ、ターンを1つ消費する。
func (pb *PushBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok || !world.ECS.Alive(p.Target) {
		Cancel(comp, "interrupted because the push target disappeared")
		return nil
	}
	if !world.Components.GridElement.Has(p.Target) {
		Cancel(comp, "interrupted because it can no longer be pushed")
		return nil
	}
	cubeCoord := world.Components.GridElement.Get(p.Target).Coord
	if !CanMoveTo(world, p.Destination.Coord, cubeCoord, p.Target) {
		Cancel(comp, "can no longer push in that direction")
		return nil
	}

	// パーティの押し力を注ぐ。強いパーティほど速く動かす
	comp.Progress.Current += query.PartyPushPower(world)
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}
	return nil
}

// Finish はBehaviorの実装。キューブだけを1タイル進める。押し手は追随させない。
// プレイヤーは次の移動入力で空いたタイルへ普通に一歩進む。押しと移動を別アクティビティに分け、
// それぞれが自然な通貨、押しはターン、移動はAPで課金される。
func (pb *PushBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return nil
	}
	cube := p.Target
	if !world.ECS.Alive(cube) || !world.Components.GridElement.Has(cube) {
		return nil
	}
	world.Components.GridElement.Get(cube).Coord = p.Destination.Coord

	// キューブは BlockPass なので通行索引が変わる。全再構築で確実に反映する
	query.InvalidateSpatialIndex(world)

	log.Debug("push finished", "actor", actor, "cube", cube, "to", p.Destination.String())
	return nil
}

// Canceled はBehaviorの実装
func (pb *PushBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("push canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// cubePushCost はキューブを1タイル動かすのに要する総コストを返す。総重量が重いほど増える。
// 押しと引きで共通。APが無ければエラー。毎ターン注ぐAPは DoTurn でパーティの押し力から
// 再計算するため、着手時に凍結するのは総コストだけにする。
//
// 総重量は内部ステージに置いた物の総和。内部はオーバーワールドと同じく単一の永続ステージなので、
// 固定キー NewCubeInteriorStage で直接引く。どのキューブを押しても同じ内部の総重量が押しの重さに
// なる。内部をキューブごとに分ける拡張へ進むときは、キューブから内部へのリンクをここで解決し直す。
func cubePushCost(world w.World) (int, error) {
	if query.PartyPushPower(world) <= 0 {
		return 0, fmt.Errorf("not enough AP to move it")
	}
	return query.PushCost(query.CubeWeight(world, gc.NewCubeInteriorStage())), nil
}

// buildCubeMove は押しと引きで共通の gc.Activity を組む。総コストは総重量で決まり、
// 押し引きで違うのは対象キューブと移動先だけなので、それを引数で受ける。
func buildCubeMove(name gc.BehaviorName, cube ecs.Entity, dest consts.Coord[consts.Tile], world w.World) (*gc.Activity, error) {
	required, err := cubePushCost(world)
	if err != nil {
		return nil, err
	}
	comp := NewActivity(name, required)
	comp.Params = &gc.PlaceParams{Target: cube, Destination: gc.GridElement{Coord: dest}}
	return comp, nil
}

// PullBehavior は BehaviorPull の実装。プレイヤーが隣接する Pushable キューブを自分の側へ引き、
// 自分は1タイル後退する。押しでは動かせない壁際・角のキューブを引き出して詰みを解く。
// 総コストは押しと同じく総重量で決まる。着手時のパラメータである引く対象キューブは
// NewPullActivity が受け取り、gc.Activity の PlaceParams へ書き込む。継続処理は
// 状態を持たないゼロ値インスタンスで回るので、着手後は gc.Activity 側だけを読む。
type PullBehavior struct{}

// Info はBehaviorの実装
func (pb *PullBehavior) Info() Info {
	return Info{
		Name:            "Pull",
		Description:     "Pull an adjacent cube toward yourself",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		// 必要総APはキューブの総重量に応じて buildCubeMove が算出するため、Info では持たず 0 とする
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (pb *PullBehavior) Name() gc.BehaviorName { return gc.BehaviorPull }

// pullRetreat は引きでプレイヤーが後退する先を返す。プレイヤーはキューブの隣に立ち、
// キューブから自分へ向かう向きへ1つ退く。キューブはプレイヤーの元タイルへ入る。
func pullRetreat(cubeCoord, playerCoord consts.Coord[consts.Tile]) consts.Coord[consts.Tile] {
	delta := playerCoord.Sub(cubeCoord)
	return playerCoord.Add(delta)
}

// canPullCube はプレイヤーがキューブを引けるかを返す。後退先が通行可能なら引ける。
// アクション実行前の可否判定に使い、引けないときは致命エラーでなく no-op にできるようにする。
func canPullCube(world w.World, actor, cube ecs.Entity) bool {
	if !world.Components.GridElement.Has(actor) || !world.Components.GridElement.Has(cube) {
		return false
	}
	playerCoord := world.Components.GridElement.Get(actor).Coord
	cubeCoord := world.Components.GridElement.Get(cube).Coord
	retreat := pullRetreat(cubeCoord, playerCoord)
	return CanMoveTo(world, retreat, playerCoord, actor)
}

// NewPullActivity は引くキューブと引き手を指定して引きアクティビティを組む。
func NewPullActivity(cube, actor ecs.Entity, world w.World) (*gc.Activity, error) {
	if !world.Components.GridElement.Has(cube) {
		return nil, fmt.Errorf("pull target has no position")
	}
	if !world.Components.GridElement.Has(actor) {
		return nil, fmt.Errorf("puller has no position")
	}
	// キューブはプレイヤーの立っているタイルへ入る。プレイヤーはそのぶん後退する
	dest := world.Components.GridElement.Get(actor).Coord
	return buildCubeMove(gc.BehaviorPull, cube, dest, world)
}

// Validate はBehaviorの実装。後退先が通行可能であることを確かめる。
func (pb *PullBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) (string, error) {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return "", fmt.Errorf("pull target is not set")
	}
	if !world.ECS.Alive(p.Target) {
		return query.T(world, "pull target does not exist"), nil
	}
	if !world.Components.Pushable.Has(p.Target) {
		return query.T(world, "target cannot be pulled"), nil
	}
	if !world.Components.GridElement.Has(p.Target) {
		return query.T(world, "pull target has no position"), nil
	}
	if !world.Components.GridElement.Has(actor) {
		// 引き手の位置欠落は不変条件違反
		return "", fmt.Errorf("puller has no position")
	}
	cubeCoord := world.Components.GridElement.Get(p.Target).Coord
	retreat := pullRetreat(cubeCoord, p.Destination.Coord)
	// 後退先がプレイヤーの行ける先であること。キューブの入る先はプレイヤーが退いて空く
	if !CanMoveTo(world, retreat, p.Destination.Coord, actor) {
		return query.T(world, "no space to pull"), nil
	}
	return "", nil
}

// Start はBehaviorの実装
func (pb *PullBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("pull started", "actor", actor)
	return nil
}

// DoTurn はBehaviorの実装。毎ターン対象の生存と後退先の通行可否を確かめ、ターンを1つ消費する。
func (pb *PullBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok || !world.ECS.Alive(p.Target) {
		Cancel(comp, "interrupted because the pull target disappeared")
		return nil
	}
	if !world.Components.GridElement.Has(p.Target) || !world.Components.GridElement.Has(actor) {
		Cancel(comp, "interrupted because it can no longer be pulled")
		return nil
	}
	cubeCoord := world.Components.GridElement.Get(p.Target).Coord
	retreat := pullRetreat(cubeCoord, p.Destination.Coord)
	if !CanMoveTo(world, retreat, p.Destination.Coord, actor) {
		Cancel(comp, "ran out of space to pull")
		return nil
	}

	// パーティの押し力を注ぐ。強いパーティほど速く動かす
	comp.Progress.Current += query.PartyPushPower(world)
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}
	return nil
}

// Finish はBehaviorの実装。キューブをプレイヤーの元タイルへ引き入れ、プレイヤーは1タイル後退する。
func (pb *PullBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return nil
	}
	cube := p.Target
	if !world.ECS.Alive(cube) || !world.Components.GridElement.Has(cube) || !world.Components.GridElement.Has(actor) {
		return nil
	}
	cubeGrid := world.Components.GridElement.Get(cube)
	cubeOld := cubeGrid.Coord
	retreat := pullRetreat(cubeOld, p.Destination.Coord)

	// 先にプレイヤーを後退させてタイルを空け、そこへキューブを引き入れる
	world.Components.GridElement.Get(actor).Coord = retreat
	cubeGrid.Coord = p.Destination.Coord

	// キューブは BlockPass なので通行索引が変わる。全再構築で確実に反映する
	query.InvalidateSpatialIndex(world)

	log.Debug("pull finished", "actor", actor, "cube", cube, "to", p.Destination.String())
	return nil
}

// Canceled はBehaviorの実装
func (pb *PullBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("pull canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}
