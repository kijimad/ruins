package components

import (
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateChangeRequest_Constructors(t *testing.T) {
	t.Parallel()

	// 各コンストラクタが正しい種別のペイロードを生成することを確認
	assert.IsType(t, WarpNext{}, WarpNextEvent().Payload)
	assert.IsType(t, WarpEscape{}, WarpEscapeEvent().Payload)
	assert.IsType(t, GameClear{}, GameClearEvent().Payload)
	assert.IsType(t, OpenDungeonSelect{}, OpenDungeonSelectEvent().Payload)

	// ペイロード付きのイベントもフィールドが設定されることを確認
	dialog := ShowDialogEvent("test", ecs.Entity{})
	require.IsType(t, ShowDialog{}, dialog.Payload)
	showDialog, ok := dialog.Payload.(ShowDialog)
	require.True(t, ok, "型が ShowDialog であるべき")
	assert.Equal(t, "test", showDialog.MessageKey)

	storage := OpenStorageEvent(ecs.Entity{})
	assert.IsType(t, OpenStorage{}, storage.Payload)
}
