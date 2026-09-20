package components_test

import (
	"testing"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestPlayTime_未開始は0(t *testing.T) {
	t.Parallel()
	var pt gc.PlayTime // ゼロ値
	assert.Zero(t, pt.Elapsed(), "Start 前は0")
	var nilPt *gc.PlayTime
	assert.Zero(t, nilPt.Elapsed(), "nil でも0を返す")
}

func TestPlayTime_Startは蓄積に今からの経過を足す(t *testing.T) {
	t.Parallel()
	pt := &gc.PlayTime{}
	pt.Start(2 * time.Hour)
	got := pt.Elapsed()
	assert.GreaterOrEqual(t, got, 2*time.Hour, "蓄積を下回らない")
	assert.Less(t, got, 2*time.Hour+time.Minute, "今セッションの経過はごくわずか")
}
