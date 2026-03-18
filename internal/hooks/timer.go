package hooks

import (
	"time"

	"github.com/kijimaD/ruins/internal/inputmapper"
)

// timerState はタイマーの内部状態
type timerState struct {
	StartedAt time.Time
	Active    bool
}

// UseTimer は再利用可能なタイマー状態管理を提供する
// 戻り値:
//   - expired: タイマーが期限切れかどうか
//   - start: タイマーを開始する関数
//   - reset: タイマーをリセットする関数
func UseTimer(store *Store, key string, duration time.Duration) (expired bool, start func(), reset func()) {
	return UseTimerAt(store, key, duration, time.Now())
}

// UseTimerAt は指定した時刻を基準にタイマー状態を管理する
// テスト用に時刻を注入可能にするためのバージョン
func UseTimerAt(store *Store, key string, duration time.Duration, now time.Time) (expired bool, start func(), reset func()) {
	state := UseState(store, key, timerState{}, func(v timerState, _ inputmapper.ActionID) timerState {
		return v // アクションでは変更しない
	})

	expired = state.Active && !state.StartedAt.IsZero() && now.Sub(state.StartedAt) >= duration

	start = func() {
		store.states[key] = timerState{
			StartedAt: now,
			Active:    true,
		}
	}

	reset = func() {
		store.states[key] = timerState{
			StartedAt: time.Time{},
			Active:    false,
		}
	}

	return
}
