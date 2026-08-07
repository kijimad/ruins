package states

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ComponentDebugState はコンポーネント数を一覧表示するデバッグ用ステート
type ComponentDebugState struct {
	es.BaseState[w.World]
	screen *menurt.Screen[ComponentDebugProps]
}

var _ es.State[w.World] = &ComponentDebugState{}

// OnStart はステートが開始される際に呼ばれる
func (st *ComponentDebugState) OnStart(_ w.World) error {
	st.screen = menurt.NewScreen[ComponentDebugProps](st)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *ComponentDebugState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *ComponentDebugState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// DoAction はActionを実行する
func (st *ComponentDebugState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuSelect:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("unknown action: %s", action)
	}
}

// NewComponentDebugState はコンポーネントデバッグ画面を作成する
func NewComponentDebugState() (es.State[w.World], error) {
	return &ComponentDebugState{}, nil
}

// ================
// Props
// ================

// ComponentDebugProps は画面の表示 props。menurt.Screen の型引数として渡す
type ComponentDebugProps struct {
	Items []componentDebugItem
	Total int
}

type componentDebugItem struct {
	Name  string
	Count int
}

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *ComponentDebugState) Fetch(world w.World) ComponentDebugProps {
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

	return ComponentDebugProps{Items: items, Total: total}
}

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *ComponentDebugState) Menu(props ComponentDebugProps) menurt.MenuConfig {
	return menurt.MenuConfig{Key: "compdbg", TabCount: 1, ItemCounts: []int{len(props.Items)}, ItemsPerPage: menuItemsPerPage}
}

// ================
// view
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *ComponentDebugState) View(_ w.World, props ComponentDebugProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
	columnWidths := []int{260, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	rows := make([]menuRow, len(props.Items))
	for i, it := range props.Items {
		rows[i] = menuRow{Cells: []string{it.Name, fmt.Sprintf("%d", it.Count)}}
	}
	container := renderMenuList(cursor.ItemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true}, res)

	// in-game モーダルの共通骨組みに揃える。見出しは合計数、下部にキー案内を常設する
	return newTabScreenUI(res, tabScreen{
		Header:  fmt.Sprintf("Components total: %d", props.Total),
		Content: container,
		Footer:  menuNavHint(false),
	})
}
