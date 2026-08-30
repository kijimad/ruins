package components

import "github.com/kijimaD/ruins/internal/consts"

// 死因の識別子。決着時に RunStats.Cause へ入れる。表示時に翻訳する。
// 区別する値が増えるまで enum にせず素の文字列の定数で持つ
const (
	// CauseFrozen は低体温で凍死したときの死因
	CauseFrozen = "frozen"
)

// RunStats は run を通じて貯める統計を保持するシングルトン。run 中ずっと存在し serde 保存する。
// 撃破・漁り・売上を積み上げ、決着時に死因を記録する。結果画面と道中の統計画面が読む。
// 生存日数と経過ターンは GameTime から引けるので持たない
type RunStats struct {
	EnemiesKilled  int             // 倒した敵の数
	ItemsScavenged int             // 漁ったアイテム数
	SalesTotal     consts.Currency // 売上累計
	Cause          string          // 死因。決着時に記録する。区別する値の集合が要るまで素の文字列
}
