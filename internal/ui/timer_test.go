package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUseTimerAt_InitialState(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expired, _, _ := UseTimerAt(store, "test", 100*time.Millisecond, now)

	assert.False(t, expired, "初期状態では expired は false")
}

func TestUseTimerAt_Start(t *testing.T) {
	t.Parallel()

	store := NewStore()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	expired, start, _ := UseTimerAt(store, "test", 100*time.Millisecond, now)
	assert.False(t, expired, "開始前は expired は false")

	start()

	// 開始直後は expired は false
	expired, _, _ = UseTimerAt(store, "test", 100*time.Millisecond, now)
	assert.False(t, expired, "開始直後は expired は false")
}

func TestUseTimerAt_Expired(t *testing.T) {
	t.Parallel()

	store := NewStore()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, start, _ := UseTimerAt(store, "test", 100*time.Millisecond, startTime)
	start()

	// 100ms 後の時刻をシミュレート
	afterExpiry := startTime.Add(150 * time.Millisecond)
	expired, _, _ := UseTimerAt(store, "test", 100*time.Millisecond, afterExpiry)

	assert.True(t, expired, "期限切れ後は expired は true")
}

func TestUseTimerAt_NotYetExpired(t *testing.T) {
	t.Parallel()

	store := NewStore()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, start, _ := UseTimerAt(store, "test", 100*time.Millisecond, startTime)
	start()

	// 50ms 後の時刻をシミュレート（まだ期限内）
	beforeExpiry := startTime.Add(50 * time.Millisecond)
	expired, _, _ := UseTimerAt(store, "test", 100*time.Millisecond, beforeExpiry)

	assert.False(t, expired, "期限内は expired は false")
}

func TestUseTimerAt_Reset(t *testing.T) {
	t.Parallel()

	store := NewStore()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, start, _ := UseTimerAt(store, "test", 100*time.Millisecond, startTime)
	start()

	// 期限切れの時刻
	afterExpiry := startTime.Add(150 * time.Millisecond)
	expired, _, reset := UseTimerAt(store, "test", 100*time.Millisecond, afterExpiry)
	assert.True(t, expired, "期限切れ後は expired は true")

	// リセット
	reset()

	// リセット後は expired は false
	expired, _, _ = UseTimerAt(store, "test", 100*time.Millisecond, afterExpiry)
	assert.False(t, expired, "リセット後は expired は false")
}

func TestUseTimerAt_MultipleTimers(t *testing.T) {
	t.Parallel()

	store := NewStore()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, start1, _ := UseTimerAt(store, "timer1", 100*time.Millisecond, startTime)
	_, start2, _ := UseTimerAt(store, "timer2", 500*time.Millisecond, startTime)

	start1()
	start2()

	// 200ms 後の時刻をシミュレート
	afterTime := startTime.Add(200 * time.Millisecond)
	expired1, _, _ := UseTimerAt(store, "timer1", 100*time.Millisecond, afterTime)
	expired2, _, _ := UseTimerAt(store, "timer2", 500*time.Millisecond, afterTime)

	assert.True(t, expired1, "timer1 は期限切れ")
	assert.False(t, expired2, "timer2 はまだ期限内")
}

func TestUseTimerAt_ExactDuration(t *testing.T) {
	t.Parallel()

	store := NewStore()
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, start, _ := UseTimerAt(store, "test", 100*time.Millisecond, startTime)
	start()

	// ちょうど100ms後
	exactTime := startTime.Add(100 * time.Millisecond)
	expired, _, _ := UseTimerAt(store, "test", 100*time.Millisecond, exactTime)

	assert.True(t, expired, "ちょうど duration 経過時点で expired は true")
}
