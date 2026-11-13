package balance

// BattleMetric 武器×敵の戦闘シミュレーション結果
type BattleMetric struct {
	DPS      float64
	Enemy    string
	IsRanged bool
	Player   string
	Weapon   string
}

// DepthStat 1深度の統計情報
type DepthStat struct {
	Depth              int
	MedianDamage       int
	MedianHP           int
	MedianHPBeforeHeal int
	MedianHealing      int
	MedianHunger       int
	MedianKillTurns    int
	MedianWeaponDamage int
	P5HP               int
	P5HPBeforeHeal     int
	P5Hunger           int
	P5KillTurns        int
	P5WeaponDamage     int
	P95HP              int
	P95HPBeforeHeal    int
	P95Hunger          int
	P95KillTurns       int
	P95WeaponDamage    int
	SuddenDeathRate    float64
}

// EnemyTableRun 1つの敵テーブルに対するシミュレーション結果
type EnemyTableRun struct {
	DeathRate   float64
	Depths      []DepthStat
	MaxDepth    int
	MedianDepth int
	Name        string
	Trials      int
}

// FacilityLoot 1施設種別の loot 分布
type FacilityLoot struct {
	Facility  string
	Templates []TemplateLoot
	Trials    int
}

// LootItemStat 1アイテムの loot 統計
type LootItemStat struct {
	ExpectedCount float64
	Name          string
	Prob          float64
	Value         int
}

// PlayerInfo プレイヤーのステータス情報
type PlayerInfo struct {
	Agility   int
	Defense   int
	Dexterity int
	HP        int
	Name      string
	Sensation int
	Strength  int
}

// Report シミュレーション結果全体を表す
type Report struct {
	BattleMetrics []BattleMetric
	EnemyTables   []EnemyTableRun
	Player        PlayerInfo
	FacilityLoot  []FacilityLoot
	Weapon        WeaponInfo
}

// TemplateLoot 施設の絵1枚の loot 分布。Trials はその絵が選ばれた試行数
type TemplateLoot struct {
	Items    []LootItemStat
	Template string
	Trials   int
}

// WeaponInfo 武器の情報
type WeaponInfo struct {
	Accuracy int
	Damage   int
	Name     string
}
