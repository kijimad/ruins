package states

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ComponentDebugState はコンポーネント数を一覧表示するデバッグ用ステート
type ComponentDebugState struct {
	es.BaseState[w.World]
	screen Screen[componentDebugProps]
}

var _ es.State[w.World] = &ComponentDebugState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *ComponentDebugState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *ComponentDebugState) OnResume(_ w.World) error { return nil }

// OnStop はステートが停止される際に呼ばれる
func (st *ComponentDebugState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *ComponentDebugState) OnStart(_ w.World) error {
	st.screen = NewScreen[componentDebugProps]()
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *ComponentDebugState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world, st)
}

// Draw はゲームステートの描画処理を行う
func (st *ComponentDebugState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// HandleInput はキー入力を Action に変換する
func (st *ComponentDebugState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
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
func NewComponentDebugState() (es.State[w.World], error) {
	return &ComponentDebugState{}, nil
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

func (st *ComponentDebugState) fetch(world w.World) componentDebugProps {
	// Ark に登録された全コンポーネントを走査し、種類ごとの保有エンティティ数を集計する
	ids := ecs.ComponentIDs(world.ECS)
	items := make([]componentDebugItem, 0, len(ids))
	total := 0

	for _, id := range ids {
		info, ok := ecs.ComponentInfo(world.ECS, id)
		if !ok {
			continue
		}
		q := ecs.NewUnsafeFilter(world.ECS, id).Query()
		count := q.Count()
		// Countは反復を完了しないためワールドロックが残る。明示的に閉じる
		q.Close()

		items = append(items, componentDebugItem{
			Name:  info.Type.Name(),
			Count: count,
		})
		total += count
	}

	// 数が多い順にソートする
	slices.SortFunc(items, func(a, b componentDebugItem) int {
		return cmp.Compare(b.Count, a.Count)
	})

	return componentDebugProps{Items: items, Total: total}
}

func (st *ComponentDebugState) menu(props componentDebugProps) MenuConfig {
	return MenuConfig{Key: "compdbg", TabCount: 1, ItemCounts: []int{len(props.Items)}, ItemsPerPage: menuItemsPerPage}
}

// ================
// view
// ================

func (st *ComponentDebugState) view(_ w.World, props componentDebugProps, sel Selection, res resources.UIResources) *ebitenui.UI {
	container := styled.NewVerticalContainer()

	pg := pagination.New(sel.ItemIndex, len(props.Items), menuItemsPerPage)
	container.AddChild(newPageIndicator(pg, res))

	columnWidths := []int{160, 60}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(props.Items, pg) {
		styled.NewTableRow(table, columnWidths,
			[]string{entry.Item.Name, fmt.Sprintf("%d", entry.Item.Count)},
			aligns, new(pg.IsSelectedInPage(entry.Index)), res,
		)
	}
	container.AddChild(table)

	// in-game モーダルの共通骨組みに揃える。見出しは合計数、下部にキー案内を常設する
	return newTabScreenUI(res, tabScreen{
		Header:  fmt.Sprintf("コンポーネント (合計: %d)", props.Total),
		Content: container,
		Footer:  menuNavHint(false),
	})
}
