package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

const storageItemsPerPage = 20

// StorageMenuState は収納メニューのゲームステート
type StorageMenuState struct {
	es.BaseState[w.World]
	storageEntity ecs.Entity
	menuMount     *hooks.Mount[storageProps]
	widget        *ebitenui.UI
}

func (st StorageMenuState) String() string {
	return "StorageMenu"
}

// State interface ================

var _ es.State[w.World] = &StorageMenuState{}
var _ es.ActionHandler[w.World] = &StorageMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *StorageMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *StorageMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *StorageMenuState) OnStart(_ w.World) error {
	st.menuMount = hooks.NewMount[storageProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *StorageMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *StorageMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// WeightDirtySystemを実行して所持重量を更新
	for _, updater := range []w.Updater{
		&gs.WeightDirtySystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
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
	hooks.UseTabMenu(st.menuMount.Store(), "storage", hooks.TabMenuConfig{
		TabCount:     len(props.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: storageItemsPerPage,
	})

	menuDirty := st.menuMount.Update()
	if menuDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *StorageMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *StorageMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *StorageMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		if err := st.executeTransfer(world); err != nil {
			return es.Transition[w.World]{}, err
		}
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("storageMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type storageProps struct {
	Tabs           []storageTabData
	StorageName    string
	WeightText     string
	WeightOverflow bool
}

type storageTabData struct {
	ID    string
	Label string
	Items []storageItemData
}

type storageItemData struct {
	Entity ecs.Entity
	Name   string
	Count  string
}

func (st *StorageMenuState) fetchProps(world w.World) storageProps {
	storageName := worldhelper.GetEntityName(st.storageEntity, world)
	wc := world.Components.WeightCapacity.Get(st.storageEntity).(*gc.WeightCapacity)
	weightText := fmt.Sprintf("%.1f / %.1f kg", wc.Current, wc.Max)

	storeTabs := st.createBackpackItemData(world)

	// 「収納」タブで選択中のアイテムが重量超過かどうか判定する
	weightOverflow := false
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "storage")
	if ok && menuState.TabIndex == 1 && len(storeTabs) > 0 && menuState.ItemIndex < len(storeTabs) {
		selectedItem := storeTabs[menuState.ItemIndex]
		itemWeight := worldhelper.GetEntityWeight(world, selectedItem.Entity)
		weightOverflow = wc.Current+itemWeight > wc.Max
	}

	return storageProps{
		Tabs: []storageTabData{
			{ID: "retrieve", Label: "取得", Items: st.createStorageItemData(world)},
			{ID: "store", Label: "収納", Items: storeTabs},
		},
		StorageName:    storageName,
		WeightText:     weightText,
		WeightOverflow: weightOverflow,
	}
}

func (st *StorageMenuState) createStorageItemData(world w.World) []storageItemData {
	items := worldhelper.GetStorageItems(world, st.storageEntity)
	sorted := worldhelper.SortEntities(world, items)
	return st.toStorageItemData(world, sorted)
}

func (st *StorageMenuState) createBackpackItemData(world w.World) []storageItemData {
	var entities []ecs.Entity
	world.Manager.Join(
		world.Components.LocationInBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entities = append(entities, entity)
	}))

	sorted := worldhelper.SortEntities(world, entities)
	return st.toStorageItemData(world, sorted)
}

func (st *StorageMenuState) toStorageItemData(world w.World, entities []ecs.Entity) []storageItemData {
	items := make([]storageItemData, len(entities))
	for i, entity := range entities {
		name := worldhelper.GetEntityName(entity, world)
		item := storageItemData{
			Entity: entity,
			Name:   name,
		}
		if entity.HasComponent(world.Components.Stackable) {
			stackable := world.Components.Stackable.Get(entity).(*gc.Stackable)
			item.Count = fmt.Sprintf("%d", stackable.Count)
		}
		items[i] = item
	}
	return items
}

// ================
// アクション実行
// ================

func (st *StorageMenuState) executeTransfer(world w.World) error {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "storage")
	if !ok {
		return fmt.Errorf("storageの取得に失敗")
	}
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	if tabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[tabIndex]
	if len(tab.Items) == 0 || itemIndex >= len(tab.Items) {
		return nil
	}

	item := tab.Items[itemIndex]

	switch tab.ID {
	case "retrieve":
		// 収納からバックパックへ移動
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			return err
		}
		worldhelper.MoveToBackpack(world, item.Entity, playerEntity)
	case "store":
		// バックパックから収納へ移動
		if !worldhelper.CanAddToStorage(world, st.storageEntity, item.Entity) {
			return nil // 重量超過の場合は何もしない
		}
		worldhelper.MoveToStorage(world, item.Entity, st.storageEntity)
	}

	return nil
}

// ================
// buildUI
// ================

func (st *StorageMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "storage")
	if !ok {
		return &ebitenui.UI{Container: widget.NewContainer()}
	}
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 1行目: タイトル、カテゴリ、重量
	root.AddChild(styled.NewTitleText(props.StorageName, res))
	root.AddChild(st.buildCategoryContainer(props.Tabs, tabIndex, res))
	root.AddChild(st.buildWeightContainer(props.WeightText, props.WeightOverflow, res))

	// 2行目: 操作対象リスト（左）、空、参照リスト（右）
	root.AddChild(st.buildActiveListContainer(props, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(st.buildReferenceListContainer(props, tabIndex, res))

	// 3行目: ヘルプテキスト
	root.AddChild(st.buildHelpContainer(props.Tabs, tabIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	return &ebitenui.UI{Container: root}
}

func (st *StorageMenuState) buildCategoryContainer(tabs []storageTabData, tabIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	for i, tab := range tabs {
		isSelected := i == tabIndex
		color := theme.TextSecondary
		if isSelected {
			color = theme.TextPrimary
		}
		container.AddChild(styled.NewListItemText(tab.Label, color, isSelected, res))
	}
	return container
}

func (st *StorageMenuState) buildWeightContainer(weightText string, overflow bool, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	textColor := theme.TextPrimary
	if overflow {
		textColor = theme.HUDWeightDanger
	}
	container.AddChild(widget.NewText(
		widget.TextOpts.Text(weightText, &res.Text.BodyFace, textColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
		),
	))
	return container
}

func (st *StorageMenuState) buildActiveListContainer(props storageProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(props.Tabs) {
		return container
	}

	currentTab := props.Tabs[tabIndex]
	columnWidths := []int{20, 150, 50}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight}

	pg := pagination.New(itemIndex, len(currentTab.Items), storageItemsPerPage)

	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " "
	}
	container.AddChild(styled.NewPageIndicator(pageText, res))

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, columnWidths, []string{"", entry.Item.Name, entry.Item.Count}, aligns, &isSelected, res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(アイテムなし)", res))
	}

	return container
}

func (st *StorageMenuState) buildReferenceListContainer(props storageProps, tabIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 操作対象の反対側のタブのアイテムを表示する
	refTabIndex := 1 - tabIndex
	if refTabIndex < 0 || refTabIndex >= len(props.Tabs) {
		return container
	}

	refTab := props.Tabs[refTabIndex]
	columnWidths := []int{150, 50}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}

	container.AddChild(styled.NewPageIndicator(" ", res))

	// 参照リストはスクロール非対応のため、先頭からstorageItemsPerPage件のみ表示する
	table := styled.NewTableContainer(columnWidths, res)
	for i, item := range refTab.Items {
		if i >= storageItemsPerPage {
			break
		}
		styled.NewTableRow(table, columnWidths, []string{item.Name, item.Count}, aligns, nil, res)
	}
	container.AddChild(table)

	if len(refTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(アイテムなし)", res))
	}

	return container
}

func (st *StorageMenuState) buildHelpContainer(tabs []storageTabData, tabIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	helpText := "Enter:取り出す  ←→:タブ切替  Esc:閉じる"
	if tabIndex < len(tabs) && tabs[tabIndex].ID == "store" {
		helpText = "Enter:収納する  ←→:タブ切替  Esc:閉じる"
	}
	container.AddChild(styled.NewMenuText(helpText, res))
	return container
}
