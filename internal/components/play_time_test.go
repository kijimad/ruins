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

func TestPlayTime_Startの蓄積にTickが実経過を足す(t *testing.T) {
	t.Parallel()
	t0 := time.Unix(1000, 0)
	pt := &PlayTime{}
	pt.Start(2*time.Hour, t0)
	assert.Equal(t, 2*time.Hour, pt.Elapsed(), "Start は蓄積をセットする")

	pt.Tick(t0.Add(time.Minute))
	assert.Equal(t, 2*time.Hour+time.Minute, pt.Elapsed(), "前回からの経過を足す")
	pt.Tick(t0.Add(3 * time.Minute))
	assert.Equal(t, 2*time.Hour+3*time.Minute, pt.Elapsed(), "累積し続ける")
}

func TestPlayTime_Tick初回は起点だけ置く(t *testing.T) {
	t.Parallel()
	t0 := time.Unix(1000, 0)
	pt := &PlayTime{} // lastTick はゼロ
	pt.Tick(t0)
	assert.Zero(t, pt.Elapsed(), "初回は足さない")
	pt.Tick(t0.Add(time.Minute))
	assert.Equal(t, time.Minute, pt.Elapsed(), "2回目から足す")
}
