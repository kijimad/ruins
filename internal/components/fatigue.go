package components

import "github.com/kijimaD/ruins/internal/oapi"

const (
	// DefaultMaxFatigue はデフォルトの最大疲労。起床中は毎ターン FatigueGainPerTurn 増える
	DefaultMaxFatigue = 2000
	// FatigueGainPerTurn は起床している間に毎ターン蓄積する疲労
	FatigueGainPerTurn = 1
)

// Fatigue は蓄積した疲労を表す。起きている間に増え、睡眠でのみ減る。
// Current は Max でクランプし、上限に達しても死なず Exhausted のペナルティが続く
type Fatigue Pool[int]

// GetLevel は現在の疲労段階を返す。段階の境目は defs が持ち、Current/Max の比率で決める
func (f *Fatigue) GetLevel(defs NeedDefs) oapi.FatigueStageId {
	if f.Max <= 0 {
		return oapi.FatigueStageIdRested
	}
	return defs.fatigueStage(float64(f.Current) / float64(f.Max)).Level
}

// FatigueSeverity は疲労段階を過労の不調の重症度へ写す。ok=false なら不調は立たない
func (f *Fatigue) FatigueSeverity(defs NeedDefs) (Severity, bool) {
	if f.Max <= 0 {
		return SeverityNone, false
	}
	s := defs.fatigueStage(float64(f.Current) / float64(f.Max))
	return s.Condition, s.Condition != SeverityNone
}

// NewFatigue は新しい Fatigue を作成する。初期は疲れていない
func NewFatigue() *Fatigue {
	return &Fatigue{
		Max:     DefaultMaxFatigue,
		Current: 0,
	}
}
