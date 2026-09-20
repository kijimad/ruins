package components

import "time"

// PlayTime はそのランの累積プレイ実時間を測るシングルトン。serde 非対象で、永続の実体はセーブ envelope。
type PlayTime struct {
	total    time.Duration
	lastTick time.Time
}

// Start は prior を累積の初期値にし、計測の起点を今へ置く。新規は Start(0)、ロードは Start(保存値)。
func (pt *PlayTime) Start(prior time.Duration) {
	pt.total = prior
	pt.lastTick = time.Now()
}

// Tick はラン進行中に毎フレーム呼び、前回からの実経過を累積へ足す。初回は起点だけ置く。
func (pt *PlayTime) Tick() {
	now := time.Now()
	if !pt.lastTick.IsZero() {
		pt.total += now.Sub(pt.lastTick)
	}
	pt.lastTick = now
}

// Elapsed は現在の累積プレイ実時間を返す。nil セーフ。
func (pt *PlayTime) Elapsed() time.Duration {
	if pt == nil {
		return 0
	}
	return pt.total
}
