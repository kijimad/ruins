package components

import "github.com/kijimaD/ruins/internal/consts"

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

// FatiguePenalty は疲労段階ごとのペナルティ係数。段階ごとの数値の唯一の置き場所。
// 適用先は回復・行動速度・命中の3出力に分かれ、適用は出力の型が違うので各サイトに残す。
// 係数だけをこの1表に集約して調整しやすくする
type FatiguePenalty struct {
	RecoveryAdd consts.Percent // 回復係数への加算%。Metabolism へ足す
	SpeedAdd    int            // 行動速度への加算
	AccuracyMul consts.Percent // 命中への乗算%
}

// Penalty は疲労段階に対応するペナルティ係数を返す
func (f *Fatigue) Penalty() FatiguePenalty {
	switch f.GetLevel() {
	case FatigueRested, FatigueNormal:
		return FatiguePenalty{RecoveryAdd: 0, SpeedAdd: 0, AccuracyMul: consts.PercentBase}
	case FatigueTired:
		return FatiguePenalty{RecoveryAdd: -20, SpeedAdd: -15, AccuracyMul: 90}
	case FatigueExhausted:
		return FatiguePenalty{RecoveryAdd: -40, SpeedAdd: -35, AccuracyMul: 75}
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
