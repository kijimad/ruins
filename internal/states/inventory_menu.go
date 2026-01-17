package states

import (
	"fmt"
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/actions"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/logger"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InventoryMenuState はインベントリメニューのゲームステート
type InventoryMenuState struct {
	es.BaseState[w.World]
	ui *ebitenui.UI

	menuView            *tabmenu.View
	selectedItem        ecs.Entity        // 選択中のアイテム
	itemDesc            *widget.Text      // アイテムの概要
	specContainer       *widget.Container // 性能表示のコンテナ
	rootContainer       *widget.Container
	tabDisplayContainer *widget.Container // タブ表示のコンテナ
	categoryContainer   *widget.Container // カテゴリ一覧のコンテナ

	// アクション選択ウィンドウ用
	actionWindow     *widget.Window // アクション選択ウィンドウ
	actionFocusIndex int            // アクションウィンドウ内のフォーカス
	actionItems      []string       // アクション項目リスト
	isWindowMode     bool           // ウィンドウ操作モードかどうか
}

func (st InventoryMenuState) String() string {
	return "InventoryMenu"
}

// State interface ================

var _ es.State[w.World] = &InventoryMenuState{}
var _ es.ActionHandler[w.World] = &InventoryMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *InventoryMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *InventoryMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *InventoryMenuState) OnStart(world w.World) error {
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *InventoryMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *InventoryMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// InventoryChangedSystemを実行して所持重量を更新
	for _, updater := range []w.Updater{
		&gs.InventoryChangedSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	// キー入力をActionに変換
	var action inputmapper.ActionID
	var ok bool
	if st.isWindowMode {
		action, ok = HandleWindowInput()
	} else {
		action, ok = st.HandleInput()
	}

	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	// アクションウィンドウ表示中はTabMenuの更新をスキップ
	if !st.isWindowMode {
		if err := st.menuView.Update(); err != nil {
			return es.Transition[w.World]{}, err
		}
	}
	st.ui.Update()

	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *InventoryMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.ui.Draw(screen)
	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *InventoryMenuState) HandleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}

	return "", false
}

// DoAction はActionを実行する
func (st *InventoryMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
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

func (st *InventoryMenuState) initUI(world w.World) *ebitenui.UI {
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
		st.rootContainer.AddChild(styled.NewTitleText("インベントリ", world.Resources.UIResources))
		st.rootContainer.AddChild(st.categoryContainer) // カテゴリ一覧の表示
		st.rootContainer.AddChild(widget.NewContainer())

		// 2行目
		st.rootContainer.AddChild(st.tabDisplayContainer) // タブとアイテム一覧の表示
		st.rootContainer.AddChild(widget.NewContainer())
		st.rootContainer.AddChild(st.specContainer)

		// 3行目
		st.rootContainer.AddChild(itemDescContainer)
		st.rootContainer.AddChild(widget.NewContainer()) // 空
		st.rootContainer.AddChild(widget.NewContainer()) // 空
	}

	return &ebitenui.UI{Container: st.rootContainer}
}

// createTabs はTabMenuで使用するタブを作成する
func (st *InventoryMenuState) createTabs(world w.World) []tabmenu.TabItem {
	tabs := []tabmenu.TabItem{
		{
			ID:    "items",
			Label: "道具",
			Items: st.createMenuItems(world, st.queryMenuItem(world)),
		},
		{
			ID:    "weapons",
			Label: "武器",
			Items: st.createMenuItems(world, st.queryMenuWeapon(world)),
		},
		{
			ID:    "wearables",
			Label: "防具",
			Items: st.createMenuItems(world, st.queryMenuWearable(world)),
		},
	}

	return tabs
}

// createMenuItems はECSエンティティをMenuItemに変換する
func (st *InventoryMenuState) createMenuItems(world w.World, entities []ecs.Entity) []tabmenu.Item {
	items := make([]tabmenu.Item, len(entities))

	for i, entity := range entities {
		name := world.Components.Name.Get(entity).(*gc.Name).Name

		item := tabmenu.Item{
			ID:       fmt.Sprintf("entity_%d", entity),
			Label:    name,
			UserData: entity,
		}

		// Stackableコンポーネントがあれば個数を表示する
		if entity.HasComponent(world.Components.Stackable) {
			itemComp := world.Components.Item.Get(entity).(*gc.Item)
			item.AdditionalLabels = []string{fmt.Sprintf("x%d", itemComp.Count)}
		}

		items[i] = item
	}

	return items
}

// handleItemSelection はアイテム選択時の処理
func (st *InventoryMenuState) handleItemSelection(world w.World, _ tabmenu.TabItem, item tabmenu.Item) error {
	entity, ok := item.UserData.(ecs.Entity)
	if !ok {
		return fmt.Errorf("unexpected item UserData")
	}

	st.selectedItem = entity
	st.showActionWindow(world, entity)
	return nil
}

// handleItemChange はアイテム変更時の処理（カーソル移動）
func (st *InventoryMenuState) handleItemChange(world w.World, item tabmenu.Item) error {
	// 無効なアイテムの場合は何もしない
	if item.UserData == nil {
		st.itemDesc.Label = " "
		st.specContainer.RemoveChildren()
		return nil
	}

	entity, ok := item.UserData.(ecs.Entity)
	if !ok {
		return fmt.Errorf("unexpected item UserData")
	}

	// Descriptionコンポーネントの存在チェック
	if !entity.HasComponent(world.Components.Description) {
		st.itemDesc.Label = TextNoDescription
		st.specContainer.RemoveChildren()
		return nil
	}

	desc := world.Components.Description.Get(entity).(*gc.Description)
	if desc == nil {
		st.itemDesc.Label = TextNoDescription
		st.specContainer.RemoveChildren()
		return nil
	}

	st.itemDesc.Label = desc.Description
	views.UpdateSpec(world, st.specContainer, entity)
	return nil
}

// closeActionWindow はアクションウィンドウを閉じる
func (st *InventoryMenuState) closeActionWindow() {
	if st.actionWindow != nil {
		st.actionWindow.Close()
		st.actionWindow = nil
	}
	st.isWindowMode = false
	st.actionFocusIndex = 0
	st.actionItems = nil
}

// showActionWindow はアクションウィンドウを表示する
func (st *InventoryMenuState) showActionWindow(world w.World, entity ecs.Entity) {
	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", world.Resources.UIResources)
	st.actionWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アクション項目を準備
	st.actionItems = []string{}
	st.selectedItem = entity

	// 使用可能なアクションを登録
	if entity.HasComponent(world.Components.Consumable) {
		st.actionItems = append(st.actionItems, "使う")
	}
	st.actionItems = append(st.actionItems, "捨てる")
	st.actionItems = append(st.actionItems, TextClose)

	st.actionFocusIndex = 0
	st.isWindowMode = true

	// UI要素を作成（表示のみ、操作はキーボードで行う）
	st.createActionWindowUI(world, windowContainer, entity)

	st.actionWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.actionWindow)
}

// createActionWindowUI はアクションウィンドウのUI要素を作成する
func (st *InventoryMenuState) createActionWindowUI(world w.World, _ *widget.Container, _ ecs.Entity) {
	st.updateActionWindowDisplay(world)
}

// updateActionWindowDisplay はアクションウィンドウの表示を更新する
func (st *InventoryMenuState) updateActionWindowDisplay(world w.World) {
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

// executeActionItem は選択されたアクション項目を実行する
func (st *InventoryMenuState) executeActionItem(world w.World) error {
	if st.actionFocusIndex >= len(st.actionItems) {
		return nil
	}

	selectedAction := st.actionItems[st.actionFocusIndex]

	switch selectedAction {
	case "使う":
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		manager := actions.NewActivityManager(logger.New(logger.CategoryAction))
		params := actions.ActionParams{
			Actor:  playerEntity,
			Target: &st.selectedItem,
		}
		_, err = manager.Execute(&actions.UseItemActivity{}, params, world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		st.closeActionWindow()
		st.reloadTabs(world)
		st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
		st.updateCategoryDisplay(world)
	case "捨てる":
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		dropActivity := &actions.DropActivity{
			Target: st.selectedItem,
		}

		manager := actions.NewActivityManager(logger.New(logger.CategoryAction))
		params := actions.ActionParams{
			Actor: playerEntity,
		}
		_, err = manager.Execute(dropActivity, params, world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		st.closeActionWindow()
		st.reloadTabs(world)
		st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
		st.updateCategoryDisplay(world)
	case TextClose:
		st.closeActionWindow()
	}

	return nil
}

// reloadTabs はタブの内容を再読み込みする
func (st *InventoryMenuState) reloadTabs(world w.World) {
	newTabs := st.createTabs(world)
	st.menuView.UpdateTabs(newTabs)
	// UpdateTabs後に表示を更新
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
}

func (st *InventoryMenuState) queryMenuItem(world w.World) []ecs.Entity {
	items := []ecs.Entity{}

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInBackpack,
		world.Components.Wearable.Not(),
		world.Components.Weapon.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		items = append(items, entity)
	}))

	return worldhelper.SortEntities(world, items)
}

func (st *InventoryMenuState) queryMenuWeapon(world w.World) []ecs.Entity {
	items := []ecs.Entity{}

	world.Manager.Join(
		world.Components.Item,
		world.Components.Weapon,
		world.Components.ItemLocationInBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		items = append(items, entity)
	}))

	return worldhelper.SortEntities(world, items)
}

func (st *InventoryMenuState) queryMenuWearable(world w.World) []ecs.Entity {
	items := []ecs.Entity{}

	world.Manager.Join(
		world.Components.Item,
		world.Components.Wearable,
		world.Components.ItemLocationInBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		items = append(items, entity)
	}))

	return worldhelper.SortEntities(world, items)
}

// createTabDisplayUI はタブ表示UIを作成する
func (st *InventoryMenuState) createTabDisplayUI(_ w.World) {
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
}

// createCategoryDisplayUI はカテゴリ表示UIを作成する
func (st *InventoryMenuState) createCategoryDisplayUI(world w.World) {
	st.updateCategoryDisplay(world)
}

// updateCategoryDisplay はカテゴリ表示を更新する
func (st *InventoryMenuState) updateCategoryDisplay(world w.World) {
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
func (st *InventoryMenuState) updateInitialItemDisplay(world w.World) error {
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
