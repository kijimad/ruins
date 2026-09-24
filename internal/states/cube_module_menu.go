package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// CubeModuleMenuState はキューブのモジュールスロット一覧。装備画面のスロット一覧と同型で、
// 装着中のモジュールと空きスロットを consts.CubeModuleSlots 個並べる。スロットを選ぶと候補選択へ進む。
type CubeModuleMenuState struct {
	es.BaseState[w.World]
	cube   ecs.Entity
	screen *menuloop.Screen[CubeModuleMenuProps]
}

var _ es.State[w.World] = &CubeModuleMenuState{}

// CubeModuleMenuProps はスロット一覧の表示 props
type CubeModuleMenuProps struct {
	Slots []ecs.Entity // 長さ consts.CubeModuleSlots。各スロットの装着モジュール。空きは gc.InvalidEntity
}

// NewCubeModuleMenuState はキューブのモジュールスロット一覧を開くファクトリを返す
func NewCubeModuleMenuState(cube ecs.Entity) (es.State[w.World], error) {
	return &CubeModuleMenuState{cube: cube}, nil
}

// OnStart はステートが開始される際に呼ばれる
func (st *CubeModuleMenuState) OnStart(_ w.World) error {
	st.screen = menuloop.NewScreen[CubeModuleMenuProps](st)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *CubeModuleMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *CubeModuleMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// Fetch は世界から表示 props を構築する。装着モジュールを保存スロット番号どおりに並べる
func (st *CubeModuleMenuState) Fetch(world w.World) (CubeModuleMenuProps, error) {
	slots := make([]ecs.Entity, consts.CubeModuleSlots)
	for i := range slots {
		slots[i] = gc.InvalidEntity
	}
	for _, m := range query.GetCubeModules(world, st.cube) {
		s := world.Components.LocationInstalled.Get(m).Slot
		// 範囲外・スロット重複は握りつぶさず error で返して早期に検知する。片方を黙って捨てると不整合を隠す
		if s < 0 || s >= consts.CubeModuleSlots {
			return CubeModuleMenuProps{}, fmt.Errorf("cube module: slot %d out of range [0,%d)", s, consts.CubeModuleSlots)
		}
		if slots[s] != gc.InvalidEntity {
			return CubeModuleMenuProps{}, fmt.Errorf("cube module: duplicate slot %d", s)
		}
		slots[s] = m
	}
	return CubeModuleMenuProps{Slots: slots}, nil
}

// Menu はスロット数ぶんの単一リストを返す。装着中と空きを合わせて常に上限ぶん並べる
func (st *CubeModuleMenuState) Menu(_ CubeModuleMenuProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "cube_module", TabCount: 1, ItemCounts: []int{consts.CubeModuleSlots}, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

// ViewUI はスロット一覧を中央パネルへ組む
func (st *CubeModuleMenuState) ViewUI(world w.World, props CubeModuleMenuProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	// スロット名を左、アイコンとアイテム名を右へ寄せる。間の伸縮スペーサで両者を離す
	cols := styled.Cols(styled.Fit(), styled.Name(), styled.Icon(), styled.Fit())
	rows := make([]menuframe.Row, consts.CubeModuleSlots)
	for i := range rows {
		label := query.T(world, "Slot %d", i+1)
		var icon *ebiten.Image
		name := query.T(world, "(empty)")
		if props.Slots[i] != gc.InvalidEntity {
			icon = menuIcon(world, props.Slots[i])
			name = query.GetEntityName(props.Slots[i], world)
		}
		rows[i] = menuframe.Row{Cells: []styled.Cell{styled.TextCell(label), styled.TextCell(""), styled.IconCell(icon), styled.TextCell(name)}}
	}
	list, pager := menuframe.RenderList(cursor.ItemIndex, rows, cols, menuframe.ListOpts{}, res)
	return menuframe.PanelScreen(world, res, query.T(world, "Module"), list, keybind.HelpHint(world), pager)
}

// DoAction はActionを実行する。スロットを選ぶと候補選択へ進む
func (st *CubeModuleMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		// menuloop が ItemCounts=CubeModuleSlots でカーソルを範囲内に保つので slots[idx] は安全
		slots := st.screen.Props().Slots
		idx := st.screen.Selection().ItemIndex
		var current *ecs.Entity
		if slots[idx] != gc.InvalidEntity {
			m := slots[idx]
			current = &m
		}
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{newCubeModuleSelectState(st.cube, idx, current)},
		}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("cubeModuleMenu: unsupported action: %s", action)
	}
}

// CubeModuleSelectState はスロット1つのモジュールを選ぶメニュー。装備の EquipSelectState と同型で、
// 装着済みなら先頭に「外す」が並ぶ。x で装着中と候補の詳細を見比べる。
type CubeModuleSelectState struct {
	es.BaseState[w.World]
	cube      ecs.Entity
	slot      int         // 編集対象のスロット番号
	installed *ecs.Entity // このスロットの装着中モジュール。空きなら nil
	detail    overlay.Detail
	screen    *menuloop.Screen[CubeModuleSelectProps]
}

var _ es.State[w.World] = &CubeModuleSelectState{}
var _ menuloop.KeyBindings = &CubeModuleSelectState{}

// CubeModuleSelectProps はモジュール選択の表示 props
type CubeModuleSelectProps struct {
	Candidates []ecs.Entity // 装着できるモジュール。キューブ収納とバックパックから集める
	Installed  *ecs.Entity
}

// newCubeModuleSelectState はスロットに対するモジュール選択を開くファクトリを返す
func newCubeModuleSelectState(cube ecs.Entity, slot int, installed *ecs.Entity) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &CubeModuleSelectState{cube: cube, slot: slot, installed: installed}, nil
	}
}

// OnStart はステートが開始される際に呼ばれる
func (st *CubeModuleSelectState) OnStart(_ w.World) error {
	st.detail = overlay.NewComparison(st.compareContents)
	st.screen = menuloop.NewScreen[CubeModuleSelectProps](st, &st.detail)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *CubeModuleSelectState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *CubeModuleSelectState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// KeyBindings は x の詳細表示を共通入力に足す
func (st *CubeModuleSelectState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
}

// Fetch は世界から表示 props を構築する。候補はプレイヤーのバックパックから集める。
// プレイヤー不在は握りつぶさず error で返して早期に検知する。
func (st *CubeModuleSelectState) Fetch(world w.World) (CubeModuleSelectProps, error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return CubeModuleSelectProps{}, fmt.Errorf("cube module select: %w", err)
	}
	return CubeModuleSelectProps{Candidates: query.BackpackCubeModules(world, player), Installed: st.installed}, nil
}

// Menu は単一リストの構成を返す。装着済みなら先頭の「外す」ぶんを1つ足す
func (st *CubeModuleSelectState) Menu(props CubeModuleSelectProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "cube_module_select", TabCount: 1, ItemCounts: []int{cubeModuleChoiceCount(props)}, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

// ViewUI は候補一覧を中央パネルへ組む。装着済みなら先頭に「外す」が並ぶ
func (st *CubeModuleSelectState) ViewUI(world w.World, props CubeModuleSelectProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	var rows []menuframe.Row
	if props.Installed != nil {
		rows = append(rows, menuframe.Row{Cells: []styled.Cell{styled.IconCell(nil), styled.TextCell(query.T(world, "Remove"))}})
	}
	for _, entity := range props.Candidates {
		rows = append(rows, menuframe.Row{Cells: []styled.Cell{styled.IconCell(menuIcon(world, entity)), styled.TextCell(query.GetEntityName(entity, world))}})
	}
	list, pager := menuframe.RenderList(cursor.ItemIndex, rows, styled.Cols(styled.Icon(), styled.Name()),
		menuframe.ListOpts{EmptyText: query.T(world, "No modules to install")}, res)
	return menuframe.PanelScreen(world, res, query.T(world, "Choose module"), list, keybind.HelpHint(world), pager)
}

// DoAction はActionを実行する。選択で装着または外して閉じる
func (st *CubeModuleSelectState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		if choice, ok := st.selection(); ok {
			if err := applyCubeModuleChoice(world, st.cube, st.slot, choice, st.installed); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("cubeModuleSelect: unsupported action: %s", action)
	}
}

// selection は現在カーソルが指す選択を返す
func (st *CubeModuleSelectState) selection() (cubeModuleChoice, bool) {
	return cubeModuleChoiceAt(st.screen.Props(), st.screen.Selection().ItemIndex)
}

// compareContents は詳細に出す内容を返す。装着中と候補を2枚、空きスロットの候補なら候補1枚、
// 「外す」なら外す対象1枚を返す。並びは左が装着中、右が候補
func (st *CubeModuleSelectState) compareContents(world w.World) ([]overlay.DetailContent, bool) {
	choice, ok := st.selection()
	if !ok {
		return nil, false
	}
	if choice.remove {
		return []overlay.DetailContent{overlay.EntityDetailContent(world, *st.installed)}, true
	}
	candidate := overlay.EntityDetailContent(world, choice.entity)
	if st.installed == nil {
		return []overlay.DetailContent{candidate}, true
	}
	return []overlay.DetailContent{overlay.EntityDetailContent(world, *st.installed), candidate}, true
}

// cubeModuleChoice はカーソルが指すモジュール選択。remove なら「外す」、そうでなければ entity が候補
type cubeModuleChoice struct {
	remove bool
	entity ecs.Entity
}

// cubeModuleChoiceAt は候補一覧の index を選択へ写す。装着済みなら先頭が「外す」で、候補は1つ後ろへ詰まる
func cubeModuleChoiceAt(props CubeModuleSelectProps, index int) (cubeModuleChoice, bool) {
	if props.Installed != nil {
		if index == 0 {
			return cubeModuleChoice{remove: true}, true
		}
		index--
	}
	if index < 0 || index >= len(props.Candidates) {
		return cubeModuleChoice{}, false
	}
	return cubeModuleChoice{entity: props.Candidates[index]}, true
}

// cubeModuleChoiceCount は選択できる項目数を返す。装着済みなら先頭の「外す」を1つ足す
func cubeModuleChoiceCount(props CubeModuleSelectProps) int {
	if props.Installed != nil {
		return len(props.Candidates) + 1
	}
	return len(props.Candidates)
}

// applyCubeModuleChoice は選択を指定スロットへ実行する。「外す」なら装着中をプレイヤーのバックパックへ戻し、
// 候補なら装着する。スロットに装着中があれば先にバックパックへ戻してから付け替える。キューブは browse できる
// 収納を持たないので、外したモジュールは取り出せるバックパックへ返す。
func applyCubeModuleChoice(world w.World, cube ecs.Entity, slot int, choice cubeModuleChoice, installed *ecs.Entity) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return fmt.Errorf("cube module: %w", err)
	}
	if choice.remove {
		if installed == nil {
			return nil
		}
		return lifecycle.MoveToBackpack(world, *installed, player)
	}
	if installed != nil {
		if err := lifecycle.MoveToBackpack(world, *installed, player); err != nil {
			return err
		}
	}
	lifecycle.MoveToInstalled(world, choice.entity, cube, slot)
	return nil
}
