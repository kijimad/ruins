package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// tabID は収納メニューのタブ識別子
type tabID string

// 収納メニューのタブID。タブ定義・転送処理で参照する。定義型にして任意文字列の混入を防ぐ
const (
	tabIDRetrieve tabID = "retrieve"
	tabIDStore    tabID = "store"
)

// StorageMenuState は収納メニューのゲームステート
type StorageMenuState struct {
	es.BaseState[w.World]
	storageEntity ecs.Entity
	showDetail    bool // x の詳細モーダルを表示中か
	detailPage    int  // 詳細モーダルの表示ページ
	rebuild       bool // 次フレームで UI を作り直すか
	menuMount     *hooks.Mount[storageProps]
	widget        *ebitenui.UI
}

// State interface ================

var _ es.State[w.World] = &StorageMenuState{}
var _ Configurable = &StorageMenuState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *StorageMenuState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

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
		ItemsPerPage: menuItemsPerPage,
	})

	menuDirty := st.menuMount.Update()
	if menuDirty || st.widget == nil || st.rebuild {
		st.widget = st.buildUI(world)
		st.rebuild = false
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
	ki := input.GetSharedKeyboardInput()
	if st.showDetail {
		if ki.IsKeyJustPressed(ebiten.KeyEscape) || ki.IsKeyJustPressed(ebiten.KeyX) || ki.IsEnterJustPressedOnce() {
			return inputmapper.ActionMenuCancel, true
		}
		if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowLeft) {
			return inputmapper.ActionMenuLeft, true
		}
		if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowRight) {
			return inputmapper.ActionMenuRight, true
		}
		return "", false
	}
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *StorageMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	if st.showDetail {
		switch action {
		case inputmapper.ActionMenuCancel:
			st.showDetail = false
			st.rebuild = true
		case inputmapper.ActionMenuLeft:
			if st.detailPage > 0 {
				st.detailPage--
				st.rebuild = true
			}
		case inputmapper.ActionMenuRight:
			if e, ok := st.selectedEntity(); ok && st.detailPage < menuscreen.DetailPageCount(world, e)-1 {
				st.detailPage++
				st.rebuild = true
			}
		default:
			// 詳細表示中は他のアクションを無視する
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.showDetail = true
		st.detailPage = 0
		st.rebuild = true
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
	ID    tabID
	Label string
	Items []storageItemData
}

type storageItemData struct {
	Entity ecs.Entity
	Name   string
	Count  string
}

func (st *StorageMenuState) fetchProps(world w.World) storageProps {
	storageName := query.GetEntityName(st.storageEntity, world)
	wc := world.Components.WeightCapacity.Get(st.storageEntity)
	weightText := fmt.Sprintf("%s / %s", wc.Current.KgString(), wc.Max.KgString())

	storeTabs := st.createBackpackItemData(world)

	// 「収納」タブで選択中のアイテムが重量超過かどうか判定する
	weightOverflow := false
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "storage")
	if ok && menuState.TabIndex == 1 && len(storeTabs) > 0 && menuState.ItemIndex < len(storeTabs) {
		selectedItem := storeTabs[menuState.ItemIndex]
		itemWeight := query.GetEntityWeight(world, selectedItem.Entity)
		weightOverflow = wc.Current+itemWeight > wc.Max
	}

	return storageProps{
		Tabs: []storageTabData{
			{ID: tabIDRetrieve, Label: "取得", Items: st.createStorageItemData(world)},
			{ID: tabIDStore, Label: "収納", Items: storeTabs},
		},
		StorageName:    storageName,
		WeightText:     weightText,
		WeightOverflow: weightOverflow,
	}
}

func (st *StorageMenuState) createStorageItemData(world w.World) []storageItemData {
	items := query.GetStorageItems(world, st.storageEntity)
	sorted := query.SortEntities(world, items)
	return st.toStorageItemData(world, sorted)
}

func (st *StorageMenuState) createBackpackItemData(world w.World) []storageItemData {
	var entities []ecs.Entity
	backpackQuery := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for backpackQuery.Next() {
		entity := backpackQuery.Entity()
		entities = append(entities, entity)
	}

	sorted := query.SortEntities(world, entities)
	return st.toStorageItemData(world, sorted)
}

func (st *StorageMenuState) toStorageItemData(world w.World, entities []ecs.Entity) []storageItemData {
	items := make([]storageItemData, len(entities))
	for i, entity := range entities {
		name := query.GetEntityName(entity, world)
		item := storageItemData{
			Entity: entity,
			Name:   name,
		}
		if world.Components.Stackable.Has(entity) {
			stackable := world.Components.Stackable.Get(entity)
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
	case tabIDRetrieve:
		// 収納からバックパックへ移動
		playerEntity, err := query.GetPlayerEntity(world)
		if err != nil {
			return err
		}
		if err := lifecycle.MoveToBackpack(world, item.Entity, playerEntity); err != nil {
			return err
		}
	case tabIDStore:
		// バックパックから収納へ移動
		if !query.CanAddToStorage(world, st.storageEntity, item.Entity) {
			return nil // 重量超過の場合は何もしない
		}
		if err := lifecycle.MoveToStorage(world, item.Entity, st.storageEntity); err != nil {
			return err
		}
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

	// カテゴリをタブ帯に寄せ、本体は1カラムの一覧にする。性能は x の詳細モーダルで見る
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}

	// 重量は見出しに出す。収納しきれない選択中は警告アイコンを添える
	header := fmt.Sprintf("重量 %s", props.WeightText)
	if props.WeightOverflow {
		header += " " + consts.IconWarning
	}

	eui := newTabScreenUI(res, tabScreen{
		Header:    header,
		TabLabels: labels,
		TabIndex:  tabIndex,
		Content:   st.buildActiveListContainer(props, tabIndex, itemIndex, res),
		Footer:    menuNavHint(true, "x 詳細"),
	})

	if st.showDetail {
		if e, ok := st.selectedEntity(); ok {
			name := query.GetEntityName(e, world)
			desc := ""
			if world.Components.Description.Has(e) {
				desc = world.Components.Description.Get(e).Description
			}
			eui.AddWindow(menuscreen.BuildDetailWindow(world, getCenterWinRect(world), name, desc, e, st.detailPage))
		}
	}

	return eui
}

// selectedEntity は現在カーソルが当たっているアイテムのエンティティを返す
func (st *StorageMenuState) selectedEntity() (ecs.Entity, bool) {
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "storage")
	if menuState.TabIndex >= len(props.Tabs) {
		return ecs.Entity{}, false
	}
	items := props.Tabs[menuState.TabIndex].Items
	if menuState.ItemIndex >= len(items) {
		return ecs.Entity{}, false
	}
	return items[menuState.ItemIndex].Entity, true
}

func (st *StorageMenuState) buildActiveListContainer(props storageProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(props.Tabs) {
		return container
	}

	currentTab := props.Tabs[tabIndex]
	columnWidths := []int{20, 150, 50}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight}

	pg := pagination.New(itemIndex, len(currentTab.Items), menuItemsPerPage)
	container.AddChild(newPageIndicator(pg, res))

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		styled.NewTableRow(table, columnWidths, []string{"", entry.Item.Name, entry.Item.Count}, aligns, new(pg.IsSelectedInPage(entry.Index)), res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(アイテムなし)", res))
	}

	return container
}
