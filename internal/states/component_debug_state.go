package states

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

const componentDebugItemsPerPage = 20

// ComponentDebugState はコンポーネント数を一覧表示するデバッグ用ステート
type ComponentDebugState struct {
	es.BaseState[w.World]
	mount  *hooks.Mount[componentDebugProps]
	widget *ebitenui.UI
}

func (st ComponentDebugState) String() string {
	return "ComponentDebug"
}

var _ es.State[w.World] = &ComponentDebugState{}

func (st *ComponentDebugState) OnPause(_ w.World) error  { return nil }
func (st *ComponentDebugState) OnResume(_ w.World) error { return nil }
func (st *ComponentDebugState) OnStop(_ w.World) error   { return nil }

func (st *ComponentDebugState) OnStart(_ w.World) error {
	st.mount = hooks.NewMount[componentDebugProps]()
	return nil
}

func (st *ComponentDebugState) Update(world w.World) (es.Transition[w.World], error) {
	action, ok := HandleMenuInput()
	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.mount.Dispatch(action)
	}

	props := st.fetchProps(world)
	st.mount.SetProps(props)

	hooks.UseTabMenu(st.mount.Store(), "compdbg", hooks.TabMenuConfig{
		TabCount:     1,
		ItemCounts:   []int{len(props.Items)},
		ItemsPerPage: componentDebugItemsPerPage,
	})

	if st.mount.Update() {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

func (st *ComponentDebugState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

func (st *ComponentDebugState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuSelect:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// NewComponentDebugState はコンポーネントデバッグ画面を作成する
func NewComponentDebugState() es.State[w.World] {
	return &ComponentDebugState{}
}

// ================
// Props
// ================

type componentDebugProps struct {
	Items []componentDebugItem
	Total int
}

type componentDebugItem struct {
	Name  string
	Count int
}

func (st *ComponentDebugState) fetchProps(world w.World) componentDebugProps {
	comps := world.Components
	val := reflect.ValueOf(comps).Elem()
	typ := val.Type()

	var items []componentDebugItem
	total := 0

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := typ.Field(i).Name

		var count int
		if !field.IsNil() {
			if comp, ok := field.Interface().(ecs.DataComponent); ok {
				count = world.Manager.Join(comp).Size()
			}
		}

		items = append(items, componentDebugItem{
			Name:  fieldName,
			Count: count,
		})
		total += count
	}

	// 数が多い順にソートする
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	return componentDebugProps{Items: items, Total: total}
}

// ================
// buildUI
// ================

func (st *ComponentDebugState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, "compdbg")
	itemIndex := menuState.ItemIndex

	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(1),
				widget.GridLayoutOpts.Spacing(0, theme.Space2),
				widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, true}),
				widget.GridLayoutOpts.Padding(&widget.Insets{
					Top:    theme.Space3,
					Bottom: theme.Space3,
					Left:   theme.Space3,
					Right:  theme.Space3,
				}),
			),
		),
	)

	// Row 0: タイトル
	root.AddChild(styled.NewTitleText(fmt.Sprintf("コンポーネント (合計: %d)", props.Total), res))

	// Row 1: リスト
	container := styled.NewVerticalContainer()

	pg := pagination.New(itemIndex, len(props.Items), componentDebugItemsPerPage)
	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " "
	}
	container.AddChild(styled.NewPageIndicator(pageText, res))

	columnWidths := []int{160, 60}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	table := styled.NewTableContainer(columnWidths, res)

	for _, entry := range pagination.VisibleEntries(props.Items, pg) {
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, columnWidths,
			[]string{entry.Item.Name, fmt.Sprintf("%d", entry.Item.Count)},
			aligns, &isSelected, res,
		)
	}
	container.AddChild(table)
	root.AddChild(container)

	return &ebitenui.UI{Container: root}
}
