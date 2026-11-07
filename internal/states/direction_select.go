package states

import (
	"fmt"
	"maps"
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

// directionAction は方向を選んで隣接タイルへ行う行動。動詞を先に決め、対象のタイルを後から選ぶ。
// 候補は矢印4方向がカメラの水平角で指す隣接タイルのうち accepts を満たすものに限る。
// 斜めのタイルは視点を回して矢印の指す先にしてから選ぶ
type directionAction struct {
	prompt   string
	emptyMsg string
	accepts  func(world w.World, c consts.Coord[consts.Tile]) bool
	run      func(world w.World, c consts.Coord[consts.Tile]) error
}

// startDirectionAction は候補タイルを数えて分岐する。候補が無ければ emptyMsg をログに出し、
// 1つならそのまま実行し、複数なら DirectionSelectState を開いて選ばせる
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
		return es.Transition[w.World]{Type: es.TransNone}, a.run(world, slices.Collect(maps.Values(options))[0])
	}
	return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
		func() (es.State[w.World], error) { return &DirectionSelectState{action: a, options: options}, nil },
	}}, nil
}

// directionOptions は選択キーごとの候補タイルを返す
func directionOptions(world w.World, a directionAction) (map[inputmapper.ActionID]consts.Coord[consts.Tile], error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil, err
	}
	var yaw float64
	if camera := query.GetPlayerCamera(world); camera != nil {
		yaw = camera.Yaw()
	}
	origin := world.Components.GridElement.Get(player).Coord
	tiles := map[inputmapper.ActionID]consts.Coord[consts.Tile]{
		inputmapper.ActionMoveNorth: origin.Add(gc.RotateScreenDir(gc.DirectionUp, yaw).GetDelta()),
		inputmapper.ActionMoveSouth: origin.Add(gc.RotateScreenDir(gc.DirectionDown, yaw).GetDelta()),
		inputmapper.ActionMoveWest:  origin.Add(gc.RotateScreenDir(gc.DirectionLeft, yaw).GetDelta()),
		inputmapper.ActionMoveEast:  origin.Add(gc.RotateScreenDir(gc.DirectionRight, yaw).GetDelta()),
	}
	options := map[inputmapper.ActionID]consts.Coord[consts.Tile]{}
	for id, c := range tiles {
		if a.accepts(world, c) {
			options[id] = c
		}
	}
	return options, nil
}

// DirectionSelectState は directionAction の対象タイルを選ぶステート。候補タイルを枠で示し、
// 矢印で選ぶ。候補でない向きの入力は無視し、Esc でやめる
type DirectionSelectState struct {
	es.BaseState[w.World]
	action  directionAction
	options map[inputmapper.ActionID]consts.Coord[consts.Tile]
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
	c, ok := st.options[action]
	if !ok {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	if err := st.action.run(world, c); err != nil {
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
	for _, c := range st.options {
		if corners, ok := projector.TileCorners(c, render3d.TileTopHeight(world, c)); ok {
			hud.TileFrame(screen, corners, cursorFrameWidth, theme.CursorLook)
		}
	}

	const panelHeight = 60
	panel := hud.NewInfoPanel(uicore.NewEbitenCanvas(screen), hud.Chrome{}, world.Resources.UIResources.Text.BodyFace, screen.Bounds().Dx(), panelHeight)
	panel.Line(st.action.prompt)
	panel.Line(query.T(world, "Arrows: Select  Esc: Cancel"))
	return nil
}

// attackAction は隣接タイルの攻撃対象を近接攻撃する方向行動。攻撃は相互作用メニューに並べず、
// この行動からだけ行う。敵への歩き込みは別経路で攻撃になる
func attackAction(world w.World) directionAction {
	return directionAction{
		prompt:   query.T(world, "Attack which direction?"),
		emptyMsg: query.T(world, "There is nothing to attack nearby."),
		accepts: func(world w.World, c consts.Coord[consts.Tile]) bool {
			_, ok := interactableAt(world, c, gc.InteractionMelee)
			return ok
		},
		run: func(world w.World, c consts.Coord[consts.Tile]) error {
			player, err := query.GetPlayerEntity(world)
			if err != nil {
				return err
			}
			target, ok := interactableAt(world, c, gc.InteractionMelee)
			if !ok {
				return fmt.Errorf("melee target disappeared at %v", c)
			}
			_, err = activity.ExecuteInteraction(player, target, gc.InteractionMelee, world)
			return err
		},
	}
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
