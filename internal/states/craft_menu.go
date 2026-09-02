package states

import (
	"fmt"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// CraftMenuState はクラフトメニューのゲームステート
type CraftMenuState struct {
	es.BaseState[w.World]
	detail       overlay.Detail // レシピの性能・材料・説明を出す詳細モーダル。overlay として Screen に登録する
	result       overlay.Detail // クラフト結果の詳細モーダル。overlay として Screen に登録する
	resultEntity ecs.Entity     // 直近でクラフトしたアイテム
	screen       *menuloop.Screen[CraftProps]
}

// State interface ================

var _ es.State[w.World] = &CraftMenuState{}
var _ menuloop.KeyBindings = &CraftMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *CraftMenuState) OnStart(_ w.World) error {
	st.detail = overlay.NewDetail(st.detailContent)
	st.result = overlay.NewEntityDetail(func() (ecs.Entity, bool) { return st.resultEntity, true })
	// result を先に登録する。クラフト結果が開いている間はそちらが入力を専有する
	st.screen = menuloop.NewScreen[CraftProps](st, &st.result, &st.detail)
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

// KeyBindings は x の詳細表示を共通入力に足す
func (st *CraftMenuState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
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
	default:
		return es.Transition[w.World]{}, fmt.Errorf("craftMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// CraftProps は画面の表示 props。menuloop.Screen の型引数として渡す
type CraftProps struct {
	Tabs []craftTabData
}

type craftTabData struct {
	ID    string
	Label string
	Items []craftItemData
}

type craftItemData struct {
	RecipeID   string // クラフトの同定キー。NewRecipeSpec/CanCraft/Craft はこれで引く
	RecipeName string // 表示名
	CanCraft   bool
}

// Fetch は世界から表示 props を構築する。menuloop.Model の Model 部にあたる
func (st *CraftMenuState) Fetch(world w.World) (CraftProps, error) {
	return CraftProps{
		Tabs: []craftTabData{
			{ID: "consumables", Label: query.T(world, "Consumables"), Items: st.createMenuItems(world, st.queryMenuConsumable(world))},
			{ID: "weapons", Label: query.T(world, "Weapons"), Items: st.createMenuItems(world, st.queryMenuWeapon(world))},
			{ID: "wearables", Label: query.T(world, "Armor"), Items: st.createMenuItems(world, st.queryMenuWearable(world))},
		},
	}, nil
}

// Menu は一覧の構成を返す。menuloop.Model の Menu 部にあたる
func (st *CraftMenuState) Menu(props CraftProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menuloop.MenuConfig{Key: "craft", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuloop.ItemsPerPageAuto}
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
// クラフト
// ================

// craftSelected は現在カーソルが当たっているレシピをクラフトし、結果モーダルを開く。
// クラフト不可のレシピは何もしない。決定で即実行し、途中のアクション選択は挟まない
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

// ViewUI はカテゴリタブとクラフト可否印つきレシピ一覧を組む。
func (st *CraftMenuState) ViewUI(world w.World, props CraftProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	content, pager := st.buildItemListUI(world, props.Tabs, cursor.TabIndex, cursor.ItemIndex, cursor.PageSize, res)
	return menuframe.TabScreen(world, res, "", labels, cursor.TabIndex, content, keybind.HelpHint(world), pager)
}

// buildItemListUI は行列とフッタ右端のページ表示を返す。
func (st *CraftMenuState) buildItemListUI(world w.World, tabs []craftTabData, tabIndex, itemIndex, perPage int, res resources.UIResources) ([]uicore.Drawable, string) {
	if tabIndex >= len(tabs) {
		return nil, ""
	}
	currentTab := tabs[tabIndex]
	cols := styled.Cols(styled.Fit(), styled.Name())
	rows := make([]menuframe.Row, len(currentTab.Items))
	for i, it := range currentTab.Items {
		mark := consts.IconClose
		if it.CanCraft {
			mark = consts.IconCheck
		}
		rows[i] = menuframe.Row{Cells: styled.TextCells(mark, it.RecipeName)}
	}
	return menuframe.RenderList(itemIndex, rows, cols, menuframe.ListOpts{EmptyText: query.T(world, "No recipes"), ItemsPerPage: perPage}, res)
}

// detailContent は現在カーソルが当たっているレシピの性能・材料・説明を返す。詳細モーダルの唯一の定義点
func (st *CraftMenuState) detailContent(world w.World) (overlay.DetailContent, bool) {
	item, ok := st.selectedRecipe()
	if !ok || item.RecipeID == "" {
		return overlay.DetailContent{}, false
	}
	spec, err := raw.NewRecipeSpec(world.Resources.RawMaster, item.RecipeID)
	if err != nil {
		return overlay.DetailContent{}, false
	}

	// 必要材料を先頭に置き、所持数が足りていれば成功色、足りなければ警告色で示す。
	// その後ろに生成物の性能行を続ける
	var rows []entityspec.SpecRow
	if spec.Recipe != nil {
		rows = append(rows, entityspec.SpecRow{Label: query.T(world, "Materials"), Header: true})
		for _, in := range spec.Recipe.Inputs {
			owned := 0
			if entity, found := query.FindStackInInventory(world, in.ID); found {
				owned = query.GetEntityCount(world, entity)
			}
			rowColor := theme.StatusDanger
			if owned >= in.Amount {
				rowColor = theme.StatusSuccess
			}
			label := query.T(world, raw.ItemName(world.Resources.RawMaster, in.ID))
			rows = append(rows, entityspec.SpecRow{Label: label, Value: fmt.Sprintf("%d / %d", in.Amount, owned), Color: &rowColor})
		}
	}
	rows = append(rows, entityspec.SpecRowsFromSpec(world, spec)...)

	desc := ""
	if spec.Description != nil {
		desc = query.T(world, spec.Description.Description)
	}
	return overlay.DetailContent{Name: item.RecipeName, Desc: desc, Rows: rows}, true
}
