package states

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// CharacterJobState はキャラクター職業選択画面のステート
type CharacterJobState struct {
	es.BaseState[w.World]
	ui              *ebitenui.UI
	menuView        *tabmenu.View
	descriptionText *widget.Text
	playerName      string
}

// NewCharacterJobState は職業選択ステートのファクトリを返す
func NewCharacterJobState(playerName string) es.StateFactory[w.World] {
	return func() es.State[w.World] {
		return &CharacterJobState{
			playerName: playerName,
		}
	}
}

func (st CharacterJobState) String() string {
	return "CharacterJob"
}

// State interface ================

var _ es.State[w.World] = &CharacterJobState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CharacterJobState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CharacterJobState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *CharacterJobState) OnStart(world w.World) error {
	st.initMenu(world)
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CharacterJobState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CharacterJobState) Update(_ w.World) (es.Transition[w.World], error) {
	if err := st.menuView.Update(); err != nil {
		return es.Transition[w.World]{Type: es.TransNone}, err
	}

	if st.ui != nil {
		st.ui.Update()
	}

	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *CharacterJobState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(consts.BlackColor)

	if st.ui != nil {
		st.ui.Draw(screen)
	}
	return nil
}

// ================

// initMenu はメニューコンポーネントを初期化する
func (st *CharacterJobState) initMenu(world w.World) {
	items := make([]tabmenu.Item, len(professions))
	for i, p := range professions {
		items[i] = tabmenu.Item{
			ID:       p.ID,
			Label:    p.Name,
			UserData: p,
		}
	}

	tabs := []tabmenu.TabItem{
		{
			ID:    "professions",
			Label: "",
			Items: items,
		},
	}

	config := tabmenu.Config{
		Tabs:             tabs,
		InitialTabIndex:  0,
		InitialItemIndex: 0,
		WrapNavigation:   true,
		ItemsPerPage:     10,
	}

	callbacks := tabmenu.Callbacks{
		OnSelectItem: func(_ int, _ int, _ tabmenu.TabItem, item tabmenu.Item) error {
			if prof, ok := item.UserData.(Profession); ok {
				st.selectProfession(world, prof)
			}
			return nil
		},
		OnItemChange: func(_ int, _, _ int, item tabmenu.Item) error {
			if prof, ok := item.UserData.(Profession); ok {
				st.updateDescription(prof)
			}
			return nil
		},
		OnCancel: func() {
			st.cancel()
		},
	}

	st.menuView = tabmenu.NewView(config, callbacks, world)
}

// initUI はUIを初期化する
func (st *CharacterJobState) initUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	centerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(20),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	titleLabel := widget.NewText(
		widget.TextOpts.Text("職業", &res.Text.TitleFontFace, consts.PrimaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	// TabMenuのUIを取得
	menuContainer := st.menuView.BuildUI()

	// 職業説明テキスト
	st.descriptionText = widget.NewText(
		widget.TextOpts.Text(professions[0].Description, &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 0),
		),
	)

	// 操作ヒント
	hintLabel := widget.NewText(
		widget.TextOpts.Text("Enter: 決定 / Esc: 戻る", &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	centerContainer.AddChild(titleLabel)
	centerContainer.AddChild(menuContainer)
	centerContainer.AddChild(st.descriptionText)
	centerContainer.AddChild(hintLabel)

	rootContainer.AddChild(centerContainer)

	return &ebitenui.UI{Container: rootContainer}
}

// updateDescription は職業説明を更新する
func (st *CharacterJobState) updateDescription(prof Profession) {
	if st.descriptionText != nil {
		st.descriptionText.Label = prof.Description
	}
}

// selectProfession は職業を選択してゲームを開始する
func (st *CharacterJobState) selectProfession(world w.World, prof Profession) {
	// プレイヤーを生成
	_, _ = worldhelper.SpawnPlayer(world, 5, 5, st.playerName)

	// 初期装備を付与
	for _, item := range prof.Items {
		_, _ = worldhelper.SpawnItem(world, item.Name, item.Count, gc.ItemLocationInPlayerBackpack)
	}

	st.SetTransition(es.Transition[w.World]{
		Type:          es.TransReplace,
		NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
	})
}

// cancel はキャンセルして名前入力に戻る
func (st *CharacterJobState) cancel() {
	st.SetTransition(es.Transition[w.World]{
		Type: es.TransPop,
	})
}

// ================
// 職業はあとで実装する
// ================

// Profession は職業を表す
type Profession struct {
	ID          string
	Name        string
	Description string
	Items       []ProfessionItem
}

// ProfessionItem は職業の初期装備アイテムを表す
type ProfessionItem struct {
	Name  string
	Count int
}

// 職業一覧
var professions = []Profession{
	{
		ID:          "evacuee",
		Name:        "避難民",
		Description: "一般市民。特別な装備はない",
		Items: []ProfessionItem{
			{Name: "パン", Count: 3},
			{Name: "回復薬", Count: 1},
		},
	},
	{
		ID:          "soldier",
		Name:        "軍人",
		Description: "元兵士。武器の扱いに慣れている",
		Items: []ProfessionItem{
			{Name: "ハンドガン", Count: 1},
			{Name: "木刀", Count: 1},
			{Name: "回復薬", Count: 1},
		},
	},
	{
		ID:          "mechanic",
		Name:        "整備士",
		Description: "機械の修理が得意",
		Items: []ProfessionItem{
			{Name: "鉄", Count: 5},
			{Name: "回復薬", Count: 2},
		},
	},
	{
		ID:          "hunter",
		Name:        "猟師",
		Description: "野外活動に長けている",
		Items: []ProfessionItem{
			{Name: "ハンドガン", Count: 1},
			{Name: "パン", Count: 5},
			{Name: "緑ハーブ", Count: 2},
		},
	},
}
