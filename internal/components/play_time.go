package components

import "time"

// PlayTime はそのランの累積プレイ実時間と計測基準を持つシングルトン。serde 非対象で、永続の実体は
// セーブ envelope。ロードで envelope から Total を seed する。SessionStart のゼロ値はラン外。
type PlayTime struct {
	Total        time.Duration
	SessionStart time.Time
}
