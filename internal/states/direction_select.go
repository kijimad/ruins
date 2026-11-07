package states

import (
	"fmt"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// directionAction は矢印で選んだ隣接タイルへ行う行動
type directionAction struct {
	prompt   string
	emptyMsg string
	accepts  func(world w.World, c consts.Coord[consts.Tile]) bool
	run      func(world w.World, c consts.Coord[consts.Tile]) error
}

// directionOption は選択キーと、そのキーが指す候補タイルの組
type directionOption struct {
	action inputmapper.ActionID
	tile   consts.Coord[consts.Tile]
}

// startDirectionAction は候補が無ければログを出し、1つならそのまま実行し、複数なら DirectionSelectState を開く
func startDirectionAction(world w.World, a directionAction) (es.Transition[w.World], error) {
	options, err := directionOptions(world, a)
	if err != nil {
		return es.Transition[w.World]{Type: es.TransNone}, err
	}
	switch len(options) {
	case 0:
		gamelog.New(query.GetGameLog(world)).Markup(a.emptyMsg).Log()
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case 1:
		return es.Transition[w.World]{Type: es.TransNone}, a.run(world, options[0].tile)
	}
	return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
		func() (es.State[w.World], error) { return &DirectionSelectState{action: a, options: options}, nil },
	}}, nil
}

// directionOptions は選択キーごとの候補タイルを北・南・西・東の順に返す
func directionOptions(world w.World, a directionAction) ([]directionOption, error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil, err
	}
	var yaw float64
	if camera := query.GetPlayerCamera(world); camera != nil {
		yaw = camera.Yaw()
	}
	origin := world.Components.GridElement.Get(player).Coord
	var options []directionOption
	for _, o := range []directionOption{
		{action: inputmapper.ActionMoveNorth, tile: origin.Add(gc.RotateScreenDir(gc.DirectionUp, yaw).GetDelta())},
		{action: inputmapper.ActionMoveSouth, tile: origin.Add(gc.RotateScreenDir(gc.DirectionDown, yaw).GetDelta())},
		{action: inputmapper.ActionMoveWest, tile: origin.Add(gc.RotateScreenDir(gc.DirectionLeft, yaw).GetDelta())},
		{action: inputmapper.ActionMoveEast, tile: origin.Add(gc.RotateScreenDir(gc.DirectionRight, yaw).GetDelta())},
	} {
		if a.accepts(world, o.tile) {
			options = append(options, o)
		}
	}
	return options, nil
}

// DirectionSelectState は directionAction の対象タイルを矢印で選ぶステート
type DirectionSelectState struct {
	es.BaseState[w.World]
	action  directionAction
	options []directionOption
}

var _ es.State[w.World] = &DirectionSelectState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *DirectionSelectState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *DirectionSelectState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *DirectionSelectState) OnStart(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *DirectionSelectState) OnStop(_ w.World) error { return nil }

// directionSelectBindings は方向選択の束縛表
var directionSelectBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
	{Key: ebiten.KeyUp, Action: inputmapper.ActionMoveNorth},
	{Key: ebiten.KeyDown, Action: inputmapper.ActionMoveSouth},
	{Key: ebiten.KeyLeft, Action: inputmapper.ActionMoveWest},
	{Key: ebiten.KeyRight, Action: inputmapper.ActionMoveEast},
}

// Update はステートの更新処理
func (st *DirectionSelectState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, directionSelectBindings); ok {
		return st.doAction(world, action)
	}
	return st.ConsumeTransition(), nil
}

// doAction はActionIDを実行する
func (st *DirectionSelectState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	if action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	i := slices.IndexFunc(st.options, func(o directionOption) bool { return o.action == action })
	if i < 0 {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	if err := st.action.run(world, st.options[i].tile); err != nil {
		return es.Transition[w.World]{Type: es.TransNone}, err
	}
	return es.Transition[w.World]{Type: es.TransPop}, nil
}

// Draw は候補タイルの枠と操作の案内を描画する
func (st *DirectionSelectState) Draw(world w.World, screen *ebiten.Image) error {
	projector, err := render3d.WorldProjector(world)
	if err != nil {
		return err
	}
	for _, o := range st.options {
		if corners, ok := projector.TileCorners(o.tile, render3d.TileTopHeight(world, o.tile)); ok {
			hud.TileFrame(screen, corners, cursorFrameWidth, theme.CursorLook)
		}
	}

	const panelHeight = 60
	panel := hud.NewInfoPanel(uicore.NewEbitenCanvas(screen), hud.Chrome{}, world.Resources.UIResources.Text.BodyFace, screen.Bounds().Dx(), panelHeight)
	panel.Line(st.action.prompt)
	panel.Line(query.T(world, "Arrows: Select  Esc: Cancel"))
	return nil
}

// attackAction は隣接タイルの攻撃対象を近接攻撃する方向行動
func attackAction(world w.World) directionAction {
	return directionAction{
		prompt:   query.T(world, "Attack which direction?"),
		emptyMsg: query.T(world, "There is nothing to attack nearby."),
		accepts: func(world w.World, c consts.Coord[consts.Tile]) bool {
			_, ok := meleeTargetAt(world, c)
			return ok
		},
		run: func(world w.World, c consts.Coord[consts.Tile]) error {
			player, err := query.GetPlayerEntity(world)
			if err != nil {
				return err
			}
			target, ok := meleeTargetAt(world, c)
			if !ok {
				return fmt.Errorf("melee target disappeared at %v", c)
			}
			_, err = activity.ExecuteInteraction(player, target, gc.InteractionMelee, world)
			return err
		},
	}
}

// meleeTargetAt はタイル上の近接攻撃の対象を返す。歩ける草などの prop と敵が重なるので、prop でない相手を優先する
func meleeTargetAt(world w.World, c consts.Coord[consts.Tile]) (ecs.Entity, bool) {
	var prop ecs.Entity
	found := false
	for _, e := range query.GetEntitiesAt(world, c.X, c.Y) {
		if world.Components.Dead.Has(e) || !world.Components.Interactable.Has(e) ||
			!slices.Contains(world.Components.Interactable.Get(e).Interactions, gc.InteractionMelee) {
			continue
		}
		if !world.Components.Prop.Has(e) {
			return e, true
		}
		if !found {
			prop, found = e, true
		}
	}
	return prop, found
}

// interactableAt はタイル上で相互作用 kind を受け付ける生存中のエンティティを返す
func interactableAt(world w.World, c consts.Coord[consts.Tile], kind gc.InteractionKind) (ecs.Entity, bool) {
	for _, e := range query.GetEntitiesAt(world, c.X, c.Y) {
		if world.Components.Dead.Has(e) || !world.Components.Interactable.Has(e) {
			continue
		}
		if slices.Contains(world.Components.Interactable.Get(e).Interactions, kind) {
			return e, true
		}
	}
	return ecs.Entity{}, false
}
