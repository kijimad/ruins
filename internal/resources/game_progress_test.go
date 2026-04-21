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
