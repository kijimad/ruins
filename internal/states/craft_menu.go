package states

import (
	"fmt"
	"image/color"
	"log"
	"sort"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// CraftMenuState はクラフトメニューのゲームステート
type CraftMenuState struct {
	es.BaseState[w.World]
	ui *ebitenui.UI

	menuView            *tabmenu.View
	selectedItem        string            // 選択中のレシピ名
	itemDesc            *widget.Text      // アイテムの概要
	specContainer       *widget.Container // 性能表示のコンテナ
	recipeList          *widget.Container // レシピリストのコンテナ
	resultWindow        *widget.Window    // 合成結果ウィンドウ
	rootContainer       *widget.Container
	tabDisplayContainer *widget.Container // タブ表示のコンテナ
	categoryContainer   *widget.Container // カテゴリ一覧のコンテナ

	// アクション選択ウィンドウ用
	actionWindow     *widget.Window // アクション選択ウィンドウ
	actionFocusIndex int            // アクションウィンドウ内のフォーカス
	actionItems      []string       // アクション項目リスト
	isWindowMode     bool           // ウィンドウ操作モードかどうか

	// 結果ウィンドウ用
	resultFocusIndex int        // 結果ウィンドウ内のフォーカス
	resultItems      []string   // 結果ウィンドウの項目リスト
	resultEntity     ecs.Entity // 生成された結果アイテムのエンティティ
	isResultMode     bool       // 結果ウィンドウ操作モードかどうか
}

func (st CraftMenuState) String() string {
	return "CraftMenu"
}

// State interface ================

var _ es.State[w.World] = &CraftMenuState{}
var _ es.ActionHandler[w.World] = &CraftMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CraftMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CraftMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *CraftMenuState) OnStart(world w.World) error {
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CraftMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CraftMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// キー入力をActionに変換
	var action inputmapper.ActionID
	var ok bool
	if st.isWindowMode || st.isResultMode {
		action, ok = HandleWindowInput()
	} else {
		action, ok = st.HandleInput(world.Config)
	}

	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	// アクションウィンドウまたは結果ウィンドウ表示中はTabMenuの更新をスキップ
	if !st.isWindowMode && !st.isResultMode {
		if err := st.menuView.Update(); err != nil {
			return es.Transition[w.World]{}, err
		}
	}
	st.ui.Update()

	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *CraftMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.ui.Draw(screen)
	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *CraftMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}

	return "", false
}

// DoAction はActionを実行する
func (st *CraftMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	// ウィンドウモード時のアクション処理
	if st.isWindowMode {
		switch action {
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			if UpdateFocusIndex(action, &st.actionFocusIndex, len(st.actionItems)) {
				st.updateActionWindowDisplay(world)
			}
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowConfirm:
			if err := st.executeActionItem(world); err != nil {
				return es.Transition[w.World]{}, err
			}
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowCancel:
			st.closeActionWindow()
			return es.Transition[w.World]{Type: es.TransNone}, nil
		default:
			return es.Transition[w.World]{}, fmt.Errorf("ウィンドウモード時の未知のアクション: %s", action)
		}
	}

	// 結果ウィンドウモード時のアクション処理
	if st.isResultMode {
		switch action {
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			if UpdateFocusIndex(action, &st.resultFocusIndex, len(st.resultItems)) {
				st.updateResultWindowDisplay(world)
			}
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowConfirm:
			st.executeResultItem(world)
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowCancel:
			st.closeResultWindow()
			return es.Transition[w.World]{Type: es.TransNone}, nil
		default:
			return es.Transition[w.World]{}, fmt.Errorf("結果ウィンドウモード時の未知のアクション: %s", action)
		}
	}

	switch action {
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// ================

func (st *CraftMenuState) initUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources

	// TabMenuの設定
	tabs := st.createTabs(world)
	config := tabmenu.Config{
		Tabs:             tabs,
		InitialTabIndex:  0,
		InitialItemIndex: 0,
		WrapNavigation:   true,
		ItemsPerPage:     20,
	}

	callbacks := tabmenu.Callbacks{
		OnSelectItem: func(_ int, _ int, tab tabmenu.TabItem, item tabmenu.Item) error {
			return st.handleItemSelection(world, tab, item)
		},
		OnCancel: func() {
			// Escapeで前の画面に戻る
			st.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		},
		OnTabChange: func(_, _ int, _ tabmenu.TabItem) {
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
			st.updateCategoryDisplay(world)
		},
		OnItemChange: func(_ int, _, _ int, item tabmenu.Item) error {
			if err := st.handleItemChange(world, item); err != nil {
				return err
			}
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
			return nil
		},
	}

	st.menuView = tabmenu.NewView(config, callbacks, world)

	// アイテムの説明文
	itemDescContainer := styled.NewRowContainer()
	st.itemDesc = styled.NewMenuText(" ", world.Resources.UIResources) // 空文字だと初期状態の縦サイズがなくなる
	itemDescContainer.AddChild(st.itemDesc)

	st.specContainer = styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	st.recipeList = styled.NewVerticalContainer()

	// 初期状態の表示を更新
	if err := st.updateInitialItemDisplay(world); err != nil {
		log.Fatalf("Failed to update initial item display: %v", err)
	}

	// タブ表示のコンテナを作成
	st.tabDisplayContainer = styled.NewVerticalContainer()
	st.createTabDisplayUI(world)

	// カテゴリ一覧のコンテナを作成（横並び）
	st.categoryContainer = styled.NewRowContainer()
	st.createCategoryDisplayUI(world)

	st.rootContainer = styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	{
		// 3x3グリッドレイアウト: 9個の要素が必要
		// 1行目
		st.rootContainer.AddChild(styled.NewTitleText("合成", world.Resources.UIResources))
		st.rootContainer.AddChild(st.categoryContainer) // カテゴリ一覧の表示
		st.rootContainer.AddChild(widget.NewContainer())

		// 2行目
		st.rootContainer.AddChild(st.tabDisplayContainer) // タブとアイテム一覧の表示
		st.rootContainer.AddChild(widget.NewContainer())
		st.rootContainer.AddChild(styled.NewVSplitContainer(st.specContainer, st.recipeList))

		// 3行目
		st.rootContainer.AddChild(itemDescContainer)
		st.rootContainer.AddChild(widget.NewContainer()) // 空
		st.rootContainer.AddChild(widget.NewContainer()) // 空
	}

	return &ebitenui.UI{Container: st.rootContainer}
}

// createTabs はTabMenuで使用するタブを作成する
func (st *CraftMenuState) createTabs(world w.World) []tabmenu.TabItem {
	tabs := []tabmenu.TabItem{
		{
			ID:    "consumables",
			Label: "道具",
			Items: st.createMenuItems(world, st.queryMenuConsumable(world)),
		},
		{
			ID:    "weapons",
			Label: "武器",
			Items: st.createMenuItems(world, st.queryMenuWeapon(world)),
		},
		{
			ID:    "wearables",
			Label: "装備",
			Items: st.createMenuItems(world, st.queryMenuWearable(world)),
		},
	}

	return tabs
}

// createMenuItems はレシピ名リストをMenuItemに変換する
func (st *CraftMenuState) createMenuItems(world w.World, recipeNames []string) []tabmenu.Item {
	items := make([]tabmenu.Item, len(recipeNames))

	for i, recipeName := range recipeNames {
		canCraft, _ := worldhelper.CanCraft(world, recipeName)

		items[i] = tabmenu.Item{
			ID:       fmt.Sprintf("recipe_%s", recipeName),
			Label:    recipeName,
			UserData: recipeName,
			Disabled: !canCraft, // 合成不可能な場合はDisabled
		}
	}

	return items
}

// handleItemSelection はアイテム選択時の処理
func (st *CraftMenuState) handleItemSelection(world w.World, _ tabmenu.TabItem, item tabmenu.Item) error {
	recipeName, ok := item.UserData.(string)
	if !ok {
		return fmt.Errorf("unexpected item UserData")
	}

	st.selectedItem = recipeName
	st.showActionWindow(world, recipeName)
	return nil
}

// handleItemChange はアイテム変更時の処理（カーソル移動）
func (st *CraftMenuState) handleItemChange(world w.World, item tabmenu.Item) error {
	// 無効なアイテムの場合は何もしない
	if item.UserData == nil {
		st.itemDesc.Label = " "
		st.specContainer.RemoveChildren()
		st.recipeList.RemoveChildren()
		return nil
	}

	recipeName, ok := item.UserData.(string)
	if !ok {
		return fmt.Errorf("unexpected item UserData")
	}

	// RawMasterからEntitySpecを取得
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	spec, err := rawMaster.NewRecipeSpec(recipeName)
	if err != nil {
		st.itemDesc.Label = TextNoDescription
		st.specContainer.RemoveChildren()
		st.recipeList.RemoveChildren()
		return err
	}

	// Descriptionを取得
	if spec.Description != nil {
		st.itemDesc.Label = spec.Description.Description
	} else {
		st.itemDesc.Label = TextNoDescription
	}

	// EntitySpecから性能表示を更新
	views.UpdateSpecFromSpec(world, st.specContainer, spec)
	if err := st.updateRecipeList(world, spec.Recipe); err != nil {
		return err
	}
	return nil
}

func (st *CraftMenuState) queryMenuConsumable(world w.World) []string {
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	var items []string

	// 全レシピから消耗品を抽出
	for recipeName := range rawMaster.RecipeIndex {
		spec, err := rawMaster.NewRecipeSpec(recipeName)
		if err != nil {
			continue
		}
		// 消耗品
		if spec.Consumable != nil {
			items = append(items, recipeName)
		}
	}

	sort.Strings(items)
	return items
}

func (st *CraftMenuState) queryMenuWeapon(world w.World) []string {
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	var items []string

	// 全レシピから武器を抽出
	for recipeName := range rawMaster.RecipeIndex {
		spec, err := rawMaster.NewRecipeSpec(recipeName)
		if err != nil {
			continue
		}
		if spec.Weapon != nil {
			items = append(items, recipeName)
		}
	}

	sort.Strings(items)
	return items
}

func (st *CraftMenuState) queryMenuWearable(world w.World) []string {
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	var items []string

	// 全レシピから装備品を抽出
	for recipeName := range rawMaster.RecipeIndex {
		spec, err := rawMaster.NewRecipeSpec(recipeName)
		if err != nil {
			continue
		}
		if spec.Wearable != nil {
			items = append(items, recipeName)
		}
	}

	sort.Strings(items)
	return items
}

// showResultWindow は合成結果ウィンドウを表示する
func (st *CraftMenuState) showResultWindow(world w.World, entity ecs.Entity) {
	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("合成結果", world.Resources.UIResources)
	st.resultWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// 結果項目を準備
	st.resultItems = []string{TextClose}
	st.resultFocusIndex = 0
	st.resultEntity = entity // 生成されたアイテムのエンティティを保存
	st.isResultMode = true

	// UI要素を作成（表示のみ、操作はキーボードで行う）
	st.createResultWindowUI(world, windowContainer, entity)

	st.resultWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.resultWindow)
}

// createResultWindowUI は結果ウィンドウのUI要素を作成する
func (st *CraftMenuState) createResultWindowUI(world w.World, container *widget.Container, entity ecs.Entity) {
	// アイテム詳細を表示
	views.UpdateSpec(world, container, entity)

	st.updateResultWindowDisplay(world)
}

// updateResultWindowDisplay は結果ウィンドウの表示を更新する
func (st *CraftMenuState) updateResultWindowDisplay(world w.World) {
	if st.resultWindow == nil {
		return
	}

	// 既存のウィンドウを閉じて新しく作成
	st.resultWindow.Close()

	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("合成結果", world.Resources.UIResources)
	st.resultWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アイテム詳細を表示（生成されたアイテムの値を使用）
	views.UpdateSpec(world, windowContainer, st.resultEntity)

	// ボタン項目を表示
	for i, action := range st.resultItems {
		isSelected := i == st.resultFocusIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, world.Resources.UIResources)
		windowContainer.AddChild(actionWidget)
	}

	st.resultWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.resultWindow)
}

func (st *CraftMenuState) updateRecipeList(world w.World, recipe *gc.Recipe) error {
	st.recipeList.RemoveChildren()

	if recipe == nil {
		return fmt.Errorf("recipeがnilです")
	}

	for _, input := range recipe.Inputs {
		var currentAmount int
		if stackableEntity, found := worldhelper.FindStackableInInventory(world, input.Name); found {
			item := world.Components.Item.Get(stackableEntity).(*gc.Item)
			currentAmount = item.Count
		}
		str := fmt.Sprintf("%s %d pcs\n    所持: %d pcs", input.Name, input.Amount, currentAmount)
		var color color.RGBA
		if currentAmount >= input.Amount {
			color = consts.SuccessColor
		} else {
			color = consts.DangerColor
		}

		st.recipeList.AddChild(styled.NewBodyText(str, color, world.Resources.UIResources))
	}
	return nil
}

// showActionWindow はアクションウィンドウを表示する
func (st *CraftMenuState) showActionWindow(world w.World, recipeName string) {
	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", world.Resources.UIResources)
	st.actionWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アクション項目を準備
	st.actionItems = []string{}
	st.selectedItem = recipeName

	// 合成可能かチェック
	if canCraft, _ := worldhelper.CanCraft(world, recipeName); canCraft {
		st.actionItems = append(st.actionItems, "合成する")
	}
	st.actionItems = append(st.actionItems, TextClose)

	st.actionFocusIndex = 0
	st.isWindowMode = true

	// UI要素を作成（表示のみ、操作はキーボードで行う）
	st.updateActionWindowDisplay(world)

	st.actionWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.actionWindow)
}

// updateActionWindowDisplay はアクションウィンドウの表示を更新する
func (st *CraftMenuState) updateActionWindowDisplay(world w.World) {
	if st.actionWindow == nil {
		return
	}

	// 既存のウィンドウを閉じて新しく作成
	st.actionWindow.Close()

	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", world.Resources.UIResources)
	st.actionWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アクション項目を表示
	for i, action := range st.actionItems {
		isSelected := i == st.actionFocusIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, world.Resources.UIResources)
		windowContainer.AddChild(actionWidget)
	}

	st.actionWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.actionWindow)
}

// closeActionWindow はアクションウィンドウを閉じる
func (st *CraftMenuState) closeActionWindow() {
	if st.actionWindow != nil {
		st.actionWindow.Close()
		st.actionWindow = nil
	}
	st.isWindowMode = false
	st.actionFocusIndex = 0
	st.actionItems = nil
}

// closeResultWindow は結果ウィンドウを閉じる
func (st *CraftMenuState) closeResultWindow() {
	if st.resultWindow != nil {
		st.resultWindow.Close()
		st.resultWindow = nil
	}
	st.isResultMode = false
	st.resultFocusIndex = 0
	st.resultItems = nil
	st.resultEntity = 0 // エンティティIDをリセット
}

// executeResultItem は選択された結果項目を実行する
func (st *CraftMenuState) executeResultItem(_ w.World) {
	if st.resultFocusIndex >= len(st.resultItems) {
		return
	}

	selectedAction := st.resultItems[st.resultFocusIndex]

	switch selectedAction {
	case TextClose:
		st.closeResultWindow()
	}
}

// executeActionItem は選択されたアクション項目を実行する
func (st *CraftMenuState) executeActionItem(world w.World) error {
	if st.actionFocusIndex >= len(st.actionItems) {
		return nil
	}

	selectedAction := st.actionItems[st.actionFocusIndex]
	recipeName := st.selectedItem

	switch selectedAction {
	case "合成する":
		resultEntity, err := worldhelper.Craft(world, recipeName)
		if err != nil {
			return err
		}

		// レシピリストを更新
		rawMaster := world.Resources.RawMaster.(*raw.Master)
		var spec gc.EntitySpec
		spec, err = rawMaster.NewRecipeSpec(recipeName)
		if err != nil {
			return err
		}
		if err = st.updateRecipeList(world, spec.Recipe); err != nil {
			return err
		}

		st.closeActionWindow()
		st.showResultWindow(world, *resultEntity)
		st.reloadTabs(world)
		st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
		st.updateCategoryDisplay(world)
	case TextClose:
		st.closeActionWindow()
	}
	return nil
}

// reloadTabs はタブの内容を再読み込みする
func (st *CraftMenuState) reloadTabs(world w.World) {
	newTabs := st.createTabs(world)
	st.menuView.UpdateTabs(newTabs)
	// UpdateTabs後に表示を更新
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
}

// createTabDisplayUI はタブ表示UIを作成する
func (st *CraftMenuState) createTabDisplayUI(_ w.World) {
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
}

// createCategoryDisplayUI はカテゴリ表示UIを作成する
func (st *CraftMenuState) createCategoryDisplayUI(world w.World) {
	st.updateCategoryDisplay(world)
}

// updateCategoryDisplay はカテゴリ表示を更新する
func (st *CraftMenuState) updateCategoryDisplay(world w.World) {
	// 既存の子要素をクリア
	st.categoryContainer.RemoveChildren()

	// 全カテゴリを横並びで表示
	currentTabIndex := st.menuView.GetCurrentTabIndex()
	tabs := st.createTabs(world) // 最新のタブ情報を取得

	for i, tab := range tabs {
		isSelected := i == currentTabIndex
		if isSelected {
			// 選択中のカテゴリは背景色付きで明るい文字色
			categoryWidget := styled.NewListItemText(tab.Label, consts.TextColor, true, world.Resources.UIResources)
			st.categoryContainer.AddChild(categoryWidget)
		} else {
			// 非選択のカテゴリは背景なしでグレー文字色
			categoryWidget := styled.NewListItemText(tab.Label, consts.ForegroundColor, false, world.Resources.UIResources)
			st.categoryContainer.AddChild(categoryWidget)
		}
	}
}

// updateInitialItemDisplay は初期状態のアイテム表示を更新する
func (st *CraftMenuState) updateInitialItemDisplay(world w.World) error {
	currentTab := st.menuView.GetCurrentTab()
	currentItemIndex := st.menuView.GetCurrentItemIndex()

	if len(currentTab.Items) > 0 && currentItemIndex >= 0 && currentItemIndex < len(currentTab.Items) {
		currentItem := currentTab.Items[currentItemIndex]
		if err := st.handleItemChange(world, currentItem); err != nil {
			return err
		}
	}
	return nil
}
