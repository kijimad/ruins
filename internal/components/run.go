package components

import "github.com/kijimaD/ruins/internal/consts"

// RunStats は run を通じて貯める統計を保持するシングルトン。run 中ずっと存在し serde 保存する。
// 撃破・漁り・売上を積み上げ、決着時に死因を記録する。結果画面と道中の統計画面が読む。
// 生存日数と経過ターンは GameTime から引けるので持たない
type RunStats struct {
	EnemiesKilled  int             // 倒した敵の数
	ItemsScavenged int             // 漁ったアイテム数
	SalesTotal     consts.Currency // 売上累計
	Cause          string          // 死因。決着時に記録する。区別する値の集合が要るまで素の文字列
}
