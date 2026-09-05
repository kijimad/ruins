package components

import "github.com/kijimaD/ruins/internal/consts"

// Sleeping は睡眠中を表すマーカー。SleepBehavior の開始で付き、起床・中断で外れる。
// 空腹進行の抑制と Metabolism の回復ボーナスがこの有無で分岐する
type Sleeping struct {
	Quality consts.Percent // 寝具の睡眠効率。地べたは100
}
