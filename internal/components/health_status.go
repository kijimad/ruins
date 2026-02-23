package components

// 健康状態システム
// 部位別健康状態を管理する
// 低体温、凍傷、食中毒などの状態を部位ごとに追跡し、ステータスへの影響を計算する

// Severity は状態の重症度
type Severity int

// 重症度定数
const (
	SeverityNone   Severity = iota // なし
	SeverityMinor                  // 軽度
	SeverityMedium                 // 中度
	SeveritySevere                 // 重度
)

// String は重症度の表示名を返す
func (s Severity) String() string {
	switch s {
	case SeverityNone:
		return ""
	case SeverityMinor:
		return "軽"
	case SeverityMedium:
		return "中"
	case SeveritySevere:
		return "重"
	default:
		panic("不正なSeverity値")
	}
}

// StatType は影響を受けるステータスの種類
type StatType string

// ステータス種類定数
const (
	StatVitality  StatType = "Vitality"
	StatStrength  StatType = "Strength"
	StatSensation StatType = "Sensation"
	StatDexterity StatType = "Dexterity"
	StatAgility   StatType = "Agility"
	StatDefense   StatType = "Defense"
)

// String はステータス種類の表示名を返す
func (st StatType) String() string {
	switch st {
	case StatVitality:
		return "体力"
	case StatStrength:
		return "筋力"
	case StatSensation:
		return "感覚"
	case StatDexterity:
		return "器用"
	case StatAgility:
		return "敏捷"
	case StatDefense:
		return "防御"
	default:
		return string(st)
	}
}

// StatEffect はステータスへの1つの影響
type StatEffect struct {
	Stat  StatType // 影響を受けるステータス
	Value int      // 修正値（固定値の場合）
}

// ConditionType は状態の種類を識別する
type ConditionType string

// 状態種類定数
const (
	ConditionHypothermia  ConditionType = "Hypothermia"  // 低体温
	ConditionHyperthermia ConditionType = "Hyperthermia" // 高体温
	ConditionFrostbite    ConditionType = "Frostbite"    // 凍傷
)

// ConditionTypeDisplayName は状態種類の表示名を返す
func ConditionTypeDisplayName(ct ConditionType) string {
	switch ct {
	case ConditionHypothermia:
		return "低体温"
	case ConditionHyperthermia:
		return "高体温"
	case ConditionFrostbite:
		return "凍傷"
	default:
		return string(ct)
	}
}

// HealthCondition は部位に付与される1つの状態
type HealthCondition struct {
	Type     ConditionType // 状態の種類
	Severity Severity      // 重症度
	Effects  []StatEffect  // この状態による影響
}

// DisplayName は状態の表示名を返す
func (hc *HealthCondition) DisplayName() string {
	name := ConditionTypeDisplayName(hc.Type)
	if hc.Severity != SeverityNone {
		name += "(" + hc.Severity.String() + ")"
	}
	return name
}

// BodyPartHealth は1つの部位の健康状態
type BodyPartHealth struct {
	Conditions []HealthCondition
}

// SetCondition は状態を設定する。既存の同種の状態は上書きする
func (bph *BodyPartHealth) SetCondition(cond HealthCondition) {
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == cond.Type {
			bph.Conditions[i] = cond
			return
		}
	}
	bph.Conditions = append(bph.Conditions, cond)
}

// RemoveCondition は状態を削除する
func (bph *BodyPartHealth) RemoveCondition(condType ConditionType) {
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == condType {
			bph.Conditions = append(bph.Conditions[:i], bph.Conditions[i+1:]...)
			return
		}
	}
}

// HealthStatus は部位ごとの健康状態を管理するコンポーネント
type HealthStatus struct {
	Parts [BodyPartCount]BodyPartHealth

	// 前回のステータス修正値のキャッシュ。変化検知に使用する
	prevModifiers map[StatType]int
}

// GetStatModifier は指定したステータスへの合計修正値を返す
func (hs *HealthStatus) GetStatModifier(stat StatType) int {
	total := 0
	for _, partHealth := range hs.Parts {
		for _, cond := range partHealth.Conditions {
			for _, effect := range cond.Effects {
				if effect.Stat == stat {
					total += effect.Value
				}
			}
		}
	}
	return total
}

// HasModifierChanged は前回からステータス修正値が変化したかを判定し、キャッシュを更新する
// TODO: もうちょっといい感じに差分判定できないか?
func (hs *HealthStatus) HasModifierChanged() bool {
	// 現在の修正値を計算
	current := map[StatType]int{
		StatVitality:  hs.GetStatModifier(StatVitality),
		StatStrength:  hs.GetStatModifier(StatStrength),
		StatSensation: hs.GetStatModifier(StatSensation),
		StatDexterity: hs.GetStatModifier(StatDexterity),
		StatAgility:   hs.GetStatModifier(StatAgility),
		StatDefense:   hs.GetStatModifier(StatDefense),
	}

	// 初回または変化があればtrue
	changed := false
	if hs.prevModifiers == nil {
		changed = true
	} else {
		for stat, val := range current {
			if hs.prevModifiers[stat] != val {
				changed = true
				break
			}
		}
	}

	// キャッシュを更新
	hs.prevModifiers = current
	return changed
}
