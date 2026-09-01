package components

import "github.com/kijimaD/ruins/internal/consts"

// DeathCause は決着時の死因。決着時に RunStats.Cause へ入れ、表示時に翻訳する
type DeathCause string

const (
	// CauseFrozen は低体温で凍死したときの死因
	CauseFrozen DeathCause = "frozen"
	// CauseIllness は病気で死んだときの死因
	CauseIllness DeathCause = "illness"
	// CauseBloodLoss は外傷の失血で死んだときの死因
	CauseBloodLoss DeathCause = "blood_loss"
	// CauseKilled は戦闘のダメージで倒れたときの死因
	CauseKilled DeathCause = "killed"
	// CauseDebug はデバッグで結果画面を確認するための死因
	CauseDebug DeathCause = "debug"
)

// DisplayName は死因の表示名 msgid を返す。結果画面が query.T で訳す。
// 保存値なので未知は素の文字列へ graceful に落とす
func (c DeathCause) DisplayName() string {
	switch c {
	case CauseFrozen:
		return "froze to death"
	case CauseIllness:
		return "died of illness"
	case CauseBloodLoss:
		return "bled out"
	case CauseKilled:
		return "killed in battle"
	case CauseDebug:
		return "debug"
	default:
		return string(c)
	}
}

// RunStats は run を通じて貯める統計を保持するシングルトン。run 中ずっと存在し serde 保存する。
// 撃破・漁り・売上を積み上げ、決着時に死因を記録する。結果画面と道中の統計画面が読む。
// 生存日数と経過ターンは GameTime から引けるので持たない
type RunStats struct {
	EnemiesKilled  int             // 倒した敵の数
	ItemsScavenged int             // 漁ったアイテム数
	SalesTotal     consts.Currency // 売上累計
	Cause          DeathCause      // 死因。決着時に記録する
}
