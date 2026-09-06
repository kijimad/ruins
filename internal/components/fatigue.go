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
	case ratio < 0.3:
		return FatigueRested
	case ratio < 0.5:
		return FatigueNormal
	case ratio < 0.8:
		return FatigueTired
	default:
		return FatigueExhausted
	}
}

// ConsciousnessPenalty は疲労段階が意識へ与える低下量を返す。意識は master 乗数なので、この1つの値が
// 命中・行動速度・回復すべてへ波及する。保存せず読み取り時に量から導出する。値は実プレイで調整する
func (f *Fatigue) ConsciousnessPenalty() int {
	switch f.GetLevel() {
	case FatigueRested, FatigueNormal:
		return 0
	case FatigueTired:
		return 10
	case FatigueExhausted:
		return 25
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
