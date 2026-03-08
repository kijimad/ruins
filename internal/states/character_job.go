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
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// CharacterJobState はキャラクター職業選択画面のステート
type CharacterJobState struct {
	es.BaseState[w.World]
	menuMount  *ui.Mount[jobMenuProps]
	widget     *ebitenui.UI
	playerName string // TODO: どうにかする。キャラメイクは複数のstateで構成され、前の決定事項を保持する必要がある...
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
var _ es.ActionHandler[w.World] = &CharacterJobState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CharacterJobState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CharacterJobState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *CharacterJobState) OnStart(_ w.World) error {
	st.menuMount = ui.NewMount[jobMenuProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CharacterJobState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CharacterJobState) Update(world w.World) (es.Transition[w.World], error) {
	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
	}

	// Props更新
	st.menuMount.SetProps(st.fetchProps())
	props := st.menuMount.GetProps()
	ui.UseTabMenu(st.menuMount.Store(), "job", ui.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	// dirty判定とUI再構築
	if st.menuMount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *CharacterJobState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(consts.BlackColor)
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *CharacterJobState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *CharacterJobState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("characterJob: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// jobMenuProps は職業選択メニューのProps
type jobMenuProps struct {
	Items []jobMenuItem
}

// jobMenuItem は職業メニューの項目
type jobMenuItem struct {
	Profession Profession
}

func (st *CharacterJobState) fetchProps() jobMenuProps {
	items := make([]jobMenuItem, len(professions))
	for i, p := range professions {
		items[i] = jobMenuItem{Profession: p}
	}
	return jobMenuProps{Items: items}
}

func (st *CharacterJobState) handleSelection(world w.World) (es.Transition[w.World], error) {
	props := st.menuMount.GetProps()
	itemIndex, ok := ui.GetState[int](st.menuMount, "job_itemIndex")
	if !ok {
		return es.Transition[w.World]{}, fmt.Errorf("job_itemIndexの取得に失敗")
	}

	if itemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	prof := props.Items[itemIndex].Profession
	st.selectProfession(world, prof)
	return st.ConsumeTransition(), nil
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

// ================
// buildUI
// ================

func (st *CharacterJobState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	itemIndex, _ := ui.GetState[int](st.menuMount, "job_itemIndex")

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

	// メニューコンテナを構築
	menuContainer := styled.NewVerticalContainer()
	for i, item := range props.Items {
		isSelected := i == itemIndex
		itemWidget := styled.NewListItemText(item.Profession.Name, consts.TextColor, isSelected, res)
		menuContainer.AddChild(itemWidget)
	}

	// 職業説明テキスト
	description := ""
	if itemIndex < len(props.Items) {
		description = props.Items[itemIndex].Profession.Description
	}
	descriptionText := widget.NewText(
		widget.TextOpts.Text(description, &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 0),
		),
	)

	// 操作ヒント
	hintLabel := widget.NewText(
		widget.TextOpts.Text(consts.IconArrowUp+consts.IconArrowDown+" 選択 / "+consts.IconKeyEnter+" 決定 / "+consts.IconKeyEsc+" 戻る", &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	centerContainer.AddChild(titleLabel)
	centerContainer.AddChild(menuContainer)
	centerContainer.AddChild(descriptionText)
	centerContainer.AddChild(hintLabel)

	rootContainer.AddChild(centerContainer)

	return &ebitenui.UI{Container: rootContainer}
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
