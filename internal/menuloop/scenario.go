package menuloop

import "github.com/kijimaD/ruins/internal/inputmapper"

// Scenario は再生するコマンド列。本番の ActionID をそのまま再生単位にし、1要素を1フレームで
// 供給する。尽きたフレーム以降はキーボード経路に戻る
type Scenario struct {
	Commands []inputmapper.ActionID
}

// CommandDriven は再生ドライバがコマンド供給源を差せる state。Screen を持つ state が
// SetCommandSource を Screen へ委譲して実装する。本番ロジックは持たない薄い口で、
// 供給源を差さない本番では従来どおりキーボードで動く。
type CommandDriven interface {
	SetCommandSource(src func() (inputmapper.ActionID, bool))
}

// NewScenarioReplay は Scenario を1フレーム1件で吐く供給源関数を返す。返す関数は残りの
// コマンドをクロージャに閉じ込め、呼ぶたび先頭から消費する。尽きたら (_, false) を返し、
// Screen はキーボード経路へ戻る。再生ドライバが CommandDriven.SetCommandSource へ渡す。
func NewScenarioReplay(sc Scenario) func() (inputmapper.ActionID, bool) {
	rest := sc.Commands
	return func() (inputmapper.ActionID, bool) {
		if len(rest) == 0 {
			return "", false
		}
		a := rest[0]
		rest = rest[1:]
		return a, true
	}
}
