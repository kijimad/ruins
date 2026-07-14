package states

import (
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
)

// NewGoalResultState は遠征達成のスコア画面を作る。summary は到達側で算出した結果テキスト。
// 「母港へ戻る」で Pop し、Switch 元（母港）へ戻る。
func NewGoalResultState(summary string) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		ms := &MessageState{}
		ms.messageData = messagedata.NewSystemMessage(summary).
			WithChoice("母港へ戻る", func(_ w.World) error {
				ms.SetTransition(es.Transition[w.World]{Type: es.TransPop})
				return nil
			})
		return ms, nil
	}
}
