package components

// Ability は変動するパラメータ値
type Ability struct {
	Base     int // 固有の値
	Modifier int // 装備や健康状態で変動する値
	Total    int // 計算した現在値。算出される値のメモ
}

// Abilities はエンティティが持つ能力値。各種計算式で使う
type Abilities struct {
	Vitality  Ability // 体力。丈夫さ、持久力、しぶとさ。HPやSPに影響する
	Strength  Ability // 筋力。主に近接攻撃のダメージに影響する
	Sensation Ability // 感覚。主に射撃攻撃のダメージに影響する
	Dexterity Ability // 器用。攻撃時の命中率に影響する
	Agility   Ability // 敏捷。回避率、行動の速さに影響する
	Defense   Ability // 防御。被弾ダメージを軽減させる
}

// AbilityID は能力値の識別子
type AbilityID int

// 能力値ID定数
const (
	AblSTR AbilityID = iota // 筋力
	AblSEN                  // 感覚
	AblDEX                  // 器用
	AblAGI                  // 敏捷
	AblVIT                  // 体力
	AblDEF                  // 防御
)

// AbilityName は能力値IDの表示名を返す
var AbilityName = map[AbilityID]string{
	AblSTR: "STR",
	AblSEN: "SEN",
	AblDEX: "DEX",
	AblAGI: "AGI",
	AblVIT: "VIT",
	AblDEF: "DEF",
}

// ValueOf は指定された能力値IDに対応するTotal値を返す
func (a *Abilities) ValueOf(id AbilityID) int {
	switch id {
	case AblSTR:
		return a.Strength.Total
	case AblSEN:
		return a.Sensation.Total
	case AblDEX:
		return a.Dexterity.Total
	case AblAGI:
		return a.Agility.Total
	case AblVIT:
		return a.Vitality.Total
	case AblDEF:
		return a.Defense.Total
	default:
		return 0
	}
}
