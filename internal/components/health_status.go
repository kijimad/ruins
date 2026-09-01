package components

import "github.com/kijimaD/ruins/internal/consts"

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
		return "Minor"
	case SeverityMedium:
		return "Medium"
	case SeveritySevere:
		return "Severe"
	default:
		panic("invalid Severity value")
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
		return "Vitality"
	case StatStrength:
		return "Strength"
	case StatSensation:
		return "Sensation"
	case StatDexterity:
		return "Dexterity"
	case StatAgility:
		return "Agility"
	case StatDefense:
		return "Defense"
	default:
		return string(st)
	}
}

// StatEffect はステータスへの1つの影響
type StatEffect struct {
	Stat  StatType // 影響を受けるステータス
	Value int      // 修正値
}

// ConditionType は状態の種類を識別する
type ConditionType string

// 状態種類定数
const (
	ConditionHypothermia  ConditionType = "Hypothermia"  // 低体温
	ConditionFracture     ConditionType = "Fracture"     // 骨折
	ConditionLaceration   ConditionType = "Laceration"   // 切り傷
	ConditionLiverIllness ConditionType = "LiverIllness" // 肝疾患
)

// conditionMeta は状態種類ごとの静的な情報。表示と身体への反応率を1つの表に畳み、
// 症状の全属性を横1行で見渡せるようにする。反応率は症状ごとに異なる
type conditionMeta struct {
	// displayName は表示名 msgid
	displayName string
	// description は概要説明 msgid。何であるかと未治療の振る舞いを1文で伝える
	description string
	// painPerSeverity は重症度1段あたりに加える痛み
	painPerSeverity int
	// capacityDropPerSeverity は重症度1段あたりに下げる該当機能の量
	capacityDropPerSeverity int
}

// conditionMetas は状態種類ごとの情報。状態を足すときはここへ1行足す。
// 反応率は実プレイで調整する。骨折は激痛で機能を大きく損ない、切り傷は軽く、
// 肝疾患は痛みが軽く全身性でHP消耗が主
var conditionMetas = map[ConditionType]conditionMeta{
	ConditionHypothermia: {
		displayName:     "Hypothermia",
		description:     "The body is dangerously cold. Warm up to recover.",
		painPerSeverity: 6, capacityDropPerSeverity: 20,
	},
	ConditionFracture: {
		displayName:     "Fracture",
		description:     "A broken bone. It will not heal until treated.",
		painPerSeverity: 18, capacityDropPerSeverity: 20,
	},
	ConditionLaceration: {
		displayName:     "Laceration",
		description:     "An open wound. It will not heal until treated.",
		painPerSeverity: 8, capacityDropPerSeverity: 8,
	},
	ConditionLiverIllness: {
		displayName:     "Liver illness",
		description:     "It worsens while untreated and drains HP when severe.",
		painPerSeverity: 4, capacityDropPerSeverity: 10,
	},
}

// ConditionTypeDisplayName は状態種類の表示名を返す。未登録なら素のIDを返す
func ConditionTypeDisplayName(ct ConditionType) string {
	if m, ok := conditionMetas[ct]; ok {
		return m.displayName
	}
	return string(ct)
}

// ConditionTypeDescription は状態種類の概要説明を返す。未登録なら空文字を返す
func ConditionTypeDescription(ct ConditionType) string {
	return conditionMetas[ct].description
}

// BodyCapacities は身体機能の一式。すべて基準 100 の consts.Percent で、100 が正常、
// 低いほど機能が落ちる。痛み Pain だけは 0 が無痛で、大きいほど痛い。
// 不調から読み取り時に導出し、保存はしない
type BodyCapacities struct {
	Pain          consts.Percent // 0 が無痛。大きいほど痛い
	Consciousness consts.Percent // 100 が正常。痛みと全身性の不調で下がる。全機能に掛かる乗数
	Manipulation  consts.Percent // 腕・手の不調で下がる
	Moving        consts.Percent // 脚・足の不調で下がる
	Sight         consts.Percent // 頭の不調で下がる
}

// painConsciousnessDivisor は痛みが意識を下げる割合。痛みをこれで割ったぶん意識が下がる。
// 症状に依らない集約側の係数。値は実プレイで調整する
const painConsciousnessDivisor = 2

// CapacityKind は部位が下げる身体機能の区分
type CapacityKind int

// 身体機能の区分
const (
	CapacityConsciousness CapacityKind = iota // 胴・全身の全身性
	CapacityManipulation                      // 腕・手
	CapacityMoving                            // 脚・足
	CapacitySight                             // 頭
)

// String は身体機能の表示名 msgid を返す。UI は query.T でこれを訳す
func (c CapacityKind) String() string {
	switch c {
	case CapacityConsciousness:
		return "Consciousness"
	case CapacityManipulation:
		return "Manipulation"
	case CapacityMoving:
		return "Moving"
	case CapacitySight:
		return "Sight"
	default:
		panic("invalid CapacityKind value")
	}
}

// bodyPartCapacity は部位が下げる身体機能を返す。部位階層を持たず固定表で対応づける
func bodyPartCapacity(part BodyPart) CapacityKind {
	switch part {
	case BodyPartHead:
		return CapacitySight
	case BodyPartArms, BodyPartHands:
		return CapacityManipulation
	case BodyPartLegs, BodyPartFeet:
		return CapacityMoving
	default:
		return CapacityConsciousness
	}
}

// HealthyCapacities は不調が無いときの身体機能を返す。全機能100で痛み0。
// HealthStatus を持たない対象の既定として使う
func HealthyCapacities() BodyCapacities {
	return (&HealthStatus{}).Capacities()
}

// ConditionCapacityImpact は不調1件が身体機能へ与える影響を表示用に返す。
// pain は加える痛み、capacity は下げる機能の区分、drop はその低下量。
// capacity は部位で定まり重症度に依らない。drop が0なら影響なしで、健康タブの症状詳細は
// drop>0 のときだけ「操作 -Y」を出す
func ConditionCapacityImpact(ct ConditionType, part BodyPart, sev Severity) (pain int, capacity CapacityKind, drop int) {
	capacity = bodyPartCapacity(part)
	pain, drop = conditionSeverityImpact(ct, sev)
	return pain, capacity, drop
}

// conditionSeverityImpact は不調1件の痛みと機能低下を返す。症状ごとの反応率に重症度を掛ける。
// 重症度なしや未登録の症状は0を返す。Capacities と ConditionCapacityImpact が共用する
func conditionSeverityImpact(ct ConditionType, sev Severity) (pain, drop int) {
	m := conditionSeverityMultiplier(sev)
	if m == 0 {
		return 0, 0
	}
	meta := conditionMetas[ct]
	return meta.painPerSeverity * m, meta.capacityDropPerSeverity * m
}

// conditionSeverityMultiplier は重症度から効果倍率を返す
func conditionSeverityMultiplier(sev Severity) int {
	switch sev {
	case SeveritySevere:
		return 3
	case SeverityMedium:
		return 2
	case SeverityMinor:
		return 1
	default:
		return 0
	}
}

// HealthCondition は部位に付与される1つの状態
type HealthCondition struct {
	Type     ConditionType // 状態の種類
	Severity Severity      // 重症度
	Timer    float64       // 進行度タイマー (0-100)
	// TendQuality は治療の質。0 なら未治療、正なら治療済みで 100 が標準、150 なら回復1.5倍。
	// 適した薬の Potency が入り、回復速度を左右する
	TendQuality consts.Percent
}

// DisplayName は状態の表示名を返す
func (hc *HealthCondition) DisplayName() string {
	name := ConditionTypeDisplayName(hc.Type)
	if hc.Severity != SeverityNone {
		// 丸括弧を避け、重症度は空白で続ける
		name += " " + hc.Severity.String()
	}
	return name
}

// UpdateTimer はタイマーを更新し、Severityを再計算する
// delta が正なら悪化、負なら回復
// 戻り値: (前のSeverity, 新しいSeverity)
func (hc *HealthCondition) UpdateTimer(delta float64) (Severity, Severity) {
	prevSeverity := hc.Severity
	hc.Timer = clamp(hc.Timer+delta, 0, 100)
	hc.Severity = TimerToSeverity(hc.Timer)
	return prevSeverity, hc.Severity
}

// IsActive はこの状態が発症しているかを返す
func (hc *HealthCondition) IsActive() bool {
	return hc.Timer >= 25
}

// TimerToSeverity はタイマー値からSeverityを導出する
func TimerToSeverity(timer float64) Severity {
	switch {
	case timer < 25:
		return SeverityNone
	case timer < 50:
		return SeverityMinor
	case timer < 75:
		return SeverityMedium
	default:
		return SeveritySevere
	}
}

// clamp は値を範囲内に収める
func clamp[T ~int | ~float64](val, minVal, maxVal T) T {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
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

// GetCondition は指定した種類の状態を取得する。存在しない場合はnil
func (bph *BodyPartHealth) GetCondition(condType ConditionType) *HealthCondition {
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == condType {
			return &bph.Conditions[i]
		}
	}
	return nil
}

// GetOrCreateCondition は指定した種類の状態を取得または作成する
func (bph *BodyPartHealth) GetOrCreateCondition(condType ConditionType) *HealthCondition {
	if cond := bph.GetCondition(condType); cond != nil {
		return cond
	}
	bph.Conditions = append(bph.Conditions, HealthCondition{
		Type:     condType,
		Severity: SeverityNone,
		Timer:    0,
	})
	return &bph.Conditions[len(bph.Conditions)-1]
}

// SeverityChange はSeverityの変化を表す
type SeverityChange struct {
	CondType ConditionType
	Prev     Severity
	Current  Severity
}

// UpdateConditionTimer は指定した状態のタイマーを更新する
// 状態が存在しない場合は作成する
// タイマーが0になった状態は削除する
// 戻り値: Severityの変化情報
func (bph *BodyPartHealth) UpdateConditionTimer(condType ConditionType, delta float64) SeverityChange {
	cond := bph.GetOrCreateCondition(condType)
	prev, current := cond.UpdateTimer(delta)

	// タイマーが0になったら状態を削除
	if cond.Timer == 0 {
		bph.RemoveCondition(condType)
	}

	return SeverityChange{CondType: condType, Prev: prev, Current: current}
}

// HealthStatus は部位ごとの健康状態を管理するコンポーネント
type HealthStatus struct {
	Parts [BodyPartCount]BodyPartHealth
	// BodyTempOffset は平熱からの体温のずれ。摂氏。0 が平熱かつ上限で、寒さで負へ動く
	BodyTempOffset float64
}

// Capacities は不調から身体機能の一式を導出する。保存済みの値でなく Timer と Severity から計算する。
// 部位ごとの不調が対応機能を下げ、痛みと全身性の不調が意識を下げ、意識が全機能へ乗算される
func (hs *HealthStatus) Capacities() BodyCapacities {
	pain := 0
	var manip, moving, sight, systemic int // 各機能の低下量
	for i := range hs.Parts {
		for _, cond := range hs.Parts[i].Conditions {
			p, drop := conditionSeverityImpact(cond.Type, cond.Severity)
			pain += p
			switch bodyPartCapacity(BodyPart(i)) {
			case CapacityManipulation:
				manip += drop
			case CapacityMoving:
				moving += drop
			case CapacitySight:
				sight += drop
			case CapacityConsciousness:
				systemic += drop
			}
		}
	}

	pain = clamp(pain, 0, 100)
	// 意識は全身性の低下と痛みで下がる
	consciousness := clamp(100-systemic-pain/painConsciousnessDivisor, 0, 100)
	// 局所機能は低下を引いたうえで、意識を全体乗数として掛ける
	withConsciousness := func(local int) consts.Percent {
		return consts.Percent(clamp(local, 0, 100) * consciousness / 100)
	}

	return BodyCapacities{
		Pain:          consts.Percent(pain),
		Consciousness: consts.Percent(consciousness),
		Manipulation:  withConsciousness(100 - manip),
		Moving:        withConsciousness(100 - moving),
		Sight:         withConsciousness(100 - sight),
	}
}
