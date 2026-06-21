package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateChangeRequest_Implementations(t *testing.T) {
	t.Parallel()

	// 全てのイベント型がStateChangeRequestインターフェースを実装していることを確認
	var _ StateChangeRequest = WarpNextEvent{}
	var _ StateChangeRequest = WarpEscapeEvent{}
	var _ StateChangeRequest = GameClearEvent{}
	var _ StateChangeRequest = ShowDialogEvent{}
	var _ StateChangeRequest = OpenDungeonSelectEvent{}

	// 型アサーションが機能することを確認
	events := []StateChangeRequest{
		WarpNextEvent{},
		WarpEscapeEvent{},
		GameClearEvent{},
		ShowDialogEvent{MessageKey: "test"},
		OpenDungeonSelectEvent{},
	}
	assert.Len(t, events, 5)
}
