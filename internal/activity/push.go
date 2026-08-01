package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// PushActivity は BehaviorPush の実装。プレイヤーが隣接する Pushable キューブを押し方向へ
// 1タイル動かす。分解などと同じ多ターン行動で、専用の進捗コンポーネントを持たず
// gc.Activity のカウントダウンで進捗を表す。重いキューブほど所要ターンが増え、
// パーティAPが多いほど減る。
//
// Cube と Dir はアクティビティ着手時のパラメータで、継続処理のマップ singleton では
// ゼロ値になる。DoTurn 以降は gc.Activity の Target と Destination だけを読む。
type PushActivity struct {
	Cube ecs.Entity
	Dir  gc.Direction
}

// NewPushActivity は押す対象キューブと押し向きを指定して押しアクティビティを作る。
func NewPushActivity(cube ecs.Entity, dir gc.Direction) *PushActivity {
	return &PushActivity{Cube: cube, Dir: dir}
}

// Info はBehaviorの実装
func (pa *PushActivity) Info() Info {
	return Info{
		Name:            "押す",
		Description:     "隣接するキューブを押して動かす",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (pa *PushActivity) Name() gc.BehaviorName {
	return gc.BehaviorPush
}

// BuildActivity はBehaviorの実装。押し先タイルを求め、総重量とパーティAPから所要ターンを決める。
func (pa *PushActivity) BuildActivity(_ ecs.Entity, world w.World) (*gc.Activity, error) {
	if !world.Components.GridElement.Has(pa.Cube) {
		return nil, fmt.Errorf("押す対象に位置がありません")
	}
	cubeCoord := world.Components.GridElement.Get(pa.Cube).Coord
	dest := cubeCoord.Add(pa.Dir.GetDelta())

	duration, err := cubePushTurns(world, pa.Cube)
	if err != nil {
		return nil, err
	}

	comp, err := NewActivity(pa, duration)
	if err != nil {
		return nil, err
	}
	comp.Target = &pa.Cube
	comp.Destination = &gc.GridElement{Coord: dest}
	return comp, nil
}

// Validate はBehaviorの実装
func (pa *PushActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("押す対象が指定されていません")
	}
	if !world.ECS.Alive(*comp.Target) {
		return fmt.Errorf("押す対象が存在しません")
	}
	if !world.Components.Pushable.Has(*comp.Target) {
		return fmt.Errorf("対象は押せません")
	}
	if !world.Components.GridElement.Has(*comp.Target) {
		return fmt.Errorf("押す対象に位置がありません")
	}
	if comp.Destination == nil {
		return fmt.Errorf("押し先が指定されていません")
	}
	if !world.Components.GridElement.Has(actor) {
		return fmt.Errorf("押し手に位置がありません")
	}
	// 押せる先はプレイヤーが行ける先に一致させる。CanMoveTo が寒波前線の破棄域や
	// 壁を弾くので、押し専用の前線チェックは持たない
	cubeCoord := world.Components.GridElement.Get(*comp.Target).Coord
	if !CanMoveTo(world, comp.Destination.Coord, cubeCoord, *comp.Target) {
		return fmt.Errorf("その方向へは押せません")
	}
	return nil
}

// Start はBehaviorの実装
func (pa *PushActivity) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("押し開始", "actor", actor)
	return nil
}

// DoTurn はBehaviorの実装。毎ターン対象の生存と押し先の通行可否を確かめ、ターンを1つ消費する。
func (pa *PushActivity) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil || !world.ECS.Alive(*comp.Target) {
		Cancel(comp, "押す対象が消えたため中断")
		return nil
	}
	if comp.Destination == nil || !world.Components.GridElement.Has(*comp.Target) {
		Cancel(comp, "押せなくなったため中断")
		return nil
	}
	cubeCoord := world.Components.GridElement.Get(*comp.Target).Coord
	if !CanMoveTo(world, comp.Destination.Coord, cubeCoord, *comp.Target) {
		Cancel(comp, "その方向へは押せなくなった")
		return nil
	}

	comp.TurnsLeft--
	if comp.TurnsLeft <= 0 {
		Complete(comp)
	}
	return nil
}

// Finish はBehaviorの実装。キューブを1タイル進め、空いたタイルへ押し手も追随する。
func (pa *PushActivity) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	cube := *comp.Target
	if !world.ECS.Alive(cube) || !world.Components.GridElement.Has(cube) {
		return nil
	}
	cubeGrid := world.Components.GridElement.Get(cube)
	cubeOld := cubeGrid.Coord
	cubeGrid.Coord = comp.Destination.Coord

	// 押し手はキューブが空けたタイルへ前進する
	if world.Components.GridElement.Has(actor) {
		world.Components.GridElement.Get(actor).Coord = cubeOld
	}

	// キューブは BlockPass なので通行索引が変わる。全再構築で確実に反映する
	query.InvalidateSpatialIndex(world)

	log.Debug("押し完了", "actor", actor, "cube", cube, "to", comp.Destination.Coord.String())
	return nil
}

// Canceled はBehaviorの実装
func (pa *PushActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("押しキャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// cubeInteriorWeight はキューブの内装ステージの総重量を返す。内装リンク PortalConnection を持たなければ0。
func cubeInteriorWeight(world w.World, cube ecs.Entity) consts.Milligram {
	if !world.Components.PortalConnection.Has(cube) {
		return 0
	}
	return query.CubeWeight(world, world.Components.PortalConnection.Get(cube).Stage)
}

// cubePushTurns はキューブを1タイル動かすのに要するターン数を返す。総重量が重いほど、
// パーティAPが少ないほど増える。押しと引きで共通。APが無ければエラー。
func cubePushTurns(world w.World, cube ecs.Entity) (consts.Turn, error) {
	power := query.PartyPushPower(world)
	if power <= 0 {
		return 0, fmt.Errorf("APが足りず動かせません")
	}
	cost := query.PushCost(cubeInteriorWeight(world, cube))
	return consts.Turn((cost + power - 1) / power), nil
}

// PullActivity は BehaviorPull の実装。プレイヤーが隣接する Pushable キューブを自分の側へ引き、
// 自分は1タイル後退する。押しでは動かせない壁際・角のキューブを引き出して詰みを解く。
// 所要ターンは押しと同じく総重量とパーティAPで決まる。Cube は着手時のパラメータで、継続処理の
// singleton ではゼロ値。DoTurn 以降は gc.Activity の Target と Destination だけを読む。
type PullActivity struct {
	Cube ecs.Entity
}

// NewPullActivity は引く対象キューブを指定して引きアクティビティを作る。
func NewPullActivity(cube ecs.Entity) *PullActivity {
	return &PullActivity{Cube: cube}
}

// Info はBehaviorの実装
func (pa *PullActivity) Info() Info {
	return Info{
		Name:            "引く",
		Description:     "隣接するキューブを自分の側へ引く",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (pa *PullActivity) Name() gc.BehaviorName { return gc.BehaviorPull }

// pullRetreat は引きでプレイヤーが後退する先を返す。プレイヤーはキューブの隣に立ち、
// キューブから自分へ向かう向きへ1つ退く。キューブはプレイヤーの元タイルへ入る。
func pullRetreat(cubeCoord, playerCoord consts.Coord[consts.Tile]) consts.Coord[consts.Tile] {
	delta := playerCoord.Sub(cubeCoord)
	return playerCoord.Add(delta)
}

// BuildActivity はBehaviorの実装。キューブの移動先はプレイヤーの現在タイル、所要ターンは重量で決まる。
func (pa *PullActivity) BuildActivity(actor ecs.Entity, world w.World) (*gc.Activity, error) {
	if !world.Components.GridElement.Has(pa.Cube) {
		return nil, fmt.Errorf("引く対象に位置がありません")
	}
	if !world.Components.GridElement.Has(actor) {
		return nil, fmt.Errorf("引き手に位置がありません")
	}
	// キューブはプレイヤーの立っているタイルへ入る。プレイヤーはそのぶん後退する
	dest := world.Components.GridElement.Get(actor).Coord

	duration, err := cubePushTurns(world, pa.Cube)
	if err != nil {
		return nil, err
	}

	comp, err := NewActivity(pa, duration)
	if err != nil {
		return nil, err
	}
	comp.Target = &pa.Cube
	comp.Destination = &gc.GridElement{Coord: dest}
	return comp, nil
}

// Validate はBehaviorの実装。後退先が通行可能であることを確かめる。
func (pa *PullActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil || !world.ECS.Alive(*comp.Target) {
		return fmt.Errorf("引く対象が存在しません")
	}
	if !world.Components.Pushable.Has(*comp.Target) {
		return fmt.Errorf("対象は引けません")
	}
	if !world.Components.GridElement.Has(*comp.Target) || comp.Destination == nil {
		return fmt.Errorf("引く対象に位置がありません")
	}
	if !world.Components.GridElement.Has(actor) {
		return fmt.Errorf("引き手に位置がありません")
	}
	cubeCoord := world.Components.GridElement.Get(*comp.Target).Coord
	retreat := pullRetreat(cubeCoord, comp.Destination.Coord)
	// 後退先がプレイヤーの行ける先であること。キューブの入る先はプレイヤーが退いて空く
	if !CanMoveTo(world, retreat, comp.Destination.Coord, actor) {
		return fmt.Errorf("引くスペースがありません")
	}
	return nil
}

// Start はBehaviorの実装
func (pa *PullActivity) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("引き開始", "actor", actor)
	return nil
}

// DoTurn はBehaviorの実装。毎ターン対象の生存と後退先の通行可否を確かめ、ターンを1つ消費する。
func (pa *PullActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil || !world.ECS.Alive(*comp.Target) {
		Cancel(comp, "引く対象が消えたため中断")
		return nil
	}
	if comp.Destination == nil || !world.Components.GridElement.Has(*comp.Target) || !world.Components.GridElement.Has(actor) {
		Cancel(comp, "引けなくなったため中断")
		return nil
	}
	cubeCoord := world.Components.GridElement.Get(*comp.Target).Coord
	retreat := pullRetreat(cubeCoord, comp.Destination.Coord)
	if !CanMoveTo(world, retreat, comp.Destination.Coord, actor) {
		Cancel(comp, "引くスペースがなくなった")
		return nil
	}

	comp.TurnsLeft--
	if comp.TurnsLeft <= 0 {
		Complete(comp)
	}
	return nil
}

// Finish はBehaviorの実装。キューブをプレイヤーの元タイルへ引き入れ、プレイヤーは1タイル後退する。
func (pa *PullActivity) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	cube := *comp.Target
	if !world.ECS.Alive(cube) || !world.Components.GridElement.Has(cube) || !world.Components.GridElement.Has(actor) {
		return nil
	}
	cubeGrid := world.Components.GridElement.Get(cube)
	cubeOld := cubeGrid.Coord
	retreat := pullRetreat(cubeOld, comp.Destination.Coord)

	// 先にプレイヤーを後退させてタイルを空け、そこへキューブを引き入れる
	world.Components.GridElement.Get(actor).Coord = retreat
	cubeGrid.Coord = comp.Destination.Coord

	// キューブは BlockPass なので通行索引が変わる。全再構築で確実に反映する
	query.InvalidateSpatialIndex(world)

	log.Debug("引き完了", "actor", actor, "cube", cube, "to", comp.Destination.Coord.String())
	return nil
}

// Canceled はBehaviorの実装
func (pa *PullActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("引きキャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}
