// Package menuloop はメニュー画面の共通ランタイムを提供する。
//
// 各メニュー state から UI 機構、mount・widget・overlay の入力ゲートと重ね、を Screen へ
// 集約し、state は Fetch・Menu・View と既存の DoAction を提供するだけにする。MVU の Model/View/Update
// に対応し、Screen がループを所有する。state package とは別 package にすることで、Model 契約を
// コンパイラに強制させ、state から Screen 内部へ触れられないようにする。
package menuloop

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/ui"
	w "github.com/kijimaD/ruins/internal/world"
)

// Selection は描画に要るカーソル位置。どのタブのどの行を強調するかを表す。
// PageSize は解決済みの1ページ件数。ItemsPerPageAuto を使う画面の描画が、カーソルと同じ
// ページ件数でページ送りできるよう渡す。0 はページ送りなし
type Selection struct {
	TabIndex  int
	ItemIndex int
	PageSize  int
}

// ItemsPerPageAuto は ItemsPerPage に、タブ帯つきモーダルの1ページへ収まる実測行数を
// 使う指定。実値は Update が UIResources から測って解決する。Menu は world を持たず
// 自力で測れないため、番兵で指定して Screen に解決させる。
// 測るのは見出し無し・タブ帯ありの構成に固定する。見出しを持つ画面はこの番兵を使わず、
// 自分の構成で menuframe.ListCapacity を呼んで求めた値を直接渡すこと
const ItemsPerPageAuto = -1

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
// Screen が ReadInput で扱い、独自キーが要る state は KeyBindings の表で宣言する。
// カーソル移動系は Screen が吸うので、DoAction には画面の意味を持つ Action だけが届く
type Model[P any] interface {
	ConsumeTransition() es.Transition[w.World]
	DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error)
	// Fetch は世界から表示 props を構築する。失敗は握りつぶさず error で返し、
	// Screen.Update がそのままフレームのエラーとして表面化させる
	Fetch(world w.World) (P, error)
	Menu(props P) MenuConfig
	// ViewUI は props とカーソルから本体を internal/ui のツリーへ組み、画面へ配置済みで返す
	ViewUI(world w.World, props P, cursor Selection, res resources.UIResources) ui.Widget
}

// KeyBindings は共通キーに加える独自キーを持つ state が満たす任意契約。キーと Action の
// 対応を表で返すだけで、キー読み取りの実行は keybind が担う。表は NewScreen が共通表と
// 1枚に合成し、共通キーとの重なりは構築時に拒否される。
// 実装 state は var _ menuloop.KeyBindings = &XState{} で綴りとシグネチャを静的に検証する
type KeyBindings interface {
	KeyBindings() []keybind.Binding
}

// Screen はメニューの UI ランタイム。mount・widget と overlay を保持し、毎フレームの
// 手順を回す。state は構造体にこれをポインタで持ち、Update と Draw を委譲する。
// widget は internal/ui のツリーを retained として扱い、props・カーソル・overlay が変わった
// フレームだけ組み直す。変化が無ければ前フレームのツリーを再利用する
type Screen[P any] struct {
	model Model[P] // メニュー画面本体。state 自身を指し、ループはこれ越しに部品を引く
	// table はこの画面のキー束縛。state 固有の断片と共通表を構築時に1枚へ合成済みで、
	// 実行時に表を重ねる階層は無い。重なりは MustMerge が構築時に拒否する
	table           []keybind.Binding
	mount           *hooks.Mount[P]
	bodyTree        ui.Widget // 本体の internal/ui ツリー。dirty なフレームだけ組み直す
	overlays        []overlay.Layer
	pendingOverlays []ui.Widget // ScreenRenderer な overlay の配置済みツリー。Draw で本体の上へ重ねる
	lastSelection   Selection   // 直近フレームで確定したカーソル位置。DoAction から参照する
	seeded          bool        // 初期タブへ寄せたか
	pageSize        int         // ItemsPerPageAuto の解決値。Update が最初のフレームで測る
}

// NewScreen は model と overlay を束ねて Screen を作る。model には state 自身を渡す。overlay は
// 優先順位順に、ポインタで渡し、state が保持する実体と同一を指す
func NewScreen[P any](model Model[P], overlays ...overlay.Layer) *Screen[P] {
	var frag []keybind.Binding
	if kb, ok := model.(KeyBindings); ok {
		frag = kb.KeyBindings()
	}
	return &Screen[P]{
		model:    model,
		table:    keybind.MustMerge(frag, keybind.MenuCommon),
		mount:    hooks.NewMount[P](),
		overlays: overlays,
	}
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

// dispatch は1件の Action を消費者の連鎖に流す。カーソルの mount、共通のキー一覧ヘルプ、
// 画面の DoAction の順に試し、先に消費したものが勝つ。DoAction には画面の意味を持つ
// Action だけが届く
func (s *Screen[P]) dispatch(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	if s.mount.DispatchNav(action) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	if action == inputmapper.ActionOpenKeyHelp {
		// ? のキー一覧ヘルプは全メニュー共通なので Screen が吸い、
		// この画面の合成済みの表から一覧を組んで push する
		return es.Transition[w.World]{Type: es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewKeyHelpState(s.table)}}, nil
	}
	return s.model.DoAction(world, action)
}

// Update はメニュー1フレームを進める。入力ゲート、Fetch/SetProps、
// UseTabMenu、dirty なら View 再構築と overlay 重ね、widget.Update、の順で回す
func (s *Screen[P]) Update(world w.World) (es.Transition[w.World], error) {
	m := s.model

	// 入力ゲート。Active な最上位 overlay が専有し、無ければ通常入力を dispatch の連鎖へ流す。
	// overlay が絡んだフレームは内容が入力で変わりうるので後段で必ず dirty にする
	ovBefore := s.activeOverlay()
	if ovBefore != nil {
		if err := ovBefore.HandleInput(world); err != nil {
			return es.Transition[w.World]{}, err
		}
	} else if action, ok := keybind.ReadInput(world, s.table); ok {
		if tr, err := s.dispatch(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if tr.Type != es.TransNone {
			return tr, nil
		}
	}

	props, err := m.Fetch(world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	s.mount.SetProps(props)
	cfg := s.resolveConfig(world, m.Menu(props))
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
	dirty := s.bodyTree == nil || overlayInvolved || changed
	if dirty {
		s.bodyTree = m.ViewUI(world, props, sel, world.Resources.UIResources)
		s.pendingOverlays = s.pendingOverlays[:0]
		rect := menuframe.WindowRect(world)
		// overlay は登録順で入力優先度が決まる。activeOverlay は先頭の Active を入力先にするので、
		// 描画は逆順に溜め、入力を受ける overlay を最前面にする。入れ子で開いた overlay が下に
		// 隠れて操作不能になるのを防ぐ。ツリーは Draw で本体の上へ重ねる
		for _, ov := range slices.Backward(s.overlays) {
			if !ov.Active() {
				continue
			}
			if r, ok := ov.(overlay.ScreenRenderer); ok {
				if tree := r.RenderOverlay(world, rect); tree != nil {
					s.pendingOverlays = append(s.pendingOverlays, tree)
				}
			}
		}
	}

	s.lastSelection = sel
	return m.ConsumeTransition(), nil
}

// SetTab は指定タブへ直接カーソルを移す。キー入力を介さずにタブを設定する。
// DoAction など入力処理の最中に呼ぶ。同じフレームの再構築で移動が反映される。
// UseTabMenu 登録後、つまり Update が1度回った後に呼ぶこと。範囲外の tab は無視する。
// 構成は model から導出するので呼び出し側は tab 番号だけを渡す
func (s *Screen[P]) SetTab(tab int) {
	cfg := s.model.Menu(s.Props())
	if cfg.ItemsPerPage == ItemsPerPageAuto {
		// SetTab は Update 後に呼ぶ前提なので、Update が測定済みの値をそのまま使う
		cfg.ItemsPerPage = s.pageSize
	}
	s.setTab(cfg, tab)
}

// resolveConfig は MenuConfig の ItemsPerPageAuto を実測のページ行数へ解決する。
// 測定は Auto 指定の画面が最初に通ったときだけ行い、以後は測定済みの値を使う
func (s *Screen[P]) resolveConfig(world w.World, cfg MenuConfig) MenuConfig {
	if cfg.ItemsPerPage == ItemsPerPageAuto {
		if s.pageSize == 0 {
			s.pageSize = menuframe.ListCapacity(world, false, true)
		}
		cfg.ItemsPerPage = s.pageSize
	}
	return cfg
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
	return Selection{TabIndex: ms.TabIndex, ItemIndex: ms.ItemIndex, PageSize: cfg.ItemsPerPage}
}

// Draw は保持中の UI を描き、その上に ScreenRenderer な overlay を重ねる。
// 各 state の Draw はこれへ委譲する
func (s *Screen[P]) Draw(screen *ebiten.Image) {
	cv := ui.NewEbitenCanvas(screen)
	if s.bodyTree != nil {
		s.bodyTree.Draw(cv)
	}
	for _, tree := range s.pendingOverlays {
		tree.Draw(cv)
	}
}
