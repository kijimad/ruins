package states

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// CraftMenuState はクラフトメニューのゲームステート
type CraftMenuState struct {
	es.BaseState[w.World]
	actionWin    menuscreen.ActionWindow // 合成のアクション選択ウィンドウ
	result       menuscreen.Detail       // 合成結果の詳細モーダル
	resultEntity ecs.Entity              // 直近で合成したアイテム
	rebuild      bool                    // 次フレームで UI を作り直すか
	menuMount    *hooks.Mount[craftProps]
	widget       *ebitenui.UI
}

// State interface ================

var _ es.State[w.World] = &CraftMenuState{}
var _ Configurable = &CraftMenuState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *CraftMenuState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

var _ es.ActionHandler[w.World] = &CraftMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CraftMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CraftMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *CraftMenuState) OnStart(_ w.World) error {
	st.menuMount = hooks.NewMount[craftProps]()
	st.actionWin = menuscreen.NewActionWindow(st.actionWindowContent)
	st.result = menuscreen.NewDetail(st.resultDetailContent)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CraftMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CraftMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 入力処理。合成結果・アクション窓が開いていればそちらが優先し、通常のメニュー入力は止まる
	if st.result.Active() {
		if st.result.HandleInput(world) {
			st.rebuild = true
		}
	} else if st.actionWin.Active() {
		dirty, err := st.actionWin.HandleInput(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if dirty {
			st.rebuild = true
		}
	} else if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	// UseTabMenuでreducerを登録・更新
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(st.menuMount.Store(), "craft", hooks.TabMenuConfig{
		TabCount:     len(props.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: menuItemsPerPage,
	})

	if st.menuMount.Update() || st.widget == nil || st.rebuild {
		st.widget = st.buildUI(world)
		st.rebuild = false
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *CraftMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *CraftMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *CraftMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		st.actionWin.Open()
		st.rebuild = true
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("craftMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type craftProps struct {
	Tabs []craftTabData
}

type craftTabData struct {
	ID    string
	Label string
	Items []craftItemData
}

type craftItemData struct {
	RecipeName string
	CanCraft   bool
}

func (st *CraftMenuState) fetchProps(world w.World) craftProps {
	return craftProps{
		Tabs: st.createTabs(world),
	}
}

func (st *CraftMenuState) createTabs(world w.World) []craftTabData {
	return []craftTabData{
		{ID: "consumables", Label: "道具", Items: st.createMenuItems(world, st.queryMenuConsumable(world))},
		{ID: "weapons", Label: "武器", Items: st.createMenuItems(world, st.queryMenuWeapon(world))},
		{ID: "wearables", Label: "装備", Items: st.createMenuItems(world, st.queryMenuWearable(world))},
	}
}

func (st *CraftMenuState) createMenuItems(world w.World, recipeNames []string) []craftItemData {
	items := make([]craftItemData, len(recipeNames))

	for i, recipeName := range recipeNames {
		canCraft, _ := gameaction.CanCraft(world, recipeName)
		items[i] = craftItemData{
			RecipeName: recipeName,
			CanCraft:   canCraft,
		}
	}

	return items
}

func (st *CraftMenuState) queryMenuConsumable(world w.World) []string {
	var items []string

	for _, recipe := range raw.PtrSlice(world.Resources.RawMaster.Recipes) {
		spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, recipe.Name)
		if err != nil {
			continue
		}
		if spec.Consumable != nil {
			items = append(items, recipe.Name)
		}
	}

	slices.Sort(items)
	return items
}

func (st *CraftMenuState) queryMenuWeapon(world w.World) []string {
	var items []string

	for _, recipe := range raw.PtrSlice(world.Resources.RawMaster.Recipes) {
		spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, recipe.Name)
		if err != nil {
			continue
		}
		// TODO: カテゴリ定義で判定したい
		if spec.Melee != nil || spec.Fire != nil {
			items = append(items, recipe.Name)
		}
	}

	slices.Sort(items)
	return items
}

func (st *CraftMenuState) queryMenuWearable(world w.World) []string {
	var items []string

	for _, recipe := range raw.PtrSlice(world.Resources.RawMaster.Recipes) {
		spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, recipe.Name)
		if err != nil {
			continue
		}
		if spec.Wearable != nil {
			items = append(items, recipe.Name)
		}
	}

	slices.Sort(items)
	return items
}

// ================
// Action Window
// ================

// actionWindowContent は現在カーソルが当たっているレシピの見出しと選択肢を返す。アクション窓の唯一の定義点。
// 合成の実行内容も Run に閉じ込め、合成・閉じるを1箇所で定義する
func (st *CraftMenuState) actionWindowContent(_ w.World) (string, []menuscreen.Action, bool) {
	item, ok := st.selectedRecipe()
	if !ok || item.RecipeName == "" {
		return "", nil, false
	}
	var actions []menuscreen.Action
	if item.CanCraft {
		recipeName := item.RecipeName
		actions = append(actions, menuscreen.Action{Label: TextCraft, Run: func(world w.World) error {
			resultEntity, err := gameaction.Craft(world, recipeName)
			if err != nil {
				return fmt.Errorf("合成に失敗: %w", err)
			}
			st.resultEntity = resultEntity
			st.result.Open()
			st.rebuild = true
			return nil
		}})
	}
	actions = append(actions, menuscreen.Action{Label: TextClose})
	return "アクション選択", actions, true
}

// selectedRecipe は現在カーソルが当たっているレシピを返す
func (st *CraftMenuState) selectedRecipe() (craftItemData, bool) {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "craft")
	if !ok || menuState.TabIndex >= len(props.Tabs) {
		return craftItemData{}, false
	}
	items := props.Tabs[menuState.TabIndex].Items
	if menuState.ItemIndex >= len(items) {
		return craftItemData{}, false
	}
	return items[menuState.ItemIndex], true
}

// resultDetailContent は直近で合成したアイテムを詳細モーダルの内容にする
func (st *CraftMenuState) resultDetailContent(world w.World) (menuscreen.DetailContent, bool) {
	if !world.ECS.Alive(st.resultEntity) {
		return menuscreen.DetailContent{}, false
	}
	return entityDetailContent(world, st.resultEntity), true
}

// ================
// buildUI
// ================

func (st *CraftMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "craft")
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	// カテゴリは標準のタブ帯に寄せる。本体は アイテム一覧+性能レシピ / 説明文 のグリッド
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}

	content := newThreeColContent(
		nil,
		st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res),
		st.buildDetailContainer(world, props, tabIndex, itemIndex, res),
		st.buildDescContainer(world, props.Tabs, tabIndex, itemIndex, res),
	)

	eui := newTabScreenUI(res, tabScreen{TabLabels: labels, TabIndex: tabIndex, Content: content, Footer: menuNavHint(true)})

	// 合成結果の詳細モーダル、無ければアクション選択ウィンドウを重ねる
	if st.result.Active() {
		if win := st.result.Window(world, getCenterWinRect(world)); win != nil {
			eui.AddWindow(win)
		}
	} else if st.actionWin.Active() {
		if win := st.actionWin.Window(world, getCenterWinRect(world)); win != nil {
			eui.AddWindow(win)
		}
	}

	return eui
}

func (st *CraftMenuState) buildItemContainer(tabs []craftTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(tabs) {
		return container
	}

	currentTab := tabs[tabIndex]
	pg := pagination.New(itemIndex, len(currentTab.Items), menuItemsPerPage)

	// ページインジケーター
	container.AddChild(newPageIndicator(pg, res))

	columnWidths := []int{20, 180}

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		styled.NewTableRow(table, columnWidths, []string{"", entry.Item.RecipeName}, nil, new(pg.IsSelectedInPage(entry.Index)), res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(レシピなし)", res))
	}

	return container
}

func (st *CraftMenuState) buildDetailContainer(world w.World, props craftProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	specContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	recipeContainer := styled.NewVerticalContainer()

	if tabIndex >= len(props.Tabs) {
		col := styled.NewVerticalContainer()
		col.AddChild(specContainer)
		col.AddChild(recipeContainer)
		return col
	}
	tab := props.Tabs[tabIndex]
	if itemIndex >= len(tab.Items) {
		col := styled.NewVerticalContainer()
		col.AddChild(specContainer)
		col.AddChild(recipeContainer)
		return col
	}
	item := tab.Items[itemIndex]

	// 性能表示
	spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, item.RecipeName)
	if err == nil {
		views.UpdateSpecFromSpec(world, specContainer, spec)
	}

	// レシピ表示
	if err == nil && spec.Recipe != nil {
		st.buildRecipeList(world, recipeContainer, spec.Recipe, res)
	}

	col := styled.NewVerticalContainer()
	col.AddChild(specContainer)
	col.AddChild(recipeContainer)
	return col
}

func (st *CraftMenuState) buildRecipeList(world w.World, container *widget.Container, recipe *gc.Recipe, res resources.UIResources) {
	for _, input := range recipe.Inputs {
		var currentAmount int
		if entity, found := query.FindStackableInInventory(world, input.Name); found {
			currentAmount = query.GetEntityCount(world, entity)
		}
		str := fmt.Sprintf("%s %d pcs\n    所持: %d pcs", input.Name, input.Amount, currentAmount)
		var textColor color.RGBA
		if currentAmount >= input.Amount {
			textColor = theme.StatusSuccess
		} else {
			textColor = theme.StatusDanger
		}

		container.AddChild(styled.NewBodyText(str, textColor, res))
	}
}

func (st *CraftMenuState) buildDescContainer(world w.World, tabs []craftTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	desc := " "

	if tabIndex < len(tabs) && itemIndex < len(tabs[tabIndex].Items) {
		item := tabs[tabIndex].Items[itemIndex]
		spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, item.RecipeName)
		if err == nil && spec.Description != nil {
			desc = spec.Description.Description
		}
	}

	if desc == "" {
		desc = " "
	}
	container.AddChild(styled.NewMenuText(desc, res))
	return container
}
