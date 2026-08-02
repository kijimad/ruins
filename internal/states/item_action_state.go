package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// verbID は動詞タブ1つを識別する
type verbID string

const (
	verbExamine verbID = "examine" // 調べる
	verbPlace   verbID = "place"   // 置く
	verbConsume verbID = "consume" // 食べる。飲み物も含む
	verbRead    verbID = "read"    // 読む
	verbUse     verbID = "use"     // 使う
)

// itemVerb は動詞タブ1つ分の定義。Accept で対象アイテムを絞り、Exec で選択アイテムへ動詞を適用する。
// Exec が nil の動詞は実行を持たず、Enter で詳細モーダルを開く。調べるがこれに当たる。
type itemVerb struct {
	ID    verbID
	Label string
	// KeyHint はタブ見出しに添える直達ショートカットの表記。大文字は Shift 併用を表す。
	// 調べる X は Shift+x、置く d は KeyD をそのまま押す
	KeyHint string
	Accept  func(world w.World, entity ecs.Entity) bool
	Exec    func(world w.World, entity ecs.Entity) (es.Transition[w.World], error)
}

// verbs は表示順に並べた動詞タブの一覧。タブ順を兼ねる。
// 投げるは Throwable と ThrowActivity の実装後に足す。
func verbs() []itemVerb {
	return []itemVerb{
		{
			ID:      verbExamine,
			Label:   "調べる",
			KeyHint: "X",
			Accept:  func(_ w.World, _ ecs.Entity) bool { return true },
			Exec:    nil,
		},
		{
			ID:      verbPlace,
			Label:   "置く",
			KeyHint: "d",
			Accept:  func(_ w.World, _ ecs.Entity) bool { return true },
			Exec: func(_ w.World, entity ecs.Entity) (es.Transition[w.World], error) {
				// 選択済みアイテムを渡し、PlaceState をタイル選択から始めて二重選択を避ける
				return es.Transition[w.World]{Type: es.TransSwitch, NewStateFuncs: []es.StateFactory[w.World]{
					func() (es.State[w.World], error) { return &PlaceState{PresetItem: entity}, nil },
				}}, nil
			},
		},
		{
			ID:      verbConsume,
			Label:   "食べる",
			KeyHint: "e",
			Accept:  acceptConsumeFood,
			Exec:    execUseItem,
		},
		{
			ID:      verbRead,
			Label:   "読む",
			KeyHint: "r",
			Accept:  func(world w.World, entity ecs.Entity) bool { return world.Components.Book.Has(entity) },
			Exec:    execRead,
		},
		{
			ID:      verbUse,
			Label:   "使う",
			KeyHint: "t",
			Accept:  acceptUseTool,
			Exec:    execUseItem,
		},
	}
}

// acceptConsumeFood は栄養か回復を持つ消費物を食べるの対象とする。飲み物も含む
func acceptConsumeFood(world w.World, entity ecs.Entity) bool {
	if !world.Components.Consumable.Has(entity) {
		return false
	}
	return world.Components.ProvidesNutrition.Has(entity) || world.Components.ProvidesHealing.Has(entity)
}

// acceptUseTool は栄養も回復も持たない消費物を使うの対象とする
func acceptUseTool(world w.World, entity ecs.Entity) bool {
	if !world.Components.Consumable.Has(entity) {
		return false
	}
	return !world.Components.ProvidesNutrition.Has(entity) && !world.Components.ProvidesHealing.Has(entity)
}

// execUseItem は選択アイテムへ UseItemBehavior を適用しダンジョンへ戻る。
// 効果の有無で食べた・使ったのログ文言は UseItemBehavior 側が出し分ける。
func execUseItem(world w.World, entity ecs.Entity) (es.Transition[w.World], error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	if _, err := activity.Execute(&activity.UseItemBehavior{Target: entity}, player, world); err != nil {
		return es.Transition[w.World]{}, err
	}
	return es.Transition[w.World]{Type: es.TransPop}, nil
}

// execRead は選択した本の読書を開始しダンジョンへ戻る。読了は複数ターンにわたりダンジョンの進行が駆動する。
func execRead(world w.World, entity ecs.Entity) (es.Transition[w.World], error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	// Duration は上限見積もり。実際の完了は DoTurn 内の IsCompleted で判定する
	book := world.Components.Book.Get(entity)
	remaining := book.Effort.Max - book.Effort.Current
	if remaining <= 0 {
		remaining = 1
	}
	if _, err := activity.Execute(&activity.ReadBehavior{Target: entity, Duration: consts.Turn(remaining)}, player, world); err != nil {
		return es.Transition[w.World]{}, err
	}
	return es.Transition[w.World]{Type: es.TransPop}, nil
}

// verbByAction はダンジョン等からの直達アクションを対応する動詞へ対応づける
func verbByAction(action inputmapper.ActionID) (verbID, bool) {
	switch action {
	case inputmapper.ActionVerbExamine:
		return verbExamine, true
	case inputmapper.ActionVerbPlace:
		return verbPlace, true
	case inputmapper.ActionVerbConsume:
		return verbConsume, true
	case inputmapper.ActionVerbRead:
		return verbRead, true
	case inputmapper.ActionVerbUse:
		return verbUse, true
	default:
		return "", false
	}
}

// verbTabIndex は動詞のタブ位置を返す
func verbTabIndex(id verbID) int {
	for i, v := range verbs() {
		if v.ID == id {
			return i
		}
	}
	return 0
}

const itemActionMenuKey = "item_action"

// ItemActionState は動詞タブ画面のステート。上部の動詞タブを切り替え、その動詞に使えるアイテムだけを
// 名前のみで一覧する。Enter で即実行しダンジョンへ戻る。x で選択中アイテムの詳細モーダルを開く。
type ItemActionState struct {
	es.BaseState[w.World]
	initialVerb verbID // 開いた直後に表示する動詞タブ
	tabSeeded   bool   // 初期タブへ寄せたか
	showDetail  bool   // 詳細モーダルを表示中か
	rebuild     bool   // 次フレームで UI を作り直すか
	mount       *hooks.Mount[itemActionProps]
	widget      *ebitenui.UI
}

var _ es.State[w.World] = &ItemActionState{}

// NewItemActionState は動詞タブ画面を initial のタブで開くファクトリを返す
func NewItemActionState(initial verbID) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &ItemActionState{initialVerb: initial}, nil
	}
}

// OnPause はステートが一時停止される際に呼ばれる
func (st *ItemActionState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *ItemActionState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *ItemActionState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *ItemActionState) OnStart(_ w.World) error {
	st.mount = hooks.NewMount[itemActionProps]()
	return nil
}

// Update はステートの更新処理
func (st *ItemActionState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := st.handleInput(); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		// 左右キーはタブ切替に読み替える。ページ送りは持たない
		dispatch := action
		switch action {
		case inputmapper.ActionMenuLeft:
			dispatch = inputmapper.ActionMenuTabPrev
		case inputmapper.ActionMenuRight:
			dispatch = inputmapper.ActionMenuTabNext
		default:
			// 他のアクションはそのままタブメニューへ送る
		}
		st.mount.Dispatch(dispatch)
	}

	props := st.fetchProps(world)
	st.mount.SetProps(props)

	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(st.mount.Store(), itemActionMenuKey, hooks.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})

	// 開いた直後は指定タブへ寄せる。Store に直接書けないため公開 API の Dispatch で送る
	if !st.tabSeeded {
		for range verbTabIndex(st.initialVerb) {
			st.mount.Dispatch(inputmapper.ActionMenuTabNext)
		}
		st.tabSeeded = true
	}

	if st.mount.Update() || st.widget == nil || st.rebuild {
		st.widget = st.buildUI(world)
		st.rebuild = false
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理
func (st *ItemActionState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// handleInput はキー入力を Action に変換する。詳細モーダル表示中は閉じる操作だけを受ける
func (st *ItemActionState) handleInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if st.showDetail {
		if ki.IsKeyJustPressed(ebiten.KeyEscape) || ki.IsKeyJustPressed(ebiten.KeyX) || ki.IsEnterJustPressedOnce() {
			return inputmapper.ActionMenuCancel, true
		}
		return "", false
	}
	// x は詳細モーダル。X すなわち Shift+x は将来調べるタブへ充てるので Shift 無しに限る
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return HandleMenuInput()
}

// DoAction は Action を実行する
func (st *ItemActionState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	if st.showDetail {
		if action == inputmapper.ActionMenuCancel {
			st.showDetail = false
			st.rebuild = true
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.showDetail = true
		st.rebuild = true
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		return st.executeSelected(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// executeSelected は選択中アイテムへ現在の動詞を適用する。Exec を持たない調べるは詳細モーダルを開く
func (st *ItemActionState) executeSelected(world w.World) (es.Transition[w.World], error) {
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, itemActionMenuKey)
	vs := verbs()
	if menuState.TabIndex >= len(vs) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	verb := vs[menuState.TabIndex]
	if verb.Exec == nil {
		st.showDetail = true
		st.rebuild = true
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	props := st.mount.GetProps()
	tab := props.Tabs[menuState.TabIndex]
	if menuState.ItemIndex >= len(tab.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	return verb.Exec(world, tab.Items[menuState.ItemIndex].Entity)
}

// ================
// Props
// ================

type itemActionProps struct {
	Tabs []verbTabData
}

type verbTabData struct {
	ID    verbID
	Label string
	Key   string // タブ見出しに添える直達ショートカット表記
	Items []itemActionEntry
}

type itemActionEntry struct {
	Entity ecs.Entity
	Name   string
	Count  string
	Desc   string
}

func (st *ItemActionState) fetchProps(world w.World) itemActionProps {
	player, err := query.GetPlayerEntity(world)
	var backpack []ecs.Entity
	if err == nil {
		backpack = playerBackpackItems(world, player)
	}

	vs := verbs()
	tabs := make([]verbTabData, len(vs))
	for i, verb := range vs {
		items := make([]itemActionEntry, 0, len(backpack))
		for _, entity := range backpack {
			if !verb.Accept(world, entity) {
				continue
			}
			items = append(items, newItemActionEntry(world, entity))
		}
		tabs[i] = verbTabData{ID: verb.ID, Label: verb.Label, Key: verb.KeyHint, Items: items}
	}
	return itemActionProps{Tabs: tabs}
}

// playerBackpackItems はプレイヤーのバックパック内アイテムを表示順に返す
func playerBackpackItems(world w.World, player ecs.Entity) []ecs.Entity {
	var result []ecs.Entity
	q := ecs.NewFilter2[gc.LocationInBackpack, gc.Name](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		loc := world.Components.LocationInBackpack.Get(entity)
		if loc.Owner == player {
			result = append(result, entity)
		}
	}
	return query.SortEntities(world, result)
}

func newItemActionEntry(world w.World, entity ecs.Entity) itemActionEntry {
	entry := itemActionEntry{
		Entity: entity,
		Name:   world.Components.Name.Get(entity).Name,
	}
	if world.Components.Stackable.Has(entity) {
		entry.Count = fmt.Sprintf("%d", world.Components.Stackable.Get(entity).Count)
	}
	if world.Components.Description.Has(entity) {
		entry.Desc = world.Components.Description.Get(entity).Description
	}
	return entry
}

// ================
// buildUI
// ================

func (st *ItemActionState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, itemActionMenuKey)
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(1),
				widget.GridLayoutOpts.Spacing(0, theme.Space2),
				widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, false, true, false}),
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
	root.AddChild(styled.NewTitleText("アイテム操作", res))

	// Row 1: 動詞タブ帯を中央寄せ
	// タブ見出しに直達ショートカットを添える。調べる(X) 置く(d) の形
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		if tab.Key != "" {
			labels[i] = fmt.Sprintf("%s(%s)", tab.Label, tab.Key)
		} else {
			labels[i] = tab.Label
		}
	}
	tabRow := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	tabBar := styled.NewTabBar(labels, tabIndex, res)
	tabBar.GetWidget().LayoutData = widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter}
	tabRow.AddChild(tabBar)
	root.AddChild(tabRow)

	// Row 2: アイテム一覧。行は名前のみ
	root.AddChild(st.buildItemList(props, tabIndex, itemIndex, res))

	// Row 3: 選択中アイテムと x の案内
	root.AddChild(st.buildDescLine(props, tabIndex, itemIndex, res))

	ui := &ebitenui.UI{Container: root}

	if st.showDetail {
		if win := st.buildDetailWindow(world, props, tabIndex, itemIndex, res); win != nil {
			ui.AddWindow(win)
		}
	}

	return ui
}

// buildItemList は現在タブのアイテムを名前のみで縦1列に並べる
func (st *ItemActionState) buildItemList(props itemActionProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(props.Tabs) {
		return container
	}
	items := props.Tabs[tabIndex].Items
	if len(items) == 0 {
		container.AddChild(styled.NewDescriptionText("該当するアイテムがありません", res))
		return container
	}
	for i, item := range items {
		isSelected := i == itemIndex
		clr := theme.TextSecondary
		if isSelected {
			clr = theme.TextPrimary
		}
		label := item.Name
		if item.Count != "" {
			label = fmt.Sprintf("%s x%s", item.Name, item.Count)
		}
		container.AddChild(styled.NewListItemText(label, clr, isSelected, res))
	}
	return container
}

// buildDescLine は最下部に選択中アイテム名と詳細キーの案内を1行で置く
func (st *ItemActionState) buildDescLine(props itemActionProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	text := " "
	if tabIndex < len(props.Tabs) {
		items := props.Tabs[tabIndex].Items
		if itemIndex < len(items) {
			text = fmt.Sprintf("%s を選択中   x で詳細", items[itemIndex].Name)
		}
	}
	container.AddChild(styled.NewMenuText(text, res))
	return container
}

// buildDetailWindow は x で開く詳細モーダルを組み立てる。選択中アイテムの情報を出す
func (st *ItemActionState) buildDetailWindow(world w.World, props itemActionProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Window {
	if tabIndex >= len(props.Tabs) {
		return nil
	}
	items := props.Tabs[tabIndex].Items
	if itemIndex >= len(items) {
		return nil
	}
	item := items[itemIndex]

	content := styled.NewWindowContainer(res)
	if item.Count != "" {
		content.AddChild(styled.NewMenuText(fmt.Sprintf("所持 x%s", item.Count), res))
	}
	desc := item.Desc
	if desc == "" {
		desc = "説明はない"
	}
	content.AddChild(styled.NewDescriptionText(desc, res))

	title := styled.NewWindowHeaderContainer(item.Name, res)
	win := styled.NewSmallWindow(title, content)
	win.SetLocation(getCenterWinRect(world))
	return win
}
