package components

import "time"

// PlayTime はそのランのプレイ実時間を測るシングルトン。serde 非対象で、永続の実体はセーブ envelope。
// Total はこのセッション開始前までの蓄積、SessionStartedAt は今のセッションを始めた実時刻。
// プレイ実時間 = Total + time.Since(SessionStartedAt)。SessionStartedAt のゼロ値はラン外。
type PlayTime struct {
	Total            time.Duration
	SessionStartedAt time.Time
}
