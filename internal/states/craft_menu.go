package states

import (
	"fmt"
	"slices"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/screenui"
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
	screen       *menurt.Screen[CraftProps]
}

// State interface ================

var _ es.State[w.World] = &CraftMenuState{}
var _ menurt.ExtraInput = &CraftMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *CraftMenuState) OnStart(_ w.World) error {
	st.detail = menuscreen.NewDetail(st.detailContent)
	st.result = menuscreen.NewEntityDetail(func() (ecs.Entity, bool) { return st.resultEntity, true })
	// result を先に登録する。合成結果が開いている間はそちらが入力を専有する
	st.screen = menurt.NewScreen[CraftProps](st, &st.result, &st.detail)
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

// ExtraInput は共通入力に加える独自キーを返す。x で選択中の詳細モーダルを開く
func (st *CraftMenuState) ExtraInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return "", false
}

// DoAction はActionを実行する
func (st *CraftMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
	case inputmapper.ActionMenuSelect:
		if err := st.craftSelected(world); err != nil {
			return es.Transition[w.World]{}, err
		}
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("craftMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// CraftProps は画面の表示 props。menurt.Screen の型引数として渡す
type CraftProps struct {
	Tabs []craftTabData
}

type craftTabData struct {
	ID    string
	Label string
	Items []craftItemData
}

type craftItemData struct {
	RecipeID   string // 合成の同定キー。NewRecipeSpec/CanCraft/Craft はこれで引く
	RecipeName string // 表示名
	CanCraft   bool
}

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *CraftMenuState) Fetch(world w.World) CraftProps {
	return CraftProps{
		Tabs: st.createTabs(world),
	}
}

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *CraftMenuState) Menu(props CraftProps) menurt.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menurt.MenuConfig{Key: "craft", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuItemsPerPage}
}

func (st *CraftMenuState) createTabs(world w.World) []craftTabData {
	return []craftTabData{
		{ID: "consumables", Label: query.T(world, "Consumables"), Items: st.createMenuItems(world, st.queryMenuConsumable(world))},
		{ID: "weapons", Label: query.T(world, "Weapons"), Items: st.createMenuItems(world, st.queryMenuWeapon(world))},
		{ID: "wearables", Label: query.T(world, "Armor"), Items: st.createMenuItems(world, st.queryMenuWearable(world))},
	}
}

func (st *CraftMenuState) createMenuItems(world w.World, recipeIDs []string) []craftItemData {
	items := make([]craftItemData, len(recipeIDs))

	for i, recipeID := range recipeIDs {
		canCraft, _ := gameaction.CanCraft(world, recipeID)
		// レシピ id は生成アイテム id と一致する。表示名はアイテムの英語名を翻訳して出し、材料表示と経路を揃える
		items[i] = craftItemData{
			RecipeID:   recipeID,
			RecipeName: query.T(world, raw.ItemName(world.Resources.RawMaster, recipeID)),
			CanCraft:   canCraft,
		}
	}

	return items
}

func (st *CraftMenuState) queryMenuConsumable(world w.World) []string {
	var items []string

	for _, recipe := range raw.PtrSlice(world.Resources.RawMaster.Recipes) {
		spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, recipe.Id)
		if err != nil {
			continue
		}
		if spec.Consumable != nil {
			items = append(items, recipe.Id)
		}
	}

	slices.Sort(items)
	return items
}

func (st *CraftMenuState) queryMenuWeapon(world w.World) []string {
	var items []string

	for _, recipe := range raw.PtrSlice(world.Resources.RawMaster.Recipes) {
		spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, recipe.Id)
		if err != nil {
			continue
		}
		// TODO: カテゴリ定義で判定したい
		if spec.Melee != nil || spec.Fire != nil {
			items = append(items, recipe.Id)
		}
	}

	slices.Sort(items)
	return items
}

func (st *CraftMenuState) queryMenuWearable(world w.World) []string {
	var items []string

	for _, recipe := range raw.PtrSlice(world.Resources.RawMaster.Recipes) {
		spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, recipe.Id)
		if err != nil {
			continue
		}
		if spec.Wearable != nil {
			items = append(items, recipe.Id)
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
	resultEntity, err := gameaction.Craft(world, item.RecipeID)
	if err != nil {
		return fmt.Errorf("failed to craft: %w", err)
	}
	st.resultEntity = resultEntity
	st.result.Open(world)
	return nil
}

// selectedRecipe は現在カーソルが当たっているレシピを返す
func (st *CraftMenuState) selectedRecipe() (craftItemData, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return craftItemData{}, false
	}
	items := props.Tabs[cursor.TabIndex].Items
	if cursor.ItemIndex >= len(items) {
		return craftItemData{}, false
	}
	return items[cursor.ItemIndex], true
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *CraftMenuState) View(world w.World, props CraftProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
	// カテゴリはタブ帯に寄せ、本体は名前のみの1カラム一覧にする。性能・材料・説明は x の詳細モーダルで見る
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	return screenui.NewTabScreen(res, screenui.TabScreen{
		TabLabels: labels,
		TabIndex:  cursor.TabIndex,
		Content:   st.buildItemContainer(world, props.Tabs, cursor.TabIndex, cursor.ItemIndex, res),
		Footer:    menuNavHint(world, true, query.T(world, "x Details")),
	})
}

func (st *CraftMenuState) buildItemContainer(world w.World, tabs []craftTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
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
		rows[i] = menuRow{Cells: styled.TextCells(mark, it.RecipeName)}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No recipes")}, res)
}

// detailContent は現在カーソルが当たっているレシピの性能・材料・説明を返す。詳細モーダルの唯一の定義点
func (st *CraftMenuState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	item, ok := st.selectedRecipe()
	if !ok || item.RecipeID == "" {
		return menuscreen.DetailContent{}, false
	}
	spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, item.RecipeID)
	if err != nil {
		return menuscreen.DetailContent{}, false
	}

	// 必要材料を先頭に置き、所持数が足りていれば成功色、足りなければ警告色で示す。
	// その後ろに生成物の性能行を続ける
	var rows []views.SpecRow
	if spec.Recipe != nil {
		rows = append(rows, views.SpecRow{Label: query.T(world, "Materials"), Header: true})
		for _, in := range spec.Recipe.Inputs {
			owned := 0
			if entity, found := query.FindStackableInInventory(world, in.ID); found {
				owned = query.GetEntityCount(world, entity)
			}
			rowColor := theme.StatusDanger
			if owned >= in.Amount {
				rowColor = theme.StatusSuccess
			}
			label := query.T(world, raw.ItemName(world.Resources.RawMaster, in.ID))
			rows = append(rows, views.SpecRow{Label: label, Value: fmt.Sprintf("%d / %d", in.Amount, owned), Color: &rowColor})
		}
	}
	rows = append(rows, views.SpecRowsFromSpec(world, spec)...)

	desc := ""
	if spec.Description != nil {
		desc = query.T(world, spec.Description.Description)
	}
	return menuscreen.DetailContent{Name: item.RecipeName, Desc: desc, Rows: rows}, true
}
