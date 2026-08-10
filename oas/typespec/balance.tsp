namespace RuinsEditorApi.Balance;

// simulate-balance が書き出す balance.json の構造を定義する。

// ================== スカラー型 ==================

/** レポートの種別 */
scalar ReportMode extends string;

/** 戦闘員名。プレイヤーや敵の raw id を指す */
scalar CombatantName extends string;

/** 武器名。武器アイテムの raw id を指す */
scalar WeaponName extends string;

/** 敵テーブル名 */
scalar EnemyTableName extends string;

/** 施設種別 id */
scalar FacilityId extends string;

/** 部屋役割 id */
scalar RoomRole extends string;

/** アイテム名。loot アイテムの raw id を指す */
scalar ItemName extends string;

/** 能力値。プレイヤーのステータス */
scalar AbilityScore extends integer;

/** HP 値。最大値や統計値を表す。計算値は負になりうるので無制約にする */
scalar HP extends integer;

/** ダメージ量 */
@minValue(0)
scalar Damage extends integer;

/** 命中率。0-100 */
@minValue(0)
@maxValue(100)
scalar Accuracy extends integer;

/** 深度。地上を1とする階層 */
@minValue(0)
scalar Depth extends integer;

/** 撃破に要するターン数 */
@minValue(0)
scalar KillTurns extends integer;

/** 空腹度 */
scalar Hunger extends integer;

/** 回復量 */
@minValue(0)
scalar Healing extends integer;

/** 試行回数 */
@minValue(0)
scalar TrialCount extends integer;

/** 試行のインデックス。0始まり */
@minValue(0)
scalar TrialIndex extends integer;

/** アイテムの売買価値 */
@minValue(0)
scalar ItemValue extends integer;

/** 割合。0.0-1.0 */
@minValue(0)
@maxValue(1)
scalar Rate extends float64;

/** 1秒あたりの推定ダメージ */
@minValue(0)
scalar DPS extends float64;

/** 期待個数 */
@minValue(0)
scalar ExpectedCount extends float64;

/** 遠距離武器かどうか */
scalar RangedFlag extends boolean;

/** 死亡したかどうか */
scalar DeathFlag extends boolean;

// ================== モデル ==================

/** balance.json の最上位構造。シミュレーション結果全体を表す */
model Report {
  /** レポートの種別 */
  mode: ReportMode;

  /** プレイヤーのステータス。生成しないモードでは省く */
  player?: PlayerInfo;

  /** 武器の情報。生成しないモードでは省く */
  weapon?: WeaponInfo;

  /** 敵テーブルごとのシミュレーション結果 */
  enemyTables: EnemyTableRun[];

  /** 武器×敵の戦闘メトリクス */
  battleMetrics: BattleMetric[];

  /** 施設種別ごとの loot 分布 */
  roomLoot: FacilityLoot[];
}

/** プレイヤーのステータス情報 */
model PlayerInfo {
  name: CombatantName;
  hp: HP;
  strength: AbilityScore;
  sensation: AbilityScore;
  dexterity: AbilityScore;
  agility: AbilityScore;
  defense: AbilityScore;
}

/** 武器の情報 */
model WeaponInfo {
  name: WeaponName;
  damage: Damage;
  accuracy: Accuracy;
}

/** 武器×敵の戦闘シミュレーション結果 */
model BattleMetric {
  player: CombatantName;
  weapon: WeaponName;
  enemy: CombatantName;
  dps: DPS;
  isRanged: RangedFlag;
}

/** 1つの敵テーブルに対するシミュレーション結果 */
model EnemyTableRun {
  name: EnemyTableName;
  maxDepth: Depth;
  trials: TrialCount;
  medianDepth: Depth;
  deathRate: Rate;

  /** 深度ごとの統計 */
  depths: DepthStat[];

  /** 試行ごとの生データ */
  trialData: TrialResult[];
}

/** 1試行の結果 */
model TrialResult {
  index: TrialIndex;
  reachedDepth: Depth;
  died: DeathFlag;
  depths: TrialDepthStat[];
}

/** 1試行の1深度の情報 */
model TrialDepthStat {
  depth: Depth;
  hp: HP;
  hpBeforeHeal: HP;
  weapon: WeaponName;
  hunger: Hunger;
}

/** 1深度の統計情報 */
model DepthStat {
  depth: Depth;
  medianHP: HP;
  p5HP: HP;
  p95HP: HP;
  medianHPBeforeHeal: HP;
  p5HPBeforeHeal: HP;
  p95HPBeforeHeal: HP;
  suddenDeathRate: Rate;
  medianWeaponDamage: Damage;
  p5WeaponDamage: Damage;
  p95WeaponDamage: Damage;
  medianKillTurns: KillTurns;
  p5KillTurns: KillTurns;
  p95KillTurns: KillTurns;
  medianHunger: Hunger;
  p5Hunger: Hunger;
  p95Hunger: Hunger;
  medianDamage: Damage;
  medianHealing: Healing;
}

/** 1施設種別の loot 分布 */
model FacilityLoot {
  facility: FacilityId;
  trials: TrialCount;

  /** 部屋役割ごとの loot */
  rooms: RoomLoot[];
}

/** 1部屋役割の loot 分布 */
model RoomLoot {
  /** 部屋役割 */
  role: RoomRole;

  /** アイテムごとの統計 */
  items: LootItemStat[];
}

/** 1アイテムの loot 統計 */
model LootItemStat {
  name: ItemName;

  /** 1棟あたり1個以上出る確率 */
  prob: Rate;

  /** 1棟あたりの期待個数 */
  expectedCount: ExpectedCount;

  /** 売買価値 */
  value: ItemValue;
}
