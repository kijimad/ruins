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
	assert.IsType(t, WarpDescend{}, WarpDescendEvent().Payload)
	assert.IsType(t, WarpAscend{}, WarpAscendEvent().Payload)
	assert.IsType(t, WarpRuinExit{}, WarpRuinExitEvent().Payload)
	assert.IsType(t, GameClear{}, GameClearEvent().Payload)
	assert.IsType(t, OpenDungeonSelect{}, OpenDungeonSelectEvent().Payload)

	// 遺跡進入は遺跡定義名を運ぶ
	ruinEnter := WarpRuinEnterEvent("ancient_ruin")
	require.IsType(t, WarpRuinEnter{}, ruinEnter.Payload)
	ruinEnterPayload, ok := ruinEnter.Payload.(WarpRuinEnter)
	require.True(t, ok, "型が WarpRuinEnter であるべき")
	assert.Equal(t, "ancient_ruin", ruinEnterPayload.DefinitionName)

	// ペイロード付きのイベントもフィールドが設定されることを確認
	dialog := ShowDialogEvent("test", ecs.Entity{})
	require.IsType(t, ShowDialog{}, dialog.Payload)
	showDialog, ok := dialog.Payload.(ShowDialog)
	require.True(t, ok, "型が ShowDialog であるべき")
	assert.Equal(t, "test", showDialog.MessageKey)

	storage := OpenStorageEvent(ecs.Entity{})
	assert.IsType(t, OpenStorage{}, storage.Payload)
}
