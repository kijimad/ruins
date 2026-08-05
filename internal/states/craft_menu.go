package states

import (
	"fmt"
	"slices"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
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
	detail       menuscreen.Detail // レシピの性能・材料・説明を出す詳細モーダル。overlay として Screen に登録する
	result       menuscreen.Detail // 合成結果の詳細モーダル。overlay として Screen に登録する
	resultEntity ecs.Entity        // 直近で合成したアイテム
	screen       Screen[craftProps]
}

// State interface ================

var _ es.State[w.World] = &CraftMenuState{}
var _ Configurable = &CraftMenuState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *CraftMenuState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

var _ es.ActionHandler[w.World] = &CraftMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *CraftMenuState) OnStart(_ w.World) error {
	st.detail = menuscreen.NewDetail(st.detailContent)
	st.result = menuscreen.NewDetail(st.resultDetailContent)
	// result を先に登録する。合成結果が開いている間はそちらが入力を専有する
	st.screen = NewScreen[craftProps](st, &st.result, &st.detail)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *CraftMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *CraftMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *CraftMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *CraftMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.screen.Open(st.detail.Open)
	case inputmapper.ActionMenuSelect:
		if err := st.craftSelected(world); err != nil {
			return es.Transition[w.World]{}, err
		}
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

func (st *CraftMenuState) fetch(world w.World) craftProps {
	return craftProps{
		Tabs: st.createTabs(world),
	}
}

func (st *CraftMenuState) menu(props craftProps) MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return MenuConfig{Key: "craft", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuItemsPerPage}
}

func (st *CraftMenuState) createTabs(world w.World) []craftTabData {
	return []craftTabData{
		{ID: "consumables", Label: "道具", Items: st.createMenuItems(world, st.queryMenuConsumable(world))},
		{ID: "weapons", Label: "武器", Items: st.createMenuItems(world, st.queryMenuWeapon(world))},
		{ID: "wearables", Label: "防具", Items: st.createMenuItems(world, st.queryMenuWearable(world))},
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
// 合成
// ================

// craftSelected は現在カーソルが当たっているレシピを合成し、結果モーダルを開く。
// 合成不可のレシピは何もしない。決定で即実行し、途中のアクション選択は挟まない
func (st *CraftMenuState) craftSelected(world w.World) error {
	item, ok := st.selectedRecipe()
	if !ok || !item.CanCraft {
		return nil
	}
	resultEntity, err := gameaction.Craft(world, item.RecipeName)
	if err != nil {
		return fmt.Errorf("合成に失敗: %w", err)
	}
	st.resultEntity = resultEntity
	st.screen.Open(st.result.Open)
	return nil
}

// selectedRecipe は現在カーソルが当たっているレシピを返す
func (st *CraftMenuState) selectedRecipe() (craftItemData, bool) {
	props := st.screen.Props()
	sel := st.screen.Selection()
	if sel.TabIndex >= len(props.Tabs) {
		return craftItemData{}, false
	}
	items := props.Tabs[sel.TabIndex].Items
	if sel.ItemIndex >= len(items) {
		return craftItemData{}, false
	}
	return items[sel.ItemIndex], true
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

func (st *CraftMenuState) view(_ w.World, props craftProps, sel Selection, res resources.UIResources) *ebitenui.UI {
	// カテゴリはタブ帯に寄せ、本体は名前のみの1カラム一覧にする。性能・材料・説明は x の詳細モーダルで見る
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	return newTabScreenUI(res, tabScreen{
		TabLabels: labels,
		TabIndex:  sel.TabIndex,
		Content:   st.buildItemContainer(props.Tabs, sel.TabIndex, sel.ItemIndex, res),
		Footer:    menuNavHint(true, "x 詳細"),
	})
}

func (st *CraftMenuState) buildItemContainer(tabs []craftTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	if tabIndex >= len(tabs) {
		return styled.NewVerticalContainer()
	}

	currentTab := tabs[tabIndex]
	// 先頭に印の列を置き、合成できるレシピにはチェック、できないレシピにはバツを付ける。名前の開始位置は揃える
	columnWidths := []int{20, 320}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft}
	rows := make([]menuRow, len(currentTab.Items))
	for i, it := range currentTab.Items {
		mark := consts.IconClose
		if it.CanCraft {
			mark = consts.IconCheck
		}
		rows[i] = menuRow{Cells: []string{mark, it.RecipeName}}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: "(レシピなし)"}, res)
}

// detailContent は現在カーソルが当たっているレシピの性能・材料・説明を返す。詳細モーダルの唯一の定義点
func (st *CraftMenuState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	item, ok := st.selectedRecipe()
	if !ok || item.RecipeName == "" {
		return menuscreen.DetailContent{}, false
	}
	spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, item.RecipeName)
	if err != nil {
		return menuscreen.DetailContent{}, false
	}

	// 必要材料を先頭に置き、所持数が足りていれば成功色、足りなければ警告色で示す。
	// その後ろに生成物の性能行を続ける
	var rows []menuscreen.SpecRow
	if spec.Recipe != nil {
		rows = append(rows, menuscreen.SpecRow{Label: "材料", Header: true})
		for _, in := range spec.Recipe.Inputs {
			owned := 0
			if entity, found := query.FindStackableInInventory(world, in.Name); found {
				owned = query.GetEntityCount(world, entity)
			}
			rowColor := theme.StatusDanger
			if owned >= in.Amount {
				rowColor = theme.StatusSuccess
			}
			rows = append(rows, menuscreen.SpecRow{Label: in.Name, Value: fmt.Sprintf("%d / %d", in.Amount, owned), Color: &rowColor})
		}
	}
	rows = append(rows, views.SpecRowsFromSpec(world, spec)...)

	desc := ""
	if spec.Description != nil {
		desc = spec.Description.Description
	}
	return menuscreen.DetailContent{Name: item.RecipeName, Desc: desc, Rows: rows}, true
}
