package components

import "time"

// PlayTime はそのランのプレイ実時間を測るシングルトン。serde 非対象で、永続の実体はセーブ envelope。
// Start でセッションの計測を始め、Elapsed で現在のプレイ実時間を読む。
// total は今セッション開始前までの蓄積、startedAt は今の開始実時刻。
type PlayTime struct {
	total     time.Duration
	startedAt time.Time
}

// Start は prior を蓄積の初期値として、今からセッションの計測を始める。
// 新規開始は Start(0)、ロード復帰は Start(保存済みのプレイ時間)。
func (pt *PlayTime) Start(prior time.Duration) {
	pt.total = prior
	pt.startedAt = time.Now()
}

// Elapsed は現在のプレイ実時間を返す。蓄積に今セッションの経過を足す。未開始のラン外は0。
func (pt *PlayTime) Elapsed() time.Duration {
	if pt == nil || pt.startedAt.IsZero() {
		return 0
	}
	return pt.total + time.Since(pt.startedAt)
}
