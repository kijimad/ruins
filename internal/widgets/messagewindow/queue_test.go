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

	t.Run("メッセージを末尾に追加する", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		msg1 := messagedata.NewSystemMessage("1件目")
		msg2 := messagedata.NewSystemMessage("2件目")

		q.Enqueue(msg1)
		q.Enqueue(msg2)

		assert.Equal(t, 2, q.Size())
		assert.True(t, q.HasNext())
		assert.Equal(t, msg1, q.Dequeue())
		assert.Equal(t, msg2, q.Dequeue())
	})

	t.Run("複数メッセージをまとめて追加する", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		msg1 := messagedata.NewSystemMessage("1件目")
		msg2 := messagedata.NewSystemMessage("2件目")

		q.Enqueue(msg1, msg2)

		assert.Equal(t, 2, q.Size())
	})
}

func TestQueueManager_EnqueueFront(t *testing.T) {
	t.Parallel()

	q := NewQueueManager()
	msg1 := messagedata.NewSystemMessage("後発")
	msg2 := messagedata.NewSystemMessage("先発")

	q.Enqueue(msg1)
	q.EnqueueFront(msg2)

	assert.Equal(t, 2, q.Size())
	assert.Equal(t, msg2, q.Dequeue())
	assert.Equal(t, msg1, q.Dequeue())
}

func TestQueueManager_Dequeue(t *testing.T) {
	t.Parallel()

	t.Run("空のキューはnilを返しCurrentも変化しない", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()

		assert.Nil(t, q.Dequeue())
		assert.Nil(t, q.Current())
	})

	t.Run("取り出したメッセージがCurrentになりキューから除かれる", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		msg := messagedata.NewSystemMessage("対象")
		q.Enqueue(msg)

		got := q.Dequeue()

		assert.Equal(t, msg, got)
		assert.Equal(t, msg, q.Current())
		assert.Equal(t, 0, q.Size())
	})

	t.Run("連鎖メッセージを持つ場合は次のメッセージがキュー先頭に展開される", func(t *testing.T) {
		t.Parallel()

		q := NewQueueManager()
		chained := messagedata.NewSystemMessage("開始").
			SystemMessage("連鎖1").
			SystemMessage("連鎖2")
		after := messagedata.NewSystemMessage("後続")
		q.Enqueue(chained, after)

		require.Equal(t, chained, q.Dequeue())
		require.Equal(t, 3, q.Size())

		next1 := q.Dequeue()
		next2 := q.Dequeue()
		last := q.Dequeue()

		assert.Equal(t, chained.GetNextMessages()[0], next1)
		assert.Equal(t, chained.GetNextMessages()[1], next2)
		assert.Equal(t, after, last)
	})
}

func TestQueueManager_HasNext(t *testing.T) {
	t.Parallel()

	q := NewQueueManager()
	assert.False(t, q.HasNext())

	q.Enqueue(messagedata.NewSystemMessage("メッセージ"))
	assert.True(t, q.HasNext())

	q.Dequeue()
	assert.False(t, q.HasNext())
}

func TestQueueManager_Clear(t *testing.T) {
	t.Parallel()

	q := NewQueueManager()
	q.Enqueue(messagedata.NewSystemMessage("1件目"), messagedata.NewSystemMessage("2件目"))
	q.Dequeue()
	require.NotNil(t, q.Current())

	q.Clear()

	assert.Equal(t, 0, q.Size())
	assert.False(t, q.HasNext())
	assert.Nil(t, q.Current())
}
