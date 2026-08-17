// Package menuloop はメニュー画面の共通ランタイムを提供する。
//
// 各メニュー state から UI 機構、mount・widget・overlay の入力ゲートと重ね、を Screen へ
// 集約し、state は Fetch・Menu・View と既存の DoAction を提供するだけにする。MVU の Model/View/Update
// に対応し、Screen がループを所有する。state package とは別 package にすることで、Model 契約を
// コンパイラに強制させ、state から Screen 内部へ触れられないようにする。
package menuloop

import (
	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
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
// DoAction・ConsumeTransition は既存の ActionHandler・BaseState をそのまま使う。メニュー入力は
// Screen が HandleMenuInput で扱い、独自キーが要る state だけ ExtraInput で先取りする
type Model[P any] interface {
	ConsumeTransition() es.Transition[w.World]
	DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error)
	Fetch(world w.World) P
	Menu(props P) MenuConfig
	View(world w.World, props P, cursor Selection, res resources.UIResources) *ebitenui.UI
}

// ExtraInput は独自キーを扱う state が満たす任意契約。Screen は各フレームでまず ExtraInput を試し、
// 拾わなければ共通の HandleMenuInput へフォールバックする。1フレーム1アクションで ExtraInput が優先する。
// 実装 state は var _ menuloop.ExtraInput = &XState{} で綴りとシグネチャを静的に検証する
type ExtraInput interface {
	ExtraInput() (inputmapper.ActionID, bool)
}

// Screen はメニューの UI ランタイム。mount・widget と overlay を保持し、毎フレームの
// 手順を回す。state は構造体にこれをポインタで持ち、Update と Draw を委譲する。
// widget は ebitenui を retained として扱い、props・カーソル・overlay が変わったフレームだけ組み直す。
// 変化が無ければ前フレームのツリーを再利用する
type Screen[P any] struct {
	model         Model[P] // メニュー画面本体。state 自身を指し、ループはこれ越しに部品を引く
	mount         *hooks.Mount[P]
	widget        *ebitenui.UI
	overlays      []overlay.Layer
	lastSelection Selection // 直近フレームで確定したカーソル位置。DoAction から参照する
	seeded        bool      // 初期タブへ寄せたか
	// commands は任意のコマンド供給源。nil なら本番のキーボード経路。
	// 設定時はキーボードより優先し、決めた順にコマンドを1フレーム1件で返す。再生ドライバが差す
	commands func() (inputmapper.ActionID, bool)
}

// SetCommandSource は再生用のコマンド供給源を差す。本番は呼ばず nil のままで、テストや再生ドライバから使う。
// 供給源は1フレーム1件でコマンドを返し、尽きたら (_, false) を返してキーボード経路へ戻す
func (s *Screen[P]) SetCommandSource(src func() (inputmapper.ActionID, bool)) {
	s.commands = src
}

// NewScreen は model と overlay を束ねて Screen を作る。model には state 自身を渡す。overlay は
// 優先順位順に、ポインタで渡し、state が保持する実体と同一を指す
func NewScreen[P any](model Model[P], overlays ...overlay.Layer) *Screen[P] {
	return &Screen[P]{model: model, mount: hooks.NewMount[P](), overlays: overlays}
}

// Props は現在の props を返す。View 以外から現在値を参照する必要があるとき使う
func (s *Screen[P]) Props() P { return s.mount.GetProps() }

// activeOverlay は登録順で最初の Active な overlay を返す。無ければ nil
func (s *Screen[P]) activeOverlay() overlay.Layer {
	for _, ov := range s.overlays {
		if ov.Active() {
			return ov
		}
	}
	return nil
}

// readAction は1フレームの入力を Action に変換する。ExtraInput を持つ state はそれを先に試し、
// 拾わなければ共通の HandleMenuInput にフォールバックする。1フレーム1アクションで ExtraInput が優先する
func (s *Screen[P]) readAction() (inputmapper.ActionID, bool) {
	if s.commands != nil {
		if action, ok := s.commands(); ok {
			return action, true
		}
	}
	if h, ok := s.model.(ExtraInput); ok {
		if action, ok := h.ExtraInput(); ok {
			return action, true
		}
	}
	return HandleMenuInput()
}

// Update はメニュー1フレームを進める。入力ゲート、Fetch/SetProps、
// UseTabMenu、dirty なら View 再構築と overlay 重ね、widget.Update、の順で回す
func (s *Screen[P]) Update(world w.World) (es.Transition[w.World], error) {
	m := s.model

	// 入力ゲート。Active な最上位 overlay が専有し、無ければ通常入力を DoAction へ流す。
	// overlay が絡んだフレームは内容が入力で変わりうるので後段で必ず dirty にする
	ovBefore := s.activeOverlay()
	if ovBefore != nil {
		if err := ovBefore.HandleInput(world); err != nil {
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

	// mount の dirty を読む。初回・props 変化・入力受理のいずれかで立つ。カーソルは Dispatch で更新済み
	changed := s.mount.Update()
	sel := s.selection(cfg)

	// dirty なフレームだけ widget ツリーを組み直す。mount の dirty を状態変化の答えとし、
	// hooks の外にある overlay だけ別途 OR する。overlay が開閉・表示中のフレームは窓内容が
	// 入力で変わりうるので常に組み直す
	overlayInvolved := ovBefore != nil || s.activeOverlay() != nil
	dirty := s.widget == nil || overlayInvolved || changed
	if dirty {
		s.widget = m.View(world, props, sel, world.Resources.UIResources)
		// overlay は登録順で入力優先度が決まる。activeOverlay は先頭の Active を入力先にするので、
		// 描画は逆順に重ね、入力を受ける overlay を最前面にする。入れ子で開いた overlay が下に
		// 隠れて操作不能になるのを防ぐ
		for i := len(s.overlays) - 1; i >= 0; i-- {
			ov := s.overlays[i]
			if ov.Active() {
				if win := ov.Window(world, menuframe.CenterWindowRect(world)); win != nil {
					s.widget.AddWindow(win)
				}
			}
		}
	}

	s.lastSelection = sel
	s.widget.Update()
	return m.ConsumeTransition(), nil
}

// SetTab は指定タブへ直接カーソルを移す。キー入力を介さずにタブを設定する。
// DoAction など入力処理の最中に呼ぶ。同じフレームの再構築で移動が反映される。
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
// Dispatch で更新されるため、DoAction 内で読むと更新前、つまり画面に見えている確定位置になる
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
