package components

const (
	// DefaultMaxFatigue はデフォルトの最大疲労。起床中は毎ターン FatigueGainPerTurn 増える
	DefaultMaxFatigue = 2000
	// FatigueGainPerTurn は起床している間に毎ターン蓄積する疲労
	FatigueGainPerTurn = 1
)

// FatigueLevel は疲労の段階を表す
type FatigueLevel string

const (
	// FatigueRested は快調。疲れておらず眠れない。寝すぎを防ぐ
	FatigueRested FatigueLevel = "Rested"
	// FatigueNormal は普通。眠れる
	FatigueNormal FatigueLevel = "Normal"
	// FatigueTired は疲労。回復と行動が鈍り始める
	FatigueTired FatigueLevel = "Tired"
	// FatigueExhausted は過労。重いペナルティを受ける
	FatigueExhausted FatigueLevel = "Exhausted"
)

// Fatigue は蓄積した疲労を表す。起きている間に増え、睡眠でのみ減る。
// Current は Max でクランプし、上限に達しても死なず Exhausted のペナルティが続く
type Fatigue Pool[int]

// GetLevel は現在の疲労段階を返す。しきい値は Current/Max の比率で決める
func (f *Fatigue) GetLevel() FatigueLevel {
	if f.Max <= 0 {
		return FatigueRested
	}

	ratio := float64(f.Current) / float64(f.Max)
	switch {
	case ratio < FatigueRestedRatio:
		return FatigueRested
	case ratio < FatigueTiredRatio:
		return FatigueNormal
	case ratio < FatigueExhaustedRatio:
		return FatigueTired
	default:
		return FatigueExhausted
	}
}

// 疲労レベルを分ける最大疲労に対する割合。バランス導出が単一出典で参照できるよう公開する。
const (
	// FatigueRestedRatio 未満は休息済み
	FatigueRestedRatio = 0.3
	// FatigueTiredRatio 以上で疲労
	FatigueTiredRatio = 0.5
	// FatigueExhaustedRatio 以上で過労
	FatigueExhaustedRatio = 0.8
)

// FatigueSeverity は疲労段階を過労の不調の重症度へ写す。ok=false なら不調は立たない
func (f *Fatigue) FatigueSeverity() (Severity, bool) {
	switch f.GetLevel() {
	case FatigueRested, FatigueNormal:
		return SeverityNone, false
	case FatigueTired:
		return SeverityMinor, true
	case FatigueExhausted:
		return SeverityMedium, true
	}
	panic("invalid FatigueLevel value")
}

// NewFatigue は新しい Fatigue を作成する。初期は疲れていない
func NewFatigue() *Fatigue {
	return &Fatigue{
		Max:     DefaultMaxFatigue,
		Current: 0,
	}
}
