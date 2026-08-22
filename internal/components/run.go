package components

import "github.com/kijimaD/ruins/internal/consts"

// DeathCause は死因を表す。結果画面の表示に使う。身体モデルが確定して渡す
type DeathCause string

// RunOutcome は run の決着を保持するシングルトン。いまは死のみ。
// 決着の瞬間に生成し、終端 State が消費する。決着時のみ存在し、セーブは生存中にしか
// 起きないので保存時には居ない。ゆえに既定の丸ごと保存のままでよく serde 除外は要らない
type RunOutcome struct {
	Cause       DeathCause // 死因
	ReachedDist int        // 到達した前進距離。スコアの主軸
	Days        int        // 生存日数
}

// RunStats は run 中に貯める統計を保持するシングルトン。
// RunOutcome と違い run を通じて積み上がるので serde 保存対象にする。
// 加算経路のあるカウンタだけを持つ。負傷や遺跡数は接続時に足す
type RunStats struct {
	EnemiesKilled  int             // 倒した敵の数
	MaxDist        int             // 到達した最大前進距離。スコアの主軸
	ItemsScavenged int             // 漁ったアイテム数
	SalesTotal     consts.Currency // 売上累計
}
