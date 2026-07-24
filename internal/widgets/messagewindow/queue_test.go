package messagewindow

import (
	"testing"

	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQueueManager_初期状態は空(t *testing.T) {
	t.Parallel()

	q := NewQueueManager()

	assert.Equal(t, 0, q.Size())
	assert.False(t, q.HasNext())
	assert.Nil(t, q.Current())
}

func TestQueueManager_Enqueue(t *testing.T) {
	t.Parallel()

	t.Run("末尾に追加される", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		msg1 := messagedata.NewSystemMessage("1")
		msg2 := messagedata.NewSystemMessage("2")

		q.Enqueue(msg1)
		q.Enqueue(msg2)

		assert.Equal(t, 2, q.Size())
		assert.True(t, q.HasNext())
		assert.Same(t, msg1, q.Dequeue())
		assert.Same(t, msg2, q.Dequeue())
	})

	t.Run("複数件を一度に追加できる", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		msg1 := messagedata.NewSystemMessage("1")
		msg2 := messagedata.NewSystemMessage("2")

		q.Enqueue(msg1, msg2)

		assert.Equal(t, 2, q.Size())
	})
}

func TestQueueManager_EnqueueFront(t *testing.T) {
	t.Parallel()

	q := NewQueueManager()
	msg1 := messagedata.NewSystemMessage("後")
	msg2 := messagedata.NewSystemMessage("先")

	q.Enqueue(msg1)
	q.EnqueueFront(msg2)

	require.Equal(t, 2, q.Size())
	assert.Same(t, msg2, q.Dequeue(), "EnqueueFrontで追加した方が先に取り出される")
	assert.Same(t, msg1, q.Dequeue())
}

func TestQueueManager_Dequeue(t *testing.T) {
	t.Parallel()

	t.Run("空のキューはnilを返す", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		assert.Nil(t, q.Dequeue())
	})

	t.Run("取り出したメッセージがCurrentになる", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		msg := messagedata.NewSystemMessage("対象")
		q.Enqueue(msg)

		got := q.Dequeue()

		assert.Same(t, msg, got)
		assert.Same(t, msg, q.Current())
	})

	t.Run("連鎖メッセージを持つ場合は先頭に展開される", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		chained := messagedata.NewSystemMessage("親").
			SystemMessage("子1").
			SystemMessage("子2")
		after := messagedata.NewSystemMessage("後続")

		q.Enqueue(chained, after)

		require.True(t, chained.HasNextMessages())
		got := q.Dequeue()
		assert.Same(t, chained, got)

		// 連鎖メッセージが先頭に追加され、元からキューにあったafterより先に出てくる
		require.Equal(t, 3, q.Size())
		nextMsgs := chained.GetNextMessages()
		assert.Same(t, nextMsgs[0], q.Dequeue())
		assert.Same(t, nextMsgs[1], q.Dequeue())
		assert.Same(t, after, q.Dequeue())
	})
}

func TestQueueManager_Clear(t *testing.T) {
	t.Parallel()

	q := NewQueueManager()
	q.Enqueue(messagedata.NewSystemMessage("1"), messagedata.NewSystemMessage("2"))
	q.Dequeue()

	q.Clear()

	assert.Equal(t, 0, q.Size())
	assert.False(t, q.HasNext())
	assert.Nil(t, q.Current(), "Clear()はCurrentもリセットする")
}
