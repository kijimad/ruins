package messagewindow

import (
	"github.com/kijimaD/ruins/internal/messagedata"
)

// QueueManager はメッセージキューを管理する
type QueueManager struct {
	queue   []*messagedata.MessageData
	current *messagedata.MessageData
}

// NewQueueManager は新しいQueueManagerを作成する
func NewQueueManager() *QueueManager {
	return &QueueManager{
		queue: make([]*messagedata.MessageData, 0),
	}
}

// Enqueue はメッセージをキューに追加する
func (q *QueueManager) Enqueue(messages ...*messagedata.MessageData) {
	q.queue = append(q.queue, messages...)
}

// EnqueueFront はメッセージをキューの先頭に追加する
func (q *QueueManager) EnqueueFront(messages ...*messagedata.MessageData) {
	q.queue = append(messages, q.queue...)
}

// Dequeue は次のメッセージを取り出す
func (q *QueueManager) Dequeue() *messagedata.MessageData {
	if len(q.queue) == 0 {
		return nil
	}

	msg := q.queue[0]
	q.queue = q.queue[1:]
	q.current = msg

	// メッセージが連鎖メッセージを持つ場合、それらを先頭に追加
	if msg.HasNextMessages() {
		q.EnqueueFront(msg.GetNextMessages()...)
	}

	return msg
}

// HasNext は次のメッセージがあるかを確認
func (q *QueueManager) HasNext() bool {
	return len(q.queue) > 0
}

// Current は現在のメッセージを取得
func (q *QueueManager) Current() *messagedata.MessageData {
	return q.current
}

// Clear はキューをクリアする
func (q *QueueManager) Clear() {
	q.queue = q.queue[:0]
	q.current = nil
}

// Size はキューのサイズを返す
func (q *QueueManager) Size() int {
	return len(q.queue)
}
