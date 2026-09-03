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
	ConditionHypothermia   ConditionType = "Hypothermia"   // 低体温
	ConditionFracture      ConditionType = "Fracture"      // 骨折
	ConditionLaceration    ConditionType = "Laceration"    // 切り傷
	ConditionLiverIllness  ConditionType = "LiverIllness"  // 肝疾患
	ConditionFoodPoisoning ConditionType = "FoodPoisoning" // 食中毒
)

// RecoveryMode は不調が未治療のときどう振る舞い、治療でどう治るかを表す。
// 低体温は TemperatureSystem が体温から進めるのでこの区分を持たない
type RecoveryMode string

const (
	// RecoverAfterTend は治療して初めて回復する。未治療は保持し悪化しない。骨折・切り傷
	RecoverAfterTend RecoveryMode = "recover_after_tend"
	// ProgressUntilTend は未治療なら悪化し続け、治療して初めて回復軌道へ乗る。病気
	ProgressUntilTend RecoveryMode = "progress_until_tend"
	// RecoverOverTime は未治療でも時間で自然に治る。治療すれば回復が速まる。食中毒のような自己限定の不調
	RecoverOverTime RecoveryMode = "recover_over_time"
)

// ConditionDef は状態種類ごとの静的な定義。表示・capacity への反応率・回復の振る舞いを1つの表に持つ。
// 症状を調整するときはここ1箇所を触る。Recovery を持つ症状は ConditionSystem が進行と回復を扱い、
// Recovery を持たない低体温は TemperatureSystem 管轄になる。反応率と速度は実プレイで調整する
type ConditionDef struct {
	displayName             string       // 表示名 msgid
	description             string       // 概要説明 msgid
	painPerSeverity         int          // 重症度1段あたりに加える痛み
	capacityDropPerSeverity int          // 重症度1段あたりに下げる該当機能の量
	Recovery                RecoveryMode // 未治療の振る舞いと治し方。空なら ConditionSystem 管轄外
	WorsenPer               int          // ProgressUntilTend で未治療のとき1ターン Timer を増やす量
	RecoverPer              int          // 治療済みで1ターン Timer を減らす基準量。質と代謝で増減する
	bloodDropPerSeverity    int          // 重症度1段あたりに下げる血液量。外傷は出血、重い全身性は循環低下
	Cause                   DeathCause   // 血液量が尽きて倒したときの死因
}

// conditionDefs は状態種類ごとの定義。状態を足すときはここへ1行足す
var conditionDefs = map[ConditionType]ConditionDef{
	ConditionHypothermia: {
		displayName: "Hypothermia",
		description: "The body is dangerously cold. Warm up to recover.",
		// 常在圧力なので反応率は外傷より軽くする。中度で戦える重さに留め、
		// 重症の凍死は血液量 34 の HP-1/ターンでじわじわ効かせる
		painPerSeverity: 6, capacityDropPerSeverity: 10,
		// Recovery なし。進行と回復は TemperatureSystem が体温から扱う。
		// 直接 HP は削らず、重症で血液量を下げて凍死させる
		bloodDropPerSeverity: 22, Cause: CauseFrozen,
	},
	ConditionFracture: {
		displayName:     "Fracture",
		description:     "A broken bone. It will not heal until treated.",
		painPerSeverity: 18, capacityDropPerSeverity: 20,
		// 骨は肉より治りが遅い。切り傷の半分の速さにし、治療しても完治まで長く付き合わせる
		Recovery: RecoverAfterTend, RecoverPer: 1,
	},
	ConditionLaceration: {
		displayName:     "Laceration",
		description:     "An open wound. It will not heal until treated.",
		painPerSeverity: 8, capacityDropPerSeverity: 8,
		Recovery: RecoverAfterTend, RecoverPer: 2, bloodDropPerSeverity: 25, Cause: CauseBloodLoss,
	},
	ConditionLiverIllness: {
		displayName:     "Liver illness",
		description:     "It worsens while untreated and drains HP when severe.",
		painPerSeverity: 4, capacityDropPerSeverity: 10,
		Recovery: ProgressUntilTend, WorsenPer: 2, RecoverPer: 1, bloodDropPerSeverity: 25, Cause: CauseIllness,
	},
	ConditionFoodPoisoning: {
		displayName:     "Food poisoning",
		description:     "Nausea from bad food. It clears on its own over time.",
		painPerSeverity: 5, capacityDropPerSeverity: 12,
		Recovery: RecoverOverTime, RecoverPer: 2,
	},
}

// ConditionDefFor は状態種類の定義を返す。未登録なら ok=false。ConditionSystem と capacity 導出が読む
func ConditionDefFor(ct ConditionType) (ConditionDef, bool) {
	def, ok := conditionDefs[ct]
	return def, ok
}

// ConditionTypeDisplayName は状態種類の表示名を返す。未登録なら素のIDを返す
func ConditionTypeDisplayName(ct ConditionType) string {
	if def, ok := conditionDefs[ct]; ok {
		return def.displayName
	}
	return string(ct)
}

// ConditionTypeDescription は状態種類の概要説明を返す。未登録なら空文字を返す
func ConditionTypeDescription(ct ConditionType) string {
	return conditionDefs[ct].description
}

// BodyCapacities は身体機能の一式。すべて基準 100 の consts.Percent で、100 が正常、
// 低いほど機能が落ちる。痛み Pain だけは 0 が無痛で、大きいほど痛い。
// 不調から読み取り時に導出し、保存はしない
type BodyCapacities struct {
	Pain          consts.Percent // 0 が無痛。大きいほど痛い
	Blood         consts.Percent // 100 が正常。失血や重い不調で下がる。危険域を下回ると HP が減る
	Consciousness consts.Percent // 100 が正常。痛みと全身性の不調で下がる。全機能に掛かる乗数
	Manipulation  consts.Percent // 腕・手の不調で下がる
	Moving        consts.Percent // 脚・足の不調で下がる
	Sight         consts.Percent // 頭の不調で下がる
}

// painConsciousnessDivisor は痛みが意識を下げる割合。痛みをこれで割ったぶん意識が下がる。
// 症状に依らない集約側の係数。値は実プレイで調整する
const painConsciousnessDivisor = 2

// CapacityKind は部位が下げる身体機能の区分。値は表示名 msgid で UI は query.T で訳す
type CapacityKind string

// 身体機能の区分
const (
	CapacityConsciousness CapacityKind = "Consciousness" // 胴・全身の全身性
	CapacityManipulation  CapacityKind = "Manipulation"  // 腕・手
	CapacityMoving        CapacityKind = "Moving"        // 脚・足
	CapacitySight         CapacityKind = "Sight"         // 頭
)

// bodyPartMeta は部位ごとの静的情報。表示名・下げる身体機能・命中重みを1つの表に持つ
type bodyPartMeta struct {
	displayName string       // 表示名 msgid
	capacity    CapacityKind // この部位の不調が下げる身体機能
	hitWeight   int          // 戦闘の命中抽選の重み。大きい部位ほど当たる。0 は命中先から除外
}

// bodyPartMetas は部位ごとの静的情報の唯一の表。部位階層は持たず平坦に持つ。
// 部位を足すときはここへ1行足す。全身は全身性の状態の受け皿で戦闘の命中先ではないので命中重み 0
var bodyPartMetas = [BodyPartCount]bodyPartMeta{
	BodyPartHead:      {displayName: "Head", capacity: CapacitySight, hitWeight: 2},
	BodyPartTorso:     {displayName: "Torso", capacity: CapacityConsciousness, hitWeight: 4},
	BodyPartArms:      {displayName: "Arm", capacity: CapacityManipulation, hitWeight: 3},
	BodyPartHands:     {displayName: "Hand", capacity: CapacityManipulation, hitWeight: 1},
	BodyPartLegs:      {displayName: "Leg", capacity: CapacityMoving, hitWeight: 3},
	BodyPartFeet:      {displayName: "Foot", capacity: CapacityMoving, hitWeight: 1},
	BodyPartWholeBody: {displayName: "Whole body", capacity: CapacityConsciousness, hitWeight: 0},
}

// bodyPartCapacity は部位が下げる身体機能を返す
func bodyPartCapacity(part BodyPart) CapacityKind {
	return bodyPartMetas[part].capacity
}

// BodyPartHitWeight は戦闘の命中抽選での部位の重みを返す。0 なら命中先にならない
func BodyPartHitWeight(part BodyPart) int {
	return bodyPartMetas[part].hitWeight
}

// HealthyCapacities は不調が無いときの身体機能を返す。HealthStatus を持たない対象の既定
func HealthyCapacities() BodyCapacities {
	return (&HealthStatus{}).Capacities()
}

// treatedPenaltyRemain は応急処置後に残る痛みと機能低下の割合。全治まで軽減して残す。
// 出血や HP 減少は治療で止まるが、機能低下はゼロにはならず全治まで引きずる
const treatedPenaltyRemain consts.Percent = 50

// ConditionCapacityImpact は不調1件が身体機能へ与える影響を返す。
// capacity は部位で定まり重症度に依らない。drop が0なら影響なし
func ConditionCapacityImpact(cond *HealthCondition, part BodyPart) (pain int, capacity CapacityKind, drop int) {
	capacity = bodyPartCapacity(part)
	pain, drop = conditionSeverityImpact(cond)
	return pain, capacity, drop
}

// conditionSeverityImpact は不調1件の痛みと機能低下を返す。症状ごとの反応率に重症度を掛ける。
// 応急処置済みなら軽減して返す。表示と適用がこの1関数を共有するので両者がずれない
func conditionSeverityImpact(cond *HealthCondition) (pain, drop int) {
	m := conditionSeverityMultiplier(cond.Severity)
	if m == 0 {
		return 0, 0
	}
	def := conditionDefs[cond.Type]
	pain, drop = def.painPerSeverity*m, def.capacityDropPerSeverity*m
	if cond.TendQuality > 0 {
		pain = treatedPenaltyRemain.ApplyInt(pain)
		drop = treatedPenaltyRemain.ApplyInt(drop)
	}
	return pain, drop
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

// AddCondition は状態を1つ積む。SetCondition と違い同種でも上書きせず別の傷として足す。
// 戦闘の怪我のように独立した傷を重ねるのに使う
func (bph *BodyPartHealth) AddCondition(cond HealthCondition) {
	bph.Conditions = append(bph.Conditions, cond)
}

// CountConditions は指定した種類の状態がこの部位にいくつあるかを返す。独立傷のソフト上限判定に使う
func (bph *BodyPartHealth) CountConditions(condType ConditionType) int {
	n := 0
	for i := range bph.Conditions {
		if bph.Conditions[i].Type == condType {
			n++
		}
	}
	return n
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

// bloodCriticalThreshold は血液量がこれを下回ると失血で HP が減り始める割合。
// bloodDrainStep は血液量の不足をこれで割った数だけ毎ターン削る。深く落ちるほど速く減る。
// 外傷は出血で、重症の低体温や病気は循環低下で血液量を下げる。値は実プレイで調整する
const (
	bloodCriticalThreshold = 40
	bloodDrainStep         = 10
)

// conditionBloodDrop は不調1件がいま下げている血液量を返す。0 なら下げない。
// 外傷は発症中ずっと出血し、重症の全身性の不調は循環を落とす。治療済みは止まる
func conditionBloodDrop(cond *HealthCondition) int {
	def := conditionDefs[cond.Type]
	if def.bloodDropPerSeverity == 0 || cond.TendQuality > 0 {
		return 0
	}
	m := conditionSeverityMultiplier(cond.Severity)
	if m == 0 {
		return 0
	}
	// 外傷は発症中ずっと血を失う。全身性の不調は重症でだけ循環が落ちる
	if def.Recovery == RecoverAfterTend {
		if !cond.IsActive() {
			return 0
		}
	} else if cond.Severity != SeveritySevere {
		return 0
	}
	return def.bloodDropPerSeverity * m
}

// ConditionBloodDrop は不調1件がいま下げている血液量を返す。健康タブの表示に使う公開ラッパ
func ConditionBloodDrop(cond *HealthCondition) int {
	return conditionBloodDrop(cond)
}

// BloodLossHPDrain は血液量の低下による毎ターンの HP 減少量と死因を返す。
// 血液量が危険域を下回ると、不足に応じてじわじわ HP を削る。失血・凍死・衰弱死はすべてここを通る
func (hs *HealthStatus) BloodLossHPDrain() (int, DeathCause) {
	blood := int(hs.Capacities().Blood)
	if blood >= bloodCriticalThreshold {
		return 0, ""
	}
	deficit := bloodCriticalThreshold - blood
	drain := (deficit + bloodDrainStep - 1) / bloodDrainStep
	return drain, hs.bloodLossCause()
}

// bloodLossCause は血液量を最も下げている不調の死因を返す。該当が無ければ失血に落とす
func (hs *HealthStatus) bloodLossCause() DeathCause {
	cause := CauseBloodLoss
	best := 0
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			c := &hs.Parts[i].Conditions[j]
			if d := conditionBloodDrop(c); d > best {
				best = d
				if def := conditionDefs[c.Type]; def.Cause != "" {
					cause = def.Cause
				}
			}
		}
	}
	return cause
}

// IsHPDraining はいま HP を削る要因があるかを返す。血液量が危険域なら削っている。
// 削っているあいだは自然回復を止め、減っているのを読み取れるようにする
func (hs *HealthStatus) IsHPDraining() bool {
	drain, _ := hs.BloodLossHPDrain()
	return drain > 0
}

// Capacities は不調から身体機能の一式を導出する。保存済みの値でなく Timer と Severity から計算する。
// 部位ごとの不調が対応機能を下げ、痛みと全身性の不調が意識を下げ、意識が全機能へ乗算される
func (hs *HealthStatus) Capacities() BodyCapacities {
	pain := 0
	bloodDrop := 0
	var manip, moving, sight, systemic int // 各機能の低下量
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			cond := &hs.Parts[i].Conditions[j]
			p, drop := conditionSeverityImpact(cond)
			pain += p
			bloodDrop += conditionBloodDrop(cond)
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
		Blood:         consts.Percent(clamp(100-bloodDrop, 0, 100)),
		Consciousness: consts.Percent(consciousness),
		Manipulation:  withConsciousness(100 - manip),
		Moving:        withConsciousness(100 - moving),
		Sight:         withConsciousness(100 - sight),
	}
}
