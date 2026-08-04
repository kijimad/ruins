package states

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

// メニュー画面の共通ランタイム。各 state から UI 機構、mount・widget・rebuild・
// overlay の入力ゲートと重ね、を Screen へ集約し、state は Fetch・Menu・View と
// 既存の HandleInput・DoAction を提供するだけにする。MVU の Model/View/Update に対応し、
// Screen がループを所有する。詳細は docs/design/20260804_87.md。

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
	InitialTab   int      // 初回に寄せるタブ番号。0 なら先頭のまま。1度だけ適用する
}

// screenModel はメニュー1画面が Screen に対して満たす契約。UI 機構は持たず純粋な部品を提供する。
// HandleInput・DoAction・ConsumeTransition は既存の ActionHandler・BaseState をそのまま使う
type screenModel[P any] interface {
	ConsumeTransition() es.Transition[w.World]
	HandleInput(cfg *config.Config) (inputmapper.ActionID, bool)
	DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error)
	fetch(world w.World) P
	menu(props P) MenuConfig
	view(world w.World, props P, sel Selection, res resources.UIResources) *ebitenui.UI
}

// Screen はメニューの UI ランタイム。mount・widget・rebuild と overlay・systems を保持し、
// 毎フレームの手順を回す。state は構造体にこれを値で埋め込み、Update と Draw を委譲する。
// コピーすると overlay のポインタが旧実体を指すため、コピーせず OnStart で NewScreen して使う
type Screen[P any] struct {
	mount    *hooks.Mount[P]
	widget   *ebitenui.UI
	rebuild  bool
	overlays []menuscreen.Overlay
	systems  []w.Updater
	sel      Selection // 直近フレームで確定したカーソル位置。DoAction から参照する
	seeded   bool      // 初期タブへ寄せたか
}

// NewScreen は overlay を優先順位順に登録して Screen を作る。overlay はポインタで渡し、
// state が保持する実体と同一を指す。追加 systems は WithSystems で登録する
func NewScreen[P any](overlays ...menuscreen.Overlay) Screen[P] {
	return Screen[P]{mount: hooks.NewMount[P](), overlays: overlays}
}

// WithSystems は毎フレーム回す systems を登録する
func (s *Screen[P]) WithSystems(systems ...w.Updater) *Screen[P] {
	s.systems = systems
	return s
}

// MarkDirty は次フレームでの UI 再構築を要求する。ドメイン操作で表示が変わったときに呼ぶ
func (s *Screen[P]) MarkDirty() { s.rebuild = true }

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

// Update はメニュー1フレームを進める。systems 実行、入力ゲート、Fetch/SetProps、
// UseTabMenu、dirty なら View 再構築と overlay 重ね、widget.Update、の順で回す
func (s *Screen[P]) Update(world w.World, m screenModel[P]) (es.Transition[w.World], error) {
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
	} else if action, ok := m.HandleInput(world.Config); ok {
		if tr, err := m.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if tr.Type != es.TransNone {
			return tr, nil
		}
		s.mount.Dispatch(action)
	}

	props := m.fetch(world)
	s.mount.SetProps(props)
	cfg := m.menu(props)
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
				s.SetTab(cfg, cfg.InitialTab)
			}
			s.seeded = true
		}
	}

	dirty := s.mount.Update()
	s.sel = s.selection(cfg)
	if dirty || s.widget == nil || s.rebuild {
		s.widget = m.view(world, props, s.sel, world.Resources.UIResources)
		for _, ov := range s.overlays {
			if ov.Active() {
				if win := ov.Window(world, getCenterWinRect(world)); win != nil {
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
// UseTabMenu 登録後、つまり Update が1度回った後に呼ぶこと
func (s *Screen[P]) SetTab(cfg MenuConfig, tab int) {
	if cfg.TabCount == 0 {
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

// Selection は直近フレームで確定したカーソル位置を返す。DoAction は Dispatch より前に
// 呼ばれるため、ここで得るのは画面に見えている確定位置になる
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
