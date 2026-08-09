namespace RuinsEditorApi.Balance;

// ================== balance レポート ==================
// simulate-balance が書き出す balance.json の構造を定義する。HTTP operation は持たず、モデルだけを
// 両側へ生成する。Go は internal/balance が生成型を直接使い、editor-ui は生成 TS 型を import する。
// 配信は静的 balance.json のままで、この namespace は保存ファイルの形だけを単一ソースとして固定する。

/** balance.json の最上位構造。シミュレーション結果全体を表す */
model Report {
  /** レポートの種別 */
  mode: string;

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
  name: string;
  hp: int32;
  strength: int32;
  sensation: int32;
  dexterity: int32;
  agility: int32;
  defense: int32;
}

/** 武器の情報 */
model WeaponInfo {
  name: string;
  damage: int32;
  accuracy: int32;
}

/** 武器×敵の戦闘シミュレーション結果 */
model BattleMetric {
  player: string;
  weapon: string;
  enemy: string;

  /** 1秒あたりの推定ダメージ */
  dps: float64;

  /** 遠距離武器かどうか */
  isRanged: boolean;
}

/** 1つの敵テーブルに対するシミュレーション結果 */
model EnemyTableRun {
  name: string;
  maxDepth: int32;
  trials: int32;
  medianDepth: int32;

  /** 死亡率 */
  deathRate: float64;

  /** 深度ごとの統計 */
  depths: DepthStat[];

  /** 試行ごとの生データ */
  trialData: TrialResult[];
}

/** 1試行の結果 */
model TrialResult {
  index: int32;
  reachedDepth: int32;
  died: boolean;
  depths: TrialDepthStat[];
}

/** 1試行の1深度の情報 */
model TrialDepthStat {
  depth: int32;
  hp: int32;
  hpBeforeHeal: int32;
  weapon: string;
  hunger: int32;
}

/** 1深度の統計情報 */
model DepthStat {
  depth: int32;
  medianHP: int32;
  p5HP: int32;
  p95HP: int32;
  medianHPBeforeHeal: int32;
  p5HPBeforeHeal: int32;
  p95HPBeforeHeal: int32;

  /** 突然死率 */
  suddenDeathRate: float64;

  medianWeaponDamage: int32;
  p5WeaponDamage: int32;
  p95WeaponDamage: int32;
  medianKillTurns: int32;
  p5KillTurns: int32;
  p95KillTurns: int32;
  medianHunger: int32;
  p5Hunger: int32;
  p95Hunger: int32;
  medianDamage: int32;
  medianHealing: int32;
}

/** 1施設種別の loot 分布 */
model FacilityLoot {
  facility: string;
  trials: int32;

  /** 部屋役割ごとの loot */
  rooms: RoomLoot[];
}

/** 1部屋役割の loot 分布 */
model RoomLoot {
  /** 部屋役割 */
  role: string;

  /** アイテムごとの統計 */
  items: LootItemStat[];
}

/** 1アイテムの loot 統計 */
model LootItemStat {
  name: string;

  /** 出現確率 */
  prob: float64;

  /** 期待個数 */
  expectedCount: float64;

  /** 売買価値 */
  value: int32;
}
