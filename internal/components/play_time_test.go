package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPlayTime_未開始とnilは0(t *testing.T) {
	t.Parallel()
	var pt PlayTime
	assert.Zero(t, pt.Elapsed(), "Start 前は0")
	var nilPt *PlayTime
	assert.Zero(t, nilPt.Elapsed(), "nil でも0")
}

func TestPlayTime_Startは累積をセットしTickが実経過を足す(t *testing.T) {
	t.Parallel()
	pt := &PlayTime{}
	pt.Start(2 * time.Hour)
	assert.Equal(t, 2*time.Hour, pt.Elapsed(), "Start は累積をセットする")

	pt.lastTick = time.Now().Add(-time.Minute) // 1分前を起点にする
	pt.Tick()
	assert.Greater(t, pt.Elapsed(), 2*time.Hour+50*time.Second, "Tick が前回からの実経過を足す")
	assert.Less(t, pt.Elapsed(), 2*time.Hour+70*time.Second)
}
