package components

import "time"

// PlayTime はそのランの累積プレイ実時間を保持するシングルトン。ゲーム内時間 GameTime とは別で、
// wall-clock で遊んだ時間を溜める。セーブ画面のスロットラベルが時:分で表示する。
type PlayTime struct {
	Duration time.Duration
}
