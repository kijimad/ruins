package states

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// FeedFuelMenuState は給油メニュー。持ち物の燃料をくべ先 FuelSink へくべる。
// くべても閉じず、続けて足せる。
type FeedFuelMenuState struct {
	es.BaseState[w.World]
	sink   FuelSink
	detail overlay.Detail
	screen *menuloop.Screen[FeedFuelProps]
}

// FuelSink は給油メニューのくべ先。一覧に出す燃料と選んでくべる燃料は同じ accepts で決める。
// done が真を返せば、くべた後にメニューを閉じる
type FuelSink struct {
	title   func(world w.World) string
	accepts func(world w.World, fuel ecs.Entity) bool
	gain    func(world w.World, fuel ecs.Entity) string
	feed    func(world w.World, fuel ecs.Entity) error
	done    func(world w.World) bool
}

// FireFuelSink は火をくべ先にする。増える量は燃焼ターン数で表す。
// メニュー表示中に火が燃え尽きても AddFuel が何もしないので、火の状態はここで確かめない
func FireFuelSink(fire ecs.Entity) FuelSink {
	return FuelSink{
		title: func(world w.World) string {
			return query.T(world, "Burning, %s left", query.EstimateBurnTurns(world, fire).String())
		},
		accepts: query.IsCombustible,
		gain: func(world w.World, fuel ecs.Entity) string {
			return query.FuelBurnTurns(world, fire, fuel).String()
		},
		feed: func(world w.World, fuel ecs.Entity) error {
			lifecycle.AddFuel(world, fire, fuel)
			return nil
		},
	}
}

// CubeFuelSink はキューブをくべ先にする。増える量は燃料ポイントで表す。
func CubeFuelSink(cube ecs.Entity) FuelSink {
	return FuelSink{
		title: func(world w.World) string {
			return query.T(world, "Fuel %s", cubeFuelText(*world.Components.Charge.Get(cube)))
		},
		accepts: query.IsCombustible,
		gain: func(world w.World, fuel ecs.Entity) string {
			return query.HeatContent(world, fuel).String()
		},
		feed: func(world w.World, fuel ecs.Entity) error {
			if !lifecycle.AddCubeFuel(world, cube, fuel) {
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "The fuel tank is full.")).
					Log()
			}
			return nil
		},
	}
}

// LightFuelSink は満たせる灯りをくべ先にする。燃料の残量を移すので、1つで満タンになるとは限らない。満タンになったらメニューを閉じる。
// メニューを開いている間はターンが進まず灯りが消えないので、灯りの生存は確かめない
func LightFuelSink(world w.World, light ecs.Entity) FuelSink {
	want, refillable := query.AcceptedFuelOf(world, light)
	return FuelSink{
		title: func(world w.World) string {
			return query.T(world, "Refill %s", query.GetEntityName(light, world))
		},
		accepts: func(world w.World, fuel ecs.Entity) bool {
			return refillable && query.FuelKindMatches(world, want, fuel) && world.Components.Charge.Get(fuel).Current > 0
		},
		gain: func(world w.World, fuel ecs.Entity) string {
			return consts.Turn(query.TransferableCharge(world, fuel, light)).String()
		},
		feed: func(world w.World, fuel ecs.Entity) error {
			_, err := gameaction.TransferCharge(world, fuel, light)
			if errors.Is(err, gameaction.ErrChargeFull) {
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "It is already full.")).
					Log()
				return nil
			}
			return err
		},
		done: func(world w.World) bool {
			charge := world.Components.Charge.Get(light)
			return charge.Current >= charge.Max
		},
	}
}

// LightDrainSink は灯りの残量を抜いて入れる、同じ種別の満タンでない燃料を並べる。灯りが0になるか、抜く先が無くなったらメニューを閉じる
func LightDrainSink(world w.World, light ecs.Entity) FuelSink {
	want, refillable := query.AcceptedFuelOf(world, light)
	var sink FuelSink
	sink = FuelSink{
		title: func(world w.World) string {
			return query.T(world, "Drain %s", query.GetEntityName(light, world))
		},
		accepts: func(world w.World, container ecs.Entity) bool {
			if !refillable || !query.FuelKindMatches(world, want, container) {
				return false
			}
			charge := world.Components.Charge.Get(container)
			return charge.Current < charge.Max
		},
		gain: func(world w.World, container ecs.Entity) string {
			return consts.Turn(query.TransferableCharge(world, light, container)).String()
		},
		feed: func(world w.World, container ecs.Entity) error {
			if _, err := gameaction.TransferCharge(world, light, container); err != nil {
				return err
			}
			gamelog.New(query.GetGameLog(world)).
				Markup(query.T(world, "Drained %s into %s.",
					gamelog.Tag("item", query.GetEntityName(light, world)),
					gamelog.Tag("item", query.GetEntityName(container, world)))).
				Log()
			return nil
		},
		done: func(world w.World) bool {
			return world.Components.Charge.Get(light).Current <= 0 || !carriesAccepted(world, sink)
		},
	}
	return sink
}

var _ es.State[w.World] = &FeedFuelMenuState{}
var _ menuloop.KeyBindings = &FeedFuelMenuState{}

// FeedFuelProps は給油メニューの表示 props
type FeedFuelProps struct {
	Title string
	Rows  []feedFuelRow
}

// feedFuelRow は給油一覧の1行。燃料スタックの代表と個数、1つくべると増える量の表示を持つ
type feedFuelRow struct {
	Entity ecs.Entity
	Count  int
	Gain   string
}

// OnStart はステートが開始される際に呼ばれる
func (st *FeedFuelMenuState) OnStart(_ w.World) error {
	st.detail = overlay.NewEntityDetail(st.selectedEntity)
	st.screen = menuloop.NewScreen[FeedFuelProps](st, &st.detail)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *FeedFuelMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *FeedFuelMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// KeyBindings は x の詳細表示を共通入力に足す
func (st *FeedFuelMenuState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
}

// DoAction はActionを実行する。選択で1つくべてメニューに留まり、続けて足せる
func (st *FeedFuelMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
	case inputmapper.ActionMenuSelect:
		// 走査時の代表は消費されるので、次フレームの Fetch が新しい代表で一覧を組み直す
		if entity, ok := st.selectedEntity(); ok && world.ECS.Alive(entity) && st.sink.accepts(world, entity) {
			if err := st.sink.feed(world, entity); err != nil {
				return es.Transition[w.World]{}, err
			}
			if st.sink.done != nil && st.sink.done(world) {
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}
		}
	default:
		return es.Transition[w.World]{}, fmt.Errorf("feedFuelMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// Fetch は世界から表示 props を構築する
func (st *FeedFuelMenuState) Fetch(world w.World) (FeedFuelProps, error) {
	title := st.sink.title(world)
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return FeedFuelProps{Title: title}, err
	}
	var rows []feedFuelRow
	for _, stack := range query.BackpackStacks(world, player) {
		if !st.sink.accepts(world, stack.Rep) {
			continue
		}
		rows = append(rows, feedFuelRow{
			Entity: stack.Rep,
			Count:  stack.Count,
			Gain:   st.sink.gain(world, stack.Rep),
		})
	}
	return FeedFuelProps{Title: title, Rows: rows}, nil
}

// Menu は単一リストの構成を返す
func (st *FeedFuelMenuState) Menu(props FeedFuelProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "feedfuel", TabCount: 1, ItemCounts: []int{len(props.Rows)}, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

// ViewUI は燃料一覧を収納やインベントリと同じ item-row 形式で組む。
// アイコン・名前の共通先頭に、右寄せの増える量の列を足す
func (st *FeedFuelMenuState) ViewUI(world w.World, props FeedFuelProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	cols := itemMenuColumns(styled.Num())
	rows := make([]menuframe.Row, len(props.Rows))
	for i, r := range props.Rows {
		rows[i] = itemMenuRow(world, r.Entity, r.Count, r.Gain)
	}
	perPage := menuframe.ListCapacity(world, false, true)
	list, pager := menuframe.RenderList(cursor.ItemIndex, rows, cols, menuframe.ListOpts{EmptyText: query.T(world, "No fuel to add"), ItemsPerPage: perPage}, res)
	return menuframe.PanelScreen(world, res, props.Title, list, keybind.HelpHint(world), pager)
}

// selectedEntity は現在カーソルが当たっている燃料の代表エンティティを返す
func (st *FeedFuelMenuState) selectedEntity() (ecs.Entity, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.ItemIndex < 0 || cursor.ItemIndex >= len(props.Rows) {
		return ecs.Entity{}, false
	}
	return props.Rows[cursor.ItemIndex].Entity, true
}
