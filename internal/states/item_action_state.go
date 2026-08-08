package states

import (
	"errors"
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menurt"
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
// 動詞はこの構造体を単一の真実にする。キー・アクション・表示・振る舞いを1行にまとめ、
// HandleInput のキー変換と verbByAction のアクション対応はこの一覧から導く。追加は1行で足りる。
type itemVerb struct {
	ID    verbID
	Label string
	// KeyHint はタブ見出しに添える直達ショートカットの表記。大文字は Shift 併用を表す。
	// 調べる X は Shift+x、置く d は KeyD をそのまま押す
	KeyHint string
	// Key と Shift は直達ショートカットのキー。Shift が真なら Shift 併用を要する。
	// Action はそのショートカットが発するアクション。ダンジョン等からの直達もこのアクションで届く
	Key    ebiten.Key
	Shift  bool
	Action inputmapper.ActionID
	Accept func(world w.World, entity ecs.Entity) bool
	Exec   func(world w.World, entity ecs.Entity) (es.Transition[w.World], error)
}

// verbList は表示順に並べた動詞タブの一覧。タブ順を兼ねる。内容は定数なのでパッケージ変数で1度だけ構築する。
// 投げるは Throwable と ThrowActivity の実装後に足す。
var verbList = []itemVerb{
	{
		ID:      verbExamine,
		Label:   "Inspect",
		KeyHint: "X",
		Key:     ebiten.KeyX,
		Shift:   true,
		Action:  inputmapper.ActionVerbExamine,
		Accept:  func(_ w.World, _ ecs.Entity) bool { return true },
		Exec:    nil,
	},
	{
		ID:      verbPlace,
		Label:   "Drop",
		KeyHint: "d",
		Key:     ebiten.KeyD,
		Action:  inputmapper.ActionVerbPlace,
		Accept:  func(_ w.World, _ ecs.Entity) bool { return true },
		Exec:    execPlace,
	},
	{
		ID:      verbConsume,
		Label:   "Eat",
		KeyHint: "e",
		Key:     ebiten.KeyE,
		Action:  inputmapper.ActionVerbConsume,
		Accept:  acceptConsumeFood,
		Exec:    execUseItem,
	},
	{
		ID:      verbRead,
		Label:   "Read",
		KeyHint: "r",
		Key:     ebiten.KeyR,
		Action:  inputmapper.ActionVerbRead,
		Accept:  func(world w.World, entity ecs.Entity) bool { return world.Components.Book.Has(entity) },
		Exec:    execRead,
	},
	{
		ID:      verbUse,
		Label:   "Use",
		KeyHint: "t",
		Key:     ebiten.KeyT,
		Action:  inputmapper.ActionVerbUse,
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
	if _, err := activity.Execute(activity.NewDropActivity(entity, dest), player, world); err != nil {
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
	if _, err := activity.Execute(activity.NewUseItemActivity(entity), player, world); err != nil {
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
	act, err := activity.NewReadActivity(entity, world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	if _, err := activity.Execute(act, player, world); err != nil {
		// 検証失敗、スキル不足や周囲の敵、はユーザー入力の失敗なのでゲームを止めない。
		// 理由をログに出して閉じる。state が返すエラーは最上位まで伝播して致命化するため、
		// ここで吸収するのは検証失敗だけにする。システムエラーは握りつぶさず伝播させる
		if errors.Is(err, activity.ErrValidationFailed) {
			// スキル不足は本のスキルと必要・現在レベルから現在言語の文を組む。ほかの検証失敗は生のまま
			msg := err.Error()
			var skillErr *gc.BookSkillError
			if errors.As(err, &skillErr) {
				msg = query.T(world, "You need %s Lv%d or higher to read this. Current Lv%d",
					query.T(world, gc.SkillName(skillErr.Skill)), skillErr.Required, skillErr.Current)
			}
			gamelog.New(query.GetGameLog(world)).Markup(msg).Log()
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}
		return es.Transition[w.World]{}, err
	}
	return es.Transition[w.World]{Type: es.TransPop}, nil
}

// verbByAction はダンジョン等からの直達アクションを対応する動詞へ対応づける。verbList から導く
func verbByAction(action inputmapper.ActionID) (verbID, bool) {
	for _, v := range verbList {
		if v.Action == action {
			return v.ID, true
		}
	}
	return "", false
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
	detail      menuscreen.Detail // 詳細モーダル。overlay として Screen に登録する
	screen      *menurt.Screen[ItemActionProps]
}

var _ es.State[w.World] = &ItemActionState{}
var _ menurt.ExtraInput = &ItemActionState{}

// NewItemActionState は動詞タブ画面を initial のタブで開くファクトリを返す
func NewItemActionState(initial verbID) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &ItemActionState{initialVerb: initial}, nil
	}
}

// OnStart はステートが開始される際に呼ばれる
func (st *ItemActionState) OnStart(_ w.World) error {
	st.detail = menuscreen.NewDetail(st.detailContent)
	st.screen = menurt.NewScreen[ItemActionProps](st, &st.detail)
	return nil
}

// Update はステートの更新処理
func (st *ItemActionState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はステートの描画処理
func (st *ItemActionState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// ExtraInput は共通入力に加える動詞ショートカットを返す。verbList から導くので追加は1行で足りる。
// Shift 無しの x は動詞でなく詳細モーダルを開く
func (st *ItemActionState) ExtraInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	shift := ki.IsKeyPressed(ebiten.KeyShift)
	// Shift 無しの x は動詞でなく詳細モーダルを開く。Shift+x の調べるとキーを共有するので先に分ける
	if ki.IsKeyJustPressed(ebiten.KeyX) && !shift {
		return inputmapper.ActionOpenItemDetail, true
	}
	for _, v := range verbList {
		if ki.IsKeyJustPressed(v.Key) && (!v.Shift || shift) {
			return v.Action, true
		}
	}
	return "", false
}

// DoAction は Action を実行する
func (st *ItemActionState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		return st.executeSelected(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		// 動詞の直達キーは対応タブへジャンプする。verbList から導くので動詞追加で列挙は要らない
		if v, ok := verbByAction(action); ok {
			st.jumpToTab(v)
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		return es.Transition[w.World]{}, fmt.Errorf("unknown action: %s", action)
	}
}

func (st *ItemActionState) jumpToTab(target verbID) {
	st.screen.SetTab(verbTabIndex(target))
}

// executeSelected は選択中アイテムへ現在の動詞を適用する。Exec を持たない調べるは詳細モーダルを開く
func (st *ItemActionState) executeSelected(world w.World) (es.Transition[w.World], error) {
	cursor := st.screen.Selection()
	vs := verbList
	if cursor.TabIndex >= len(vs) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	verb := vs[cursor.TabIndex]
	if verb.Exec == nil {
		st.detail.Open(world)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	props := st.screen.Props()
	tab := props.Tabs[cursor.TabIndex]
	if cursor.ItemIndex >= len(tab.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	return verb.Exec(world, tab.Items[cursor.ItemIndex].Entity)
}

// ================
// Props
// ================

// ItemActionProps は画面の表示 props。menurt.Screen の型引数として渡す
type ItemActionProps struct {
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

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *ItemActionState) Fetch(world w.World) ItemActionProps {
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
		tabs[i] = verbTabData{ID: verb.ID, Label: query.T(world, verb.Label), Key: verb.KeyHint, Items: items}
	}
	return ItemActionProps{Tabs: tabs}
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
		Name:   query.GetEntityName(entity, world),
		Weight: query.GetEntityWeight(world, entity).KgString(),
	}
	if world.Components.Stackable.Has(entity) {
		entry.Count = world.Components.Stackable.Get(entity).Count
	}
	if world.Components.Description.Has(entity) {
		entry.Desc = query.T(world, world.Components.Description.Get(entity).Description)
	}
	return entry
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *ItemActionState) View(world w.World, props ItemActionProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
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
	return newTabScreenUI(res, tabScreen{
		TabLabels: labels,
		TabIndex:  cursor.TabIndex,
		Content:   st.buildItemList(world, props, cursor.TabIndex, cursor.ItemIndex, res),
		Footer:    menuNavHint(world, true, query.T(world, "x Details")),
	})
}

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *ItemActionState) Menu(props ItemActionProps) menurt.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menurt.MenuConfig{Key: itemActionMenuKey, TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuItemsPerPage, InitialTab: verbTabIndex(st.initialVerb)}
}

func (st *ItemActionState) buildItemList(world w.World, props ItemActionProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	if tabIndex >= len(props.Tabs) {
		return styled.NewVerticalContainer()
	}
	items := props.Tabs[tabIndex].Items
	columnWidths := []int{260, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	rows := make([]menuRow, len(items))
	for i, it := range items {
		rows[i] = menuRow{Cells: []string{nameWithCount(it.Name, it.Count), it.Weight}}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No matching items")}, res)
}

// detailContent は現在カーソルが当たっているアイテムの詳細内容を返す。詳細モーダルの唯一の定義点
func (st *ItemActionState) detailContent(_ w.World) (menuscreen.DetailContent, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return menuscreen.DetailContent{}, false
	}
	items := props.Tabs[cursor.TabIndex].Items
	if cursor.ItemIndex >= len(items) {
		return menuscreen.DetailContent{}, false
	}
	item := items[cursor.ItemIndex]
	return menuscreen.DetailContent{Name: item.Name, Desc: item.Desc, Entity: item.Entity}, true
}
