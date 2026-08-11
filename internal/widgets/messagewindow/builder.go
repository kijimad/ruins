package messagewindow

import (
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
)

// choiceOption は選択肢を表す
type choiceOption struct {
	Text   string
	Action func() error // 選択時の処理
}

// messageContent はメッセージの内容
type messageContent struct {
	Choices          []choiceOption              // 選択肢システム
	SpeakerName      string                      // 話者名
	TextSegmentLines [][]messagedata.TextSegment // 行ごとの色付きテキストセグメント
}

// NewWindow はMessageDataからメッセージウィンドウを構築する
func NewWindow(world w.World, initialMessage *messagedata.MessageData) *Window {
	window := &Window{
		config:         defaultWindowConfig(),
		world:          world,
		isOpen:         true,
		queueManager:   newQueueManager(),
		currentMessage: initialMessage,
	}

	window.updateContentFromMessage(initialMessage)

	// 連鎖メッセージがある場合はキューに追加
	if initialMessage.HasNextMessages() {
		window.queueManager.Enqueue(initialMessage.GetNextMessages()...)
	}

	return window
}
