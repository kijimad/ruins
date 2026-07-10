package components

import (
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestStateChangeRequest_Constructors(t *testing.T) {
	t.Parallel()

	// 各コンストラクタが正しい Kind のリクエストを生成することを確認
	assert.Equal(t, EventWarpNext, WarpNextEvent().Kind)
	assert.Equal(t, EventWarpEscape, WarpEscapeEvent().Kind)
	assert.Equal(t, EventGameClear, GameClearEvent().Kind)
	assert.Equal(t, EventOpenDungeonSelect, OpenDungeonSelectEvent().Kind)

	// ペイロード付きのイベントもフィールドが設定されることを確認
	dialog := ShowDialogEvent("test", ecs.Entity{})
	assert.Equal(t, EventShowDialog, dialog.Kind)
	assert.Equal(t, "test", dialog.MessageKey)

	storage := OpenStorageEvent(ecs.Entity{})
	assert.Equal(t, EventOpenStorage, storage.Kind)
}
