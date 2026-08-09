// Package menurt はメニュー画面の共通ランタイムを提供する。
//
// 各メニュー state から UI 機構、mount・widget・overlay の入力ゲートと重ね、を Screen へ
// 集約し、state は Fetch・Menu・View と既存の DoAction を提供するだけにする。MVU の Model/View/Update
// に対応し、Screen がループを所有する。state package とは別 package にすることで、Model 契約を
// コンパイラに強制させ、state から Screen 内部へ触れられないようにする。
package menurt

import (
	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	w "github.com/kijimaD/ruins/internal/world"
)

// Selection は描画に要るカーソル位置。どのタブのどの行を強調するかを表す
type Selection struct {
	TabIndex  int
	ItemIndex int
}

// MenuConfig は Fetch 済みの props から決まるメニュー構成。state 固有差をここで吸収する。
// TabCount が 0 のときは一覧を持たない画面として UseTabMenu を登録しない
type MenuConfig struct {
	Key          string
	TabCount     int
	ItemCounts   []int
	ItemsPerPage int      // 1ページの件数。0 はページ送りなし
	Skips        [][]bool // カーソルを飛ばす見出し行。nil 可
	InitialTab   int      // 初回に寄せるタブ番号。0 なら先頭のまま。1度だけ適用する。TabCount 未満を前提とする
}

// Model はメニュー1画面が Screen に対して満たす契約。UI 機構は持たず純粋な部品を提供する。
// DoAction・ConsumeTransition は既存の ActionHandler・BaseState をそのまま使う。共通のメニュー入力は
// Screen が HandleMenuInput で必ず適用し、独自キーが要る state だけ ExtraInput でその分を足す
type Model[P any] interface {
	ConsumeTransition() es.Transition[w.World]
	DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error)
	Fetch(world w.World) P
	Menu(props P) MenuConfig
	View(world w.World, props P, cursor Selection, res resources.UIResources) *ebitenui.UI
}

// ExtraInput は共通のメニュー入力に加えて独自キーを扱う state が満たす任意契約。
// 返すのは独自キーの分だけでよい。共通の HandleMenuInput は Screen が必ず後段で適用する。
// 実装 state は var _ menurt.ExtraInput = &XState{} で綴りとシグネチャを静的に検証する
type ExtraInput interface {
	ExtraInput() (inputmapper.ActionID, bool)
}

// Screen はメニューの UI ランタイム。mount・widget と overlay を保持し、毎フレームの
// 手順を回す。state は構造体にこれをポインタで持ち、Update と Draw を委譲する。widget は毎フレーム
// View から組み直すので、表示は常に最新の props とカーソルに追従する
type Screen[P any] struct {
	model         Model[P] // メニュー画面本体。state 自身を指し、ループはこれ越しに部品を引く
	mount         *hooks.Mount[P]
	widget        *ebitenui.UI
	overlays      []menuscreen.Overlay
	lastSelection Selection // 直近フレームで確定したカーソル位置。DoAction から参照する
	seeded        bool      // 初期タブへ寄せたか
}

// NewScreen は model と overlay を束ねて Screen を作る。model には state 自身を渡す。overlay は
// 優先順位順に、ポインタで渡し、state が保持する実体と同一を指す
func NewScreen[P any](model Model[P], overlays ...menuscreen.Overlay) *Screen[P] {
	return &Screen[P]{model: model, mount: hooks.NewMount[P](), overlays: overlays}
}

// Props は現在の props を返す。View 以外から現在値を参照する必要があるとき使う
func (s *Screen[P]) Props() P { return s.mount.GetProps() }

// activeOverlay は登録順で最初の Active な overlay を返す。無ければ nil
func (s *Screen[P]) activeOverlay() menuscreen.Overlay {
	for _, ov := range s.overlays {
		if ov.Active() {
			return ov
		}
	}
	return nil
}

// readAction は1フレームの入力を Action に変換する。独自キーを持つ state は extraInput でその分だけを
// 返し、共通の HandleMenuInput は必ず後段で適用する。共通入力を Screen に集約し、state には追加分だけ書かせる
func (s *Screen[P]) readAction() (inputmapper.ActionID, bool) {
	if h, ok := s.model.(ExtraInput); ok {
		if action, ok := h.ExtraInput(); ok {
			return action, true
		}
	}
	return HandleMenuInput()
}

// Update はメニュー1フレームを進める。入力ゲート、Fetch/SetProps、
// UseTabMenu、View 再構築と overlay 重ね、widget.Update、の順で回す
func (s *Screen[P]) Update(world w.World) (es.Transition[w.World], error) {
	m := s.model

	// 入力ゲート。Active な最上位 overlay が専有し、無ければ通常入力を DoAction へ流す
	if ov := s.activeOverlay(); ov != nil {
		if err := ov.HandleInput(world); err != nil {
			return es.Transition[w.World]{}, err
		}
	} else if action, ok := s.readAction(); ok {
		if tr, err := m.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if tr.Type != es.TransNone {
			return tr, nil
		}
		s.mount.Dispatch(action)
	}

	props := m.Fetch(world)
	s.mount.SetProps(props)
	cfg := m.Menu(props)
	if cfg.TabCount > 0 {
		hooks.UseTabMenu(s.mount.Store(), cfg.Key, hooks.TabMenuConfig{
			TabCount:     cfg.TabCount,
			ItemCounts:   cfg.ItemCounts,
			ItemsPerPage: cfg.ItemsPerPage,
			Skips:        cfg.Skips,
		})
		if !s.seeded {
			if cfg.InitialTab > 0 {
				s.setTab(cfg, cfg.InitialTab)
			}
			s.seeded = true
		}
	}

	// カーソル状態を進める。widget は毎フレーム組み直すので dirty 判定は要らない
	s.mount.Update()
	s.lastSelection = s.selection(cfg)
	s.widget = m.View(world, props, s.lastSelection, world.Resources.UIResources)
	// overlay は登録順で入力優先度が決まる。activeOverlay は先頭の Active を入力先にするので、
	// 描画は逆順に重ね、入力を受ける overlay を最前面にする。入れ子で開いた overlay が下に
	// 隠れて操作不能になるのを防ぐ
	for i := len(s.overlays) - 1; i >= 0; i-- {
		ov := s.overlays[i]
		if ov.Active() {
			if win := ov.Window(world, menuscreen.CenterWindowRect(world)); win != nil {
				s.widget.AddWindow(win)
			}
		}
	}

	s.widget.Update()
	return m.ConsumeTransition(), nil
}

// SetTab は指定タブへ直接カーソルを移して再描画を要求する。キー再生をせずにタブを設定する。
// UseTabMenu 登録後、つまり Update が1度回った後に呼ぶこと。範囲外の tab は無視する。
// 構成は model から導出するので呼び出し側は tab 番号だけを渡す
func (s *Screen[P]) SetTab(tab int) {
	s.setTab(s.model.Menu(s.Props()), tab)
}

// setTab は構成を渡してタブを設定する内部処理。Update の初期タブ寄せと公開 SetTab が共有する
func (s *Screen[P]) setTab(cfg MenuConfig, tab int) {
	if cfg.TabCount == 0 || tab < 0 || tab >= cfg.TabCount {
		return
	}
	hooks.SetTab(s.mount.Store(), cfg.Key, hooks.TabMenuConfig{
		TabCount:     cfg.TabCount,
		ItemCounts:   cfg.ItemCounts,
		ItemsPerPage: cfg.ItemsPerPage,
		Skips:        cfg.Skips,
	}, tab)
}

// Selection は前フレームで確定したカーソル位置を返す。カーソルは DoAction のあとの
// mount.Update で更新されるため、DoAction 内で読むと画面に見えている確定位置になる
func (s *Screen[P]) Selection() Selection { return s.lastSelection }

// selection は現在のカーソル位置を mount から読む。一覧を持たない画面はゼロ値
func (s *Screen[P]) selection(cfg MenuConfig) Selection {
	if cfg.TabCount == 0 {
		return Selection{}
	}
	ms, _ := hooks.GetState[hooks.TabMenuState](s.mount, cfg.Key)
	return Selection{TabIndex: ms.TabIndex, ItemIndex: ms.ItemIndex}
}

// Draw は保持中の UI を描く。各 state の Draw はこれへ委譲する
func (s *Screen[P]) Draw(screen *ebiten.Image) {
	if s.widget != nil {
		s.widget.Draw(screen)
	}
}
