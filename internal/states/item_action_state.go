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
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/styled"
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

// verbList は表示順に並べた動詞タブの一覧。タブ順を兼ねる。内容は定数なのでパッケージ変数で1度だけ構築する。
// 投げるは Throwable と ThrowActivity の実装後に足す。
var verbList = []itemVerb{
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
		Exec:    execPlace,
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

// execPlace は選択アイテムをプレイヤーの足元に置いてダンジョンへ戻る。置く位置は指定しない
func execPlace(world w.World, entity ecs.Entity) (es.Transition[w.World], error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	dest := gc.GridElement{Coord: world.Components.GridElement.Get(player).Coord}
	if _, err := activity.Execute(&activity.DropBehavior{Target: entity, Destination: dest}, player, world); err != nil {
		return es.Transition[w.World]{}, err
	}
	return es.Transition[w.World]{Type: es.TransPop}, nil
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
	for i, v := range verbList {
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
	initialVerb verbID            // 開いた直後に表示する動詞タブ
	tabSeeded   bool              // 初期タブへ寄せたか
	detail      menuscreen.Detail // 詳細モーダルの表示状態とページ送り
	rebuild     bool              // 次フレームで UI を作り直すか
	mount       *hooks.Mount[itemActionProps]
	widget      *ebitenui.UI
}

var _ es.State[w.World] = &ItemActionState{}
var _ Configurable = &ItemActionState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *ItemActionState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

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
	st.detail = menuscreen.NewDetail(st.detailContent)
	return nil
}

// Update はステートの更新処理
func (st *ItemActionState) Update(world w.World) (es.Transition[w.World], error) {
	if st.detail.Active() {
		// 詳細表示中はページ送りと閉じるだけを扱い、通常のメニュー入力は止める
		if st.detail.HandleInput(world) {
			st.rebuild = true
		}
	} else if action, ok := st.handleInput(); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.mount.Dispatch(action)
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

// handleInput はキー入力を Action に変換する。詳細モーダルの入力は Update 側で detail が扱う
func (st *ItemActionState) handleInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	// 動詞ショートカットは開いている間もタブ移動に使える。調べる X は Shift+x、詳細 x は Shift 無し
	if ki.IsKeyJustPressed(ebiten.KeyX) {
		if ki.IsKeyPressed(ebiten.KeyShift) {
			return inputmapper.ActionVerbExamine, true
		}
		return inputmapper.ActionOpenItemDetail, true
	}
	if ki.IsKeyJustPressed(ebiten.KeyD) {
		return inputmapper.ActionVerbPlace, true
	}
	if ki.IsKeyJustPressed(ebiten.KeyE) {
		return inputmapper.ActionVerbConsume, true
	}
	if ki.IsKeyJustPressed(ebiten.KeyR) {
		return inputmapper.ActionVerbRead, true
	}
	if ki.IsKeyJustPressed(ebiten.KeyT) {
		return inputmapper.ActionVerbUse, true
	}
	return HandleMenuInput()
}

// DoAction は Action を実行する
func (st *ItemActionState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open()
		st.rebuild = true
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionVerbExamine, inputmapper.ActionVerbPlace, inputmapper.ActionVerbConsume, inputmapper.ActionVerbRead, inputmapper.ActionVerbUse:
		// 開いている間の動詞キーは対応タブへジャンプする
		if v, ok := verbByAction(action); ok {
			st.jumpToTab(v)
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		return st.executeSelected(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// jumpToTab は指定した動詞のタブへ移動する。現在タブから差分だけ TabNext を送る
func (st *ItemActionState) jumpToTab(target verbID) {
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, itemActionMenuKey)
	n := len(verbList)
	if n == 0 {
		return
	}
	steps := ((verbTabIndex(target)-menuState.TabIndex)%n + n) % n
	for range steps {
		st.mount.Dispatch(inputmapper.ActionMenuTabNext)
	}
	st.rebuild = true
}

// executeSelected は選択中アイテムへ現在の動詞を適用する。Exec を持たない調べるは詳細モーダルを開く
func (st *ItemActionState) executeSelected(world w.World) (es.Transition[w.World], error) {
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, itemActionMenuKey)
	vs := verbList
	if menuState.TabIndex >= len(vs) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	verb := vs[menuState.TabIndex]
	if verb.Exec == nil {
		st.detail.Open()
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
	Weight string
	Count  int
	Desc   string
}

func (st *ItemActionState) fetchProps(world w.World) itemActionProps {
	player, err := query.GetPlayerEntity(world)
	var backpack []ecs.Entity
	if err == nil {
		backpack = playerBackpackItems(world, player)
	}

	vs := verbList
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
		Weight: query.GetEntityWeight(world, entity).KgString(),
	}
	if world.Components.Stackable.Has(entity) {
		entry.Count = world.Components.Stackable.Get(entity).Count
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

	// タブ見出しに直達ショートカットを添える。調べる(X) 置く(d) の形
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		if tab.Key != "" {
			labels[i] = fmt.Sprintf("%s(%s)", tab.Label, tab.Key)
		} else {
			labels[i] = tab.Label
		}
	}

	// タイトルは置かず、タブ帯から始める。詳細は x のモーダルで見る
	ui := newTabScreenUI(res, tabScreen{
		TabLabels: labels,
		TabIndex:  tabIndex,
		Content:   st.buildItemList(props, tabIndex, itemIndex, res),
		Footer:    menuNavHint(true, "x 詳細"),
	})

	if st.detail.Active() {
		if win := st.detail.Window(world, getCenterWinRect(world)); win != nil {
			ui.AddWindow(win)
		}
	}

	return ui
}

// buildItemList は現在タブのアイテムを、他メニューと同じテーブル描画で縦1列に並べる。
// 左に名前、右に個数。行高とフォントはテーブル共通で揃う
func (st *ItemActionState) buildItemList(props itemActionProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(props.Tabs) {
		return container
	}
	items := props.Tabs[tabIndex].Items
	// 全タブで開始位置と高さを揃えるための行。アイテムメニューはページ送りしないので空行にする
	container.AddChild(styled.NewPageIndicator(" ", res))
	if len(items) == 0 {
		container.AddChild(styled.NewDescriptionText("該当するアイテムがありません", res))
		return container
	}
	columnWidths := []int{240, 70}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	table := styled.NewTableContainer(columnWidths, res)
	for i, item := range items {
		isSelected := i == itemIndex
		styled.NewTableRow(table, columnWidths, []string{nameWithCount(item.Name, item.Count), item.Weight}, aligns, &isSelected, res)
	}
	container.AddChild(table)
	return container
}

// detailContent は現在カーソルが当たっているアイテムの詳細内容を返す。詳細モーダルの唯一の定義点
func (st *ItemActionState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	props := st.fetchProps(world)
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, itemActionMenuKey)
	if menuState.TabIndex >= len(props.Tabs) {
		return menuscreen.DetailContent{}, false
	}
	items := props.Tabs[menuState.TabIndex].Items
	if menuState.ItemIndex >= len(items) {
		return menuscreen.DetailContent{}, false
	}
	item := items[menuState.ItemIndex]
	return menuscreen.DetailContent{Name: item.Name, Desc: item.Desc, Entity: item.Entity}, true
}
