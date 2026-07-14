package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/route"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// expeditionChoice は遠征（背骨）の選択肢。§13 の遠征タイプに対応する。
type expeditionChoice struct {
	label string
	exp   route.ExpeditionType
}

// expeditionChoices は選択できる遠征の一覧（docs/design/20260712_54.md §13）。
var expeditionChoices = []expeditionChoice{
	{"深層ヴォールト（遺物を回収し納品）", route.ExpeditionDeepVault},
	{"交易都市（富を稼いで到達）", route.ExpeditionTradeCity},
	{"庇護者/派閥拠点（物資を護送）", route.ExpeditionPatron},
	{"辺境/未踏（未知の奥地へ）", route.ExpeditionFrontier},
}

// NewExpeditionSelectState は遠征（目的地＝背骨）を選ぶステートを作成する。
// 選ぶとルート網を生成してラン開始し、マクロ移動へ入る。
func NewExpeditionSelectState() (es.State[w.World], error) {
	messageState := &MessageState{}

	md := messagedata.NewSystemMessage("遠征を選ぶ（目的地＝背骨）")
	for _, choice := range expeditionChoices {
		exp := choice.exp
		md = md.WithChoice(choice.label, func(world w.World) error {
			if err := startExpedition(world, exp); err != nil {
				return err
			}
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransSwitch,
				NewStateFuncs: []es.StateFactory[w.World]{NewMacroMapState},
			})
			return nil
		})
	}
	md = md.WithChoice("やめる", func(_ w.World) error {
		messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	messageState.messageData = md
	return messageState, nil
}

// startExpedition は選んだ遠征でルート網を生成し、ラン状態を用意する。
// シードは毎ラン変わるため、遠征タイプが同じでも網は異なる。
func startExpedition(world w.World, exp route.ExpeditionType) error {
	// デバッグ入口ではプレイヤーが未生成のことがあるため用意する（正式には母港で済み）
	if err := ensureDebugParty(world); err != nil {
		return err
	}
	seed := world.Config.RNG.Uint64()
	query.SetCaravanRun(world, gc.NewCaravanRun(seed, exp))
	return nil
}
