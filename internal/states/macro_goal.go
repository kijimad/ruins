package states

import (
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
)

// NewGoalResultState は遠征達成の結果画面を作る。summary は到達側で算出した結果テキスト。
// 目標地点は旅の終わりでなく次の目的地への起点。ここで次の遠征を選んで旅を続けられる。
func NewGoalResultState(summary string) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		ms := &MessageState{}
		ms.messageData = messagedata.NewSystemMessage(summary).
			WithChoice("次の目的地を選ぶ", func(_ w.World) error {
				// 目標地点を起点に次の遠征を選ぶ（隊は引き継ぎ、新たなルートへ）
				ms.SetTransition(es.Transition[w.World]{
					Type:          es.TransSwitch,
					NewStateFuncs: []es.StateFactory[w.World]{NewExpeditionSelectState},
				})
				return nil
			}).
			WithChoice("遠征を終える", func(_ w.World) error {
				ms.SetTransition(es.Transition[w.World]{Type: es.TransPop})
				return nil
			})
		return ms, nil
	}
}
