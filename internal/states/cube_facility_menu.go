package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// CubeFacilityMenuState はキューブの展開空間へ設備を据える画面。展開空間を表す UI 内グリッドを描き、
// カーソルでマスを選ぶ。空きマスで決定すると据付アイテムの選択へ進み、設備マスで決定すると撤去する。
// フィールドを歩いて設置/撤去する経路は持たず、設置と撤去をこの画面へ集約する。展開中のみ開く。
type CubeFacilityMenuState struct {
	es.BaseState[w.World]
	cube   ecs.Entity
	cursor consts.Coord[consts.Tile] // キューブ中心からの相対オフセット
}

var _ es.State[w.World] = &CubeFacilityMenuState{}

// NewCubeFacilityMenuState はキューブの設備画面を開くファクトリを返す。カーソルはキューブ本体でなく
// 隣のマスから始める。本体マスは据付先にならず、そこから始めると初手の決定が空振りして戸惑うため。
func NewCubeFacilityMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return &CubeFacilityMenuState{cube: cube, cursor: consts.Coord[consts.Tile]{X: 1}}, nil
}

// OnStart はステートが開始される際に呼ばれる。OnPause/OnResume/OnStop は BaseState の既定に委ねる
func (st *CubeFacilityMenuState) OnStart(_ w.World) error { return nil }

// cubeFacilityBindings は設備画面の束縛表。矢印でカーソルを動かし、Enter で決定、Esc で閉じる
var cubeFacilityBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Press: keybind.PressJust, Action: inputmapper.ActionMenuCancel},
	{Key: ebiten.KeyArrowUp, Press: keybind.PressRepeat, Action: inputmapper.ActionMenuUp},
	{Key: ebiten.KeyArrowDown, Press: keybind.PressRepeat, Action: inputmapper.ActionMenuDown},
	{Key: ebiten.KeyArrowLeft, Press: keybind.PressRepeat, Action: inputmapper.ActionMenuLeft},
	{Key: ebiten.KeyArrowRight, Press: keybind.PressRepeat, Action: inputmapper.ActionMenuRight},
	{Key: ebiten.KeyEnter, Press: keybind.PressJust, Action: inputmapper.ActionMenuSelect},
}

// Update はゲームステートの更新処理を行う
func (st *CubeFacilityMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 展開中でなければ空間が無いので閉じる。呼び出し側が展開中のみ開くので通常は起きない
	if !world.Components.Deployed.Has(st.cube) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	if action, ok := keybind.ReadInput(world, cubeFacilityBindings); ok {
		return st.doAction(world, action)
	}
	return st.ConsumeTransition(), nil
}

// doAction は Action を実行する
func (st *CubeFacilityMenuState) doAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuUp:
		st.moveCursor(world, 0, -1)
	case inputmapper.ActionMenuDown:
		st.moveCursor(world, 0, 1)
	case inputmapper.ActionMenuLeft:
		st.moveCursor(world, -1, 0)
	case inputmapper.ActionMenuRight:
		st.moveCursor(world, 1, 0)
	case inputmapper.ActionMenuSelect:
		return st.selectCell(world)
	default:
		return es.Transition[w.World]{}, fmt.Errorf("cubeFacility: unsupported action: %s", action)
	}
	return st.ConsumeTransition(), nil
}

// moveCursor はカーソルを凍結した展開範囲の中でクランプして動かす
func (st *CubeFacilityMenuState) moveCursor(world w.World, dx consts.Tile, dy consts.Tile) {
	r := world.Components.Deployed.Get(st.cube).Range
	next := st.cursor.Add(consts.Coord[consts.Tile]{X: dx, Y: dy})
	if next.X < -r.X || next.X > r.X || next.Y < -r.Y || next.Y > r.Y {
		return
	}
	st.cursor = next
}

// selectCell はカーソルのマスを決定する。空きなら据付アイテム選択へ、設備なら撤去する
func (st *CubeFacilityMenuState) selectCell(world w.World) (es.Transition[w.World], error) {
	base := world.Components.GridElement.Get(st.cube).Coord
	coord := base.Add(st.cursor)
	kind, _ := st.cellKindAt(world, coord)
	switch kind {
	case hud.FacilityCellEmpty:
		player, err := query.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if len(query.BackpackDeployables(world, player)) == 0 {
			gamelog.New(query.GetGameLog(world)).
				Markup(query.T(world, "You have nothing to deploy.")).
				Log()
			return st.ConsumeTransition(), nil
		}
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{newCubeFacilitySelectState(st.cube, coord)},
		}, nil
	case hud.FacilityCellUsed:
		return st.removeAt(world, coord)
	default:
		// キューブ本体・障害物のマスは決定しても何もしない
		return st.ConsumeTransition(), nil
	}
}

// removeAt は coord の設備を撤去してアイテムへ戻す。中身が残る収納など撤去できない場合はログを出す
func (st *CubeFacilityMenuState) removeAt(world w.World, coord consts.Coord[consts.Tile]) (es.Transition[w.World], error) {
	prop, ok := query.FacilityAt(world, coord)
	if !ok {
		return st.ConsumeTransition(), nil
	}
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	// FacilityAt が Deployable を保証するので、ここへ来る err は収納非空のみ。他種別が増えたら文言を分ける
	if err := lifecycle.RemoveFacility(world, prop, player); err != nil {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Cannot remove: empty its contents first.")).
			Log()
	}
	return st.ConsumeTransition(), nil
}

// cellKindAt は coord のマスの見た目種別と、Used のとき据わっている設備を返す。据付でない prop や壁は障害物。
func (st *CubeFacilityMenuState) cellKindAt(world w.World, coord consts.Coord[consts.Tile]) (hud.FacilityCellKind, ecs.Entity) {
	if world.Components.GridElement.Get(st.cube).Coord == coord {
		return hud.FacilityCellCube, ecs.Entity{}
	}
	if e, ok := query.FacilityAt(world, coord); ok {
		return hud.FacilityCellUsed, e
	}
	if lifecycle.FacilityCellFree(world, st.cube, coord) {
		return hud.FacilityCellEmpty, ecs.Entity{}
	}
	return hud.FacilityCellBlocked, ecs.Entity{}
}

// Draw は展開空間のグリッドを組んで hud へ渡す。uicore の組み立ては hud に閉じ、画面はデータを渡すだけにする
func (st *CubeFacilityMenuState) Draw(world w.World, screen *ebiten.Image) error {
	// Update は展開中でなければ Pop するが、遷移フレームで Draw が先に来ても Deployed.Get で
	// panic しないよう守る。展開空間が無ければ描くものが無い
	if !world.Components.Deployed.Has(st.cube) {
		return nil
	}
	r := world.Components.Deployed.Get(st.cube).Range
	base := world.Components.GridElement.Get(st.cube).Coord
	cols := 2*int(r.X) + 1
	rows := 2*int(r.Y) + 1

	cells := make([]hud.FacilityCell, 0, cols*rows)
	for row := range rows {
		for col := range cols {
			coord := base.Add(consts.Coord[consts.Tile]{X: consts.Tile(col) - r.X, Y: consts.Tile(row) - r.Y})
			kind, facility := st.cellKindAt(world, coord)
			cells = append(cells, hud.FacilityCell{Kind: kind, Icon: st.iconFor(world, kind, facility)})
		}
	}

	view := hud.FacilityGridView{
		Cols:      cols,
		Rows:      rows,
		Cells:     cells,
		CursorCol: int(st.cursor.X + r.X),
		CursorRow: int(st.cursor.Y + r.Y),
		Title:     query.T(world, "Facility"),
		Footer:    query.T(world, "Arrows: Move  Esc: Close"),
	}
	view.Content, view.Hint = st.cursorInfo(world, base.Add(st.cursor))

	hud.DrawFacilityGrid(uicore.NewEbitenCanvas(screen), menuframe.WindowRect(world), world.Resources.UIResources.Text.BodyFace, view)
	return nil
}

// iconFor はマスに重ねるスプライトを返す。キューブ本体と据わった設備だけ絵を持つ。
func (st *CubeFacilityMenuState) iconFor(world w.World, kind hud.FacilityCellKind, facility ecs.Entity) *ebiten.Image {
	switch kind {
	case hud.FacilityCellCube:
		return menuIcon(world, st.cube)
	case hud.FacilityCellUsed:
		return menuIcon(world, facility)
	case hud.FacilityCellEmpty, hud.FacilityCellBlocked:
	}
	return nil
}

// cursorInfo はカーソルのマスの内容名と押せる操作を返す
func (st *CubeFacilityMenuState) cursorInfo(world w.World, coord consts.Coord[consts.Tile]) (content string, hint string) {
	kind, facility := st.cellKindAt(world, coord)
	switch kind {
	case hud.FacilityCellCube:
		return query.T(world, "Cube"), ""
	case hud.FacilityCellUsed:
		return query.GetEntityName(facility, world), query.T(world, "Enter: remove")
	case hud.FacilityCellEmpty:
		return query.T(world, "Empty"), query.T(world, "Enter: place")
	default:
		return query.T(world, "Occupied"), ""
	}
}

// CubeFacilitySelectState は空きマスへ据える据付アイテムを選ぶメニュー。モジュール選択と同型で、
// バックパックの据付アイテムを並べ、選ぶと coord へ据えて閉じる。
type CubeFacilitySelectState struct {
	es.BaseState[w.World]
	cube   ecs.Entity
	coord  consts.Coord[consts.Tile]
	screen *menuloop.Screen[CubeFacilitySelectProps]
}

var _ es.State[w.World] = &CubeFacilitySelectState{}

// CubeFacilitySelectProps は据付アイテム選択の表示 props
type CubeFacilitySelectProps struct {
	Candidates []ecs.Entity // 据えられる据付アイテム。プレイヤーのバックパックから集める
}

// newCubeFacilitySelectState は coord への据付アイテム選択を開くファクトリを返す
func newCubeFacilitySelectState(cube ecs.Entity, coord consts.Coord[consts.Tile]) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &CubeFacilitySelectState{cube: cube, coord: coord}, nil
	}
}

// OnStart はステートが開始される際に呼ばれる
func (st *CubeFacilitySelectState) OnStart(_ w.World) error {
	st.screen = menuloop.NewScreen[CubeFacilitySelectProps](st)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *CubeFacilitySelectState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *CubeFacilitySelectState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// Fetch は世界から表示 props を構築する。候補はプレイヤーのバックパックから集める。
// プレイヤー不在は握りつぶさず error で返して早期に検知する。
func (st *CubeFacilitySelectState) Fetch(world w.World) (CubeFacilitySelectProps, error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return CubeFacilitySelectProps{}, fmt.Errorf("cube facility select: %w", err)
	}
	return CubeFacilitySelectProps{Candidates: query.BackpackDeployables(world, player)}, nil
}

// Menu は単一リストの構成を返す
func (st *CubeFacilitySelectState) Menu(props CubeFacilitySelectProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "cube_facility_select", TabCount: 1, ItemCounts: []int{len(props.Candidates)}, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

// ViewUI は候補一覧を中央パネルへ組む
func (st *CubeFacilitySelectState) ViewUI(world w.World, props CubeFacilitySelectProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	rows := make([]menuframe.Row, 0, len(props.Candidates))
	for _, entity := range props.Candidates {
		rows = append(rows, menuframe.Row{Cells: []styled.Cell{styled.IconCell(menuIcon(world, entity)), styled.TextCell(query.GetEntityName(entity, world))}})
	}
	list, pager := menuframe.RenderList(cursor.ItemIndex, rows, styled.Cols(styled.Icon(), styled.Name()),
		menuframe.ListOpts{EmptyText: query.T(world, "No items to deploy")}, res)
	return menuframe.PanelScreen(world, res, query.T(world, "Choose facility"), list, keybind.HelpHint(world), pager)
}

// DoAction はActionを実行する。選択で据えて閉じる
func (st *CubeFacilitySelectState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		if err := placeFacilityChoice(world, st.cube, st.coord, st.screen.Props().Candidates, st.screen.Selection().ItemIndex); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("cubeFacilitySelect: unsupported action: %s", action)
	}
}

// placeFacilityChoice は候補一覧の idx を coord へ据える。範囲外の idx は据えず nil を返す。
// 画面の選択状態から据付を切り出し、UI ループを介さずに設置ロジックを試験できるようにする。
func placeFacilityChoice(world w.World, cube ecs.Entity, coord consts.Coord[consts.Tile], candidates []ecs.Entity, idx int) error {
	if idx < 0 || idx >= len(candidates) {
		return nil
	}
	_, err := lifecycle.PlaceFacility(world, cube, candidates[idx], coord)
	return err
}
