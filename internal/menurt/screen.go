// Package menurt はメニュー画面の共通ランタイムを提供する。
//
// 各メニュー state から UI 機構、mount・widget・rebuild・overlay の入力ゲートと重ね、を Screen へ
// 集約し、state は Fetch・Menu・View と既存の DoAction を提供するだけにする。MVU の Model/View/Update
// に対応し、Screen がループを所有する。state package とは別 package にすることで、Model 契約を
// コンパイラに強制させ、state から Screen 内部へ触れられないようにする。詳細は docs/design/20260804_87.md。
package menurt

import (
	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
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
// DoAction・ConsumeTransition は既存の ActionHandler・BaseState をそのまま使う。入力変換は既定で
// HandleMenuInput を使い、独自キーが要る state だけ customInput を実装して上書きする
type Model[P any] interface {
	ConsumeTransition() es.Transition[w.World]
	DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error)
	Fetch(world w.World) P
	Menu(props P) MenuConfig
	View(world w.World, props P, sel Selection, res resources.UIResources) *ebitenui.UI
}

// customInput は既定のメニュー入力に加えて独自キーを扱う state が満たす任意契約。
// Screen は model がこれを満たすときだけ HandleInput を呼び、満たさなければ HandleMenuInput を使う
type customInput interface {
	HandleInput(cfg *config.Config) (inputmapper.ActionID, bool)
}

// Screen はメニューの UI ランタイム。mount・widget・rebuild と overlay・systems を保持し、
// 毎フレームの手順を回す。state は構造体にこれを値で埋め込み、Update と Draw を委譲する。
// コピーすると overlay のポインタが旧実体を指すため、コピーせず OnStart で NewScreen して使う
type Screen[P any] struct {
	model    Model[P] // メニュー画面本体。state 自身を指し、ループはこれ越しに部品を引く
	mount    *hooks.Mount[P]
	widget   *ebitenui.UI
	rebuild  bool
	overlays []menuscreen.Overlay
	systems  []w.Updater
	sel      Selection // 直近フレームで確定したカーソル位置。DoAction から参照する
	seeded   bool      // 初期タブへ寄せたか
}

// NewScreen は model と overlay を束ねて Screen を作る。model には state 自身を渡す。overlay は
// 優先順位順に、ポインタで渡し、state が保持する実体と同一を指す。追加 systems は WithSystems で登録する
func NewScreen[P any](model Model[P], overlays ...menuscreen.Overlay) Screen[P] {
	return Screen[P]{model: model, mount: hooks.NewMount[P](), overlays: overlays}
}

// WithSystems は毎フレーム回す systems を登録する
func (s *Screen[P]) WithSystems(systems ...w.Updater) *Screen[P] {
	s.systems = systems
	return s
}

// Open は overlay を開いて再描画を要求する。overlay の Open メソッドを渡すことで、
// 開いたのに UI の作り直しを忘れる取りこぼしを構造的に防ぐ
func (s *Screen[P]) Open(open func()) {
	open()
	s.rebuild = true
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

// readAction は1フレームの入力を Action に変換する。model が customInput を満たすときはその独自変換を、
// 満たさないときは共通の HandleMenuInput を使う。既定入力を Screen に集約し、独自キーが要る state
// だけが customInput で上書きする
func (s *Screen[P]) readAction(world w.World) (inputmapper.ActionID, bool) {
	if h, ok := s.model.(customInput); ok {
		return h.HandleInput(world.Config)
	}
	return HandleMenuInput()
}

// Update はメニュー1フレームを進める。systems 実行、入力ゲート、Fetch/SetProps、
// UseTabMenu、dirty なら View 再構築と overlay 重ね、widget.Update、の順で回す
func (s *Screen[P]) Update(world w.World) (es.Transition[w.World], error) {
	m := s.model
	// systems は登録済みインスタンスを名前で引いて回す。状態を持つ system を壊さない
	for _, u := range s.systems {
		if sys, ok := world.Updaters[u.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	// 入力ゲート。Active な最上位 overlay が専有し、無ければ通常入力を DoAction へ流す
	if ov := s.activeOverlay(); ov != nil {
		dirty, err := ov.HandleInput(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if dirty {
			s.rebuild = true
		}
	} else if action, ok := s.readAction(world); ok {
		if tr, err := m.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if tr.Type != es.TransNone {
			return tr, nil
		}
		// 遷移なしで入力を消費したフレームは表示が変わりうるので再構築を要求する。カーソル移動や
		// タブ切替は下の mount.Update が dirty を返すが、ドメイン操作は mount の外を変えるため
		// 検知できない。ここで一律 dirty にし、state 側の再構築の取りこぼしを構造的に防ぐ
		s.rebuild = true
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
		// 初回だけ指定タブへ寄せる
		if !s.seeded {
			if cfg.InitialTab > 0 {
				s.setTab(cfg, cfg.InitialTab)
			}
			s.seeded = true
		}
	}

	dirty := s.mount.Update()
	s.sel = s.selection(cfg)
	if dirty || s.widget == nil || s.rebuild {
		s.widget = m.View(world, props, s.sel, world.Resources.UIResources)
		for _, ov := range s.overlays {
			if ov.Active() {
				if win := ov.Window(world, menuscreen.CenterWindowRect(world)); win != nil {
					s.widget.AddWindow(win)
				}
			}
		}
		s.rebuild = false
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
	s.rebuild = true
}

// Selection は前フレームで確定したカーソル位置を返す。カーソルは DoAction のあとの
// mount.Update で更新されるため、DoAction 内で読むと画面に見えている確定位置になる
func (s *Screen[P]) Selection() Selection { return s.sel }

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
