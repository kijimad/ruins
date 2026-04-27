package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGameProgress_MarkAndCheck(t *testing.T) {
	t.Parallel()
	gp := NewGameProgress()

	assert.False(t, gp.IsDungeonCleared("廃坑"))

	gp.MarkDungeonCleared("廃坑")
	assert.True(t, gp.IsDungeonCleared("廃坑"))
	assert.False(t, gp.IsDungeonCleared("研究所"))
}

func TestGameProgress_IsAllCleared(t *testing.T) {
	t.Parallel()
	gp := NewGameProgress()
	names := []string{"森", "洞窟", "廃墟"}

	assert.False(t, gp.IsAllCleared(names))

	gp.MarkDungeonCleared("森")
	gp.MarkDungeonCleared("洞窟")
	assert.False(t, gp.IsAllCleared(names))

	gp.MarkDungeonCleared("廃墟")
	assert.True(t, gp.IsAllCleared(names))
}

func TestGameProgress_EventState(t *testing.T) {
	t.Parallel()
	gp := NewGameProgress()

	// 未登録イベントはnilを返す
	assert.Nil(t, gp.GetEvent(EventAllCleared))

	// Activeを設定
	gp.SetEventActive(EventAllCleared)
	ev := gp.GetEvent(EventAllCleared)
	assert.NotNil(t, ev)
	assert.True(t, ev.Active)
	assert.False(t, ev.Seen)

	// Seenを設定
	ev.Seen = true
	ev2 := gp.GetEvent(EventAllCleared)
	assert.True(t, ev2.Seen)
}

func TestGameProgress_IsAllCleared_EmptyList(t *testing.T) {
	t.Parallel()
	gp := NewGameProgress()

	// 空リストは常にtrue
	assert.True(t, gp.IsAllCleared([]string{}))
}
