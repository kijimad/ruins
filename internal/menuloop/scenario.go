package menuloop

import "github.com/kijimaD/ruins/internal/inputmapper"

// Command はメニュー1操作。本番の ActionID をそのまま再生単位にする
type Command = inputmapper.ActionID

// Scenario は再生するコマンド列。1要素を1フレームで供給する。
// 尽きたフレーム以降はキーボード経路に戻る
type Scenario struct {
	Commands []Command
}

// CommandDriven は再生ドライバがコマンド供給源を差せる state。Screen を持つ state が
// SetCommandSource を Screen へ委譲して実装する。本番ロジックは持たない薄い口で、
// 供給源を差さない本番では従来どおりキーボードで動く。
type CommandDriven interface {
	SetCommandSource(src func() (inputmapper.ActionID, bool))
}

// scenarioSource は Scenario を先頭から1フレーム1件で供給する。Screen.SetCommandSource に next を渡す
type scenarioSource struct {
	rest []Command
}

// newScenarioSource は Scenario から供給源を作る
func newScenarioSource(sc Scenario) *scenarioSource {
	return &scenarioSource{rest: sc.Commands}
}

// next は次のコマンドを1件返す。尽きたら (_, false) を返し、Screen はキーボード経路へ戻る
func (s *scenarioSource) next() (inputmapper.ActionID, bool) {
	if len(s.rest) == 0 {
		return "", false
	}
	a := s.rest[0]
	s.rest = s.rest[1:]
	return a, true
}

// NewScenarioReplay は Scenario を1フレーム1件で吐く供給源関数を返す。
// 再生ドライバが CommandDriven.SetCommandSource へ渡す。テスト専用の生成口。
func NewScenarioReplay(sc Scenario) func() (inputmapper.ActionID, bool) {
	return newScenarioSource(sc).next
}
