package messagewindow

import (
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
)

// Choice は選択肢を表す
type Choice struct {
	Text   string
	Action func() error // 選択時の処理
}

// MessageContent はメッセージの内容
type MessageContent struct {
	Choices          []Choice                    // 選択肢システム
	SpeakerName      string                      // 話者名
	TextSegmentLines [][]messagedata.TextSegment // 行ごとの色付きテキストセグメント
}

// Builder はメッセージウィンドウを構築するためのビルダー
type Builder struct {
	world w.World
}

// NewBuilder は新しいBuilderを作成する
func NewBuilder(world w.World) *Builder {
	return &Builder{
		world: world,
	}
}

// Build はMessageDataからウィンドウを構築する
func (b *Builder) Build(initialMessage *messagedata.MessageData) *Window {
	window := &Window{
		config:         DefaultConfig(),
		world:          b.world,
		isOpen:         true,
		queueManager:   NewQueueManager(),
		currentMessage: initialMessage,
	}

	window.updateContentFromMessage(initialMessage)

	// 連鎖メッセージがある場合はキューに追加
	if initialMessage.HasNextMessages() {
		window.queueManager.Enqueue(initialMessage.GetNextMessages()...)
	}

	return window
}
