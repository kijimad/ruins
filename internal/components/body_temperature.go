package components

// 部位別体温システム
// 収束温度モデルを採用し、6部位の体温を追跡する

// BodyPart は体の部位を表す
type BodyPart int

// 体の部位定数
const (
	BodyPartTorso BodyPart = iota // 胴体
	BodyPartHead                  // 頭
	BodyPartArms                  // 腕
	BodyPartHands                 // 手
	BodyPartLegs                  // 脚
	BodyPartFeet                  // 足
	BodyPartCount                 // 部位数 = 6
)

// String は部位名を返す
func (bp BodyPart) String() string {
	switch bp {
	case BodyPartTorso:
		return "胴体"
	case BodyPartHead:
		return "頭"
	case BodyPartArms:
		return "腕"
	case BodyPartHands:
		return "手"
	case BodyPartLegs:
		return "脚"
	case BodyPartFeet:
		return "足"
	default:
		return unknownString
	}
}

// 温度定数
// 内部値は0-100スケールで、50が正常体温を表す
const (
	TempFreezing  = 10 // 凍結状態の閾値
	TempVeryCold  = 25 // 非常に寒い状態の閾値
	TempCold      = 40 // 寒い状態の閾値
	TempNormal    = 50 // 正常体温
	TempHot       = 60 // 暑い状態の閾値
	TempVeryHot   = 75 // 非常に暑い状態の閾値
	TempScorching = 90 // 灼熱状態の閾値
)

// TempLevel は体温レベルを表す
type TempLevel int

// 体温レベル定数
const (
	TempLevelFreezing  TempLevel = iota // 凍結 (0-10)
	TempLevelVeryCold                   // 非常に寒い (11-25)
	TempLevelCold                       // 寒い (26-40)
	TempLevelNormal                     // 正常 (41-60)
	TempLevelHot                        // 暑い (61-75)
	TempLevelVeryHot                    // 非常に暑い (76-90)
	TempLevelScorching                  // 灼熱 (91-100)
)

// String は温度レベル名を返す
func (tl TempLevel) String() string {
	switch tl {
	case TempLevelFreezing:
		return "凍結"
	case TempLevelVeryCold:
		return "非常に寒い"
	case TempLevelCold:
		return "寒い"
	case TempLevelNormal:
		return "正常"
	case TempLevelHot:
		return "暑い"
	case TempLevelVeryHot:
		return "非常に暑い"
	case TempLevelScorching:
		return "灼熱"
	default:
		return unknownString
	}
}

// BodyPartState は1部位の温度状態を管理する
type BodyPartState struct {
	Temp           int  // 現在体温（0-100、50が正常）
	Convergent     int  // 収束先体温
	FrostbiteTimer int  // 凍傷タイマー（0-100、100で凍傷発症）
	HasFrostbite   bool // 凍傷フラグ
}

// BodyTemperature は部位ごとの体温を管理するコンポーネント
type BodyTemperature struct {
	Parts [BodyPartCount]BodyPartState
}

// NewBodyTemperature は正常体温で初期化された BodyTemperature を作成する
func NewBodyTemperature() *BodyTemperature {
	bt := &BodyTemperature{}
	for i := 0; i < int(BodyPartCount); i++ {
		bt.Parts[i] = BodyPartState{
			Temp:       TempNormal,
			Convergent: TempNormal,
		}
	}
	return bt
}

// GetLevel は指定部位の体温レベルを返す
func (bt *BodyTemperature) GetLevel(part BodyPart) TempLevel {
	temp := bt.Parts[part].Temp
	switch {
	case temp <= TempFreezing:
		return TempLevelFreezing
	case temp <= TempVeryCold:
		return TempLevelVeryCold
	case temp <= TempCold:
		return TempLevelCold
	case temp <= TempHot:
		return TempLevelNormal
	case temp <= TempVeryHot:
		return TempLevelHot
	case temp <= TempScorching:
		return TempLevelVeryHot
	default:
		return TempLevelScorching
	}
}

// GetPenalty は指定部位の体温によるステータスペナルティを返す
func (bt *BodyTemperature) GetPenalty(part BodyPart) int {
	level := bt.GetLevel(part)

	switch level {
	case TempLevelFreezing:
		return -3
	case TempLevelVeryCold:
		return -2
	case TempLevelCold:
		return -1
	case TempLevelNormal:
		return 0
	case TempLevelHot:
		return -1
	case TempLevelVeryHot:
		return -2
	case TempLevelScorching:
		return -3
	default:
		return 0
	}
}

// GetWorstConvergentLevel は全部位の収束温度から最悪のレベルを返す
// 正常から離れているほど悪い
func (bt *BodyTemperature) GetWorstConvergentLevel() TempLevel {
	worst := TempLevelNormal
	worstDistance := 0

	for i := 0; i < int(BodyPartCount); i++ {
		temp := bt.Parts[i].Convergent
		var level TempLevel
		switch {
		case temp <= TempFreezing:
			level = TempLevelFreezing
		case temp <= TempVeryCold:
			level = TempLevelVeryCold
		case temp <= TempCold:
			level = TempLevelCold
		case temp <= TempHot:
			level = TempLevelNormal
		case temp <= TempVeryHot:
			level = TempLevelHot
		case temp <= TempScorching:
			level = TempLevelVeryHot
		default:
			level = TempLevelScorching
		}

		// 正常からの距離を計算
		distance := 0
		switch level {
		case TempLevelFreezing, TempLevelScorching:
			distance = 3
		case TempLevelVeryCold, TempLevelVeryHot:
			distance = 2
		case TempLevelCold, TempLevelHot:
			distance = 1
		case TempLevelNormal:
			distance = 0
		}

		if distance > worstDistance {
			worstDistance = distance
			worst = level
		}
	}

	return worst
}

// IsExtremity は末端部位かどうかを返す
// 凍傷は末端部位のみで発症する
func IsExtremity(part BodyPart) bool {
	return part == BodyPartHands || part == BodyPartFeet
}

// ClampTemp は体温を有効範囲(0-100)に収める
func ClampTemp(temp int) int {
	return max(0, min(100, temp))
}
