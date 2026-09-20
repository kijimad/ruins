package components

import "time"

// PlayTime はそのランのプレイ実時間を測るシングルトン。serde 非対象で、永続の実体はセーブ envelope。
// SessionStart は累積が0になる仮想の基準時刻で、プレイ実時間 = time.Since(SessionStart)。
// 新規開始は now、ロードは now から保存値を引いた点に置く。ゼロ値はラン外。
type PlayTime struct {
	SessionStart time.Time
}
