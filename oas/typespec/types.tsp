namespace RuinsEditorApi;

// ================== スカラー型 ==================

/** 命中率。0で必中なし、100で必中 */
@minValue(0)
@maxValue(100)
scalar Accuracy extends int32;

/** 命中率補正値 */
@minValue(-100)
@maxValue(100)
scalar AccuracyBonus extends int32;

/** 行動コスト */
@minValue(0)
@maxValue(100)
scalar ActionCost extends int32;

/** 敏捷性。行動順に影響する */
@minValue(-100)
@maxValue(100)
scalar Agility extends int32;

/** 防具の防御力 */
@minValue(0)
@maxValue(999)
scalar ArmorDefense extends int32;

/** 1ターンあたりの攻撃回数 */
@minValue(1)
@maxValue(10)
scalar AttackCount extends int32;

/** 基本ダメージ */
@minValue(0)
@maxValue(9999)
scalar BaseDamage extends int32;

/** 通行を妨げるかどうか */
scalar BlocksPassage extends boolean;

/** 視線を遮るかどうか */
scalar BlocksView extends boolean;

/** RGBA色チャネル値 (0-255) */
scalar ColorChannel extends uint8;

/** ダメージ補正値 */
@minValue(-100)
@maxValue(100)
scalar DamageBonus extends int32;

/** 防御力。被ダメージを軽減する */
@minValue(-100)
@maxValue(100)
scalar Defense extends int32;

/** 階層レベル */
@minValue(0)
@maxValue(100)
scalar DepthLevel extends int32;

/** 器用さ。クリティカル率に影響する */
@minValue(-100)
@maxValue(100)
scalar Dexterity extends int32;

/** 説明文 */
@minLength(1)
@maxLength(200)
scalar EntityDescription extends string;

/** エンティティ名 */
@minLength(1)
@maxLength(50)
scalar EntityName extends string;

/** テーブルエントリの重み。大きいほど選ばれやすい */
@minValue(0)
scalar EntryWeight extends float64;

/** 耐久値。設定すると破壊可能になる */
@minValue(1)
@maxValue(9999)
scalar HitPoints extends int32;

/** 回復固定量 */
@minValue(0)
@maxValue(9999)
scalar HealAmount extends int32;

/** 回復割合。0.0〜1.0 */
@minValue(0)
@maxValue(1)
scalar HealRatio extends float64;

/** 画像ファイルパス */
@minLength(1)
@maxLength(200)
@pattern("^[a-zA-Z0-9_/\\-]+\\.[a-zA-Z0-9]+$")
scalar ImagePath extends string;

/** 耐寒性能 */
@minValue(-100)
@maxValue(100)
scalar InsulationCold extends int32;

/** 耐暑性能 */
@minValue(-100)
@maxValue(100)
scalar InsulationHeat extends int32;

/** ボスモンスターかどうか */
scalar IsBoss extends boolean;

/** プレイヤーキャラクターかどうか */
scalar IsPlayer extends boolean;

/** アイテム所持数 */
@minValue(1)
@maxValue(9999)
scalar ItemCount extends int32;

/** 売買価格 */
@minValue(0)
@maxValue(99999)
scalar ItemValue extends int32;

/** アイテム重量（kg） */
@minValue(0)
@maxValue(1000)
scalar ItemWeight extends float64;

/** 光源が有効かどうか */
scalar LightEnabled extends boolean;

/** 光の到達半径（タイル単位） */
@minValue(1)
@maxValue(100)
scalar LightRadius extends int32;

/** マガジン容量 */
@minValue(1)
@maxValue(100)
scalar MagazineSize extends int32;

/** 素材必要数 */
@minValue(1)
@maxValue(99)
scalar MaterialAmount extends int32;

/** 通行コスト加算値。0で変化なし、50でベースコスト+50 */
@minValue(0)
@maxValue(9999)
scalar PassCost extends int32;

/** メッセージリソースのキー */
@minLength(1)
@maxLength(100)
@pattern("^[a-z][a-z0-9_]*$")
scalar MessageKey extends string;

/** 栄養価 */
@minValue(0)
@maxValue(9999)
scalar NutritionAmount extends int32;

/** パックの最大数 */
@minValue(1)
@maxValue(100)
scalar PackMax extends int32;

/** パックの最小数 */
@minValue(1)
@maxValue(100)
scalar PackMin extends int32;

/** パレットID */
@minLength(1)
@maxLength(50)
@pattern("^[a-zA-Z][a-zA-Z0-9_]*$")
scalar PaletteId extends string;

/** 職業ID */
@minLength(1)
@maxLength(50)
@pattern("^[a-z][a-z0-9_]*$")
scalar ProfessionId extends string;

/** 読了に必要な総読書量 */
@minValue(1)
@maxValue(9999)
scalar ReadingEffort extends int32;

/** リロードに必要な行動力 */
@minValue(1)
@maxValue(100)
scalar ReloadEffort extends int32;

/** 感覚。命中率と回避率に影響する */
@minValue(-100)
@maxValue(100)
scalar Sensation extends int32;

/** スキルID */
@minLength(1)
@maxLength(50)
@pattern("^[a-z][a-z0-9_]*$")
scalar SkillId extends string;

/** スキルレベル */
@minValue(0)
@maxValue(100)
scalar SkillLevel extends int32;

/** スプライトキー */
@minLength(1)
@maxLength(100)
@pattern("^[a-z][a-z0-9_]*$")
scalar SpriteKey extends string;

/** スプライトシート名 */
@minLength(1)
@maxLength(50)
@pattern("^[a-z][a-z0-9_]*$")
scalar SpriteSheetName extends string;

/** スタック可能かどうか */
scalar Stackable extends boolean;

/** 筋力。物理ダメージに影響する */
@minValue(-100)
@maxValue(100)
scalar Strength extends int32;

/** AI視界距離（タイル単位） */
@minValue(1)
@maxValue(100)
scalar ViewDistance extends int32;

/** 体力。HPに影響する */
@minValue(-100)
@maxValue(100)
scalar Vitality extends int32;

// ================== 共通enum ==================

/** ターゲット数 */
enum TargetNum {
  /** 単体対象 */
  SINGLE,
  /** 全体対象 */
  ALL,
}

/** ターゲットグループ */
enum TargetGroup {
  /** 敵を対象にする */
  ENEMY,
  /** 味方を対象にする */
  ALLY,
  /** 武器依存の対象 */
  WEAPON,
  /** 対象なし */
  NONE,
}

/** 使用可能シーン */
enum UsableScene {
  /** 戦闘中のみ使用可能 */
  BATTLE,
  /** フィールドでのみ使用可能 */
  FIELD,
  /** どこでも使用可能 */
  ANY,
}

/** 攻撃属性 */
enum Element {
  /** 無属性 */
  NONE,
  /** 火属性 */
  FIRE,
  /** 雷属性 */
  THUNDER,
  /** 氷属性 */
  CHILL,
  /** 光属性 */
  PHOTON,
}

/** 攻撃種別 */
enum AttackCategory {
  /** 素手 */
  FIST,
  /** 剣 */
  SWORD,
  /** 槍 */
  SPEAR,
  /** 拳銃 */
  HANDGUN,
  /** ライフル */
  RIFLE,
  /** 砲 */
  CANON,
  /** 弓 */
  BOW,
}

/** 装備部位 */
enum EquipmentCategory {
  /** 頭 */
  HEAD,
  /** 胴 */
  TORSO,
  /** 腕 */
  ARMS,
  /** 手 */
  HANDS,
  /** 脚 */
  LEGS,
  /** 足 */
  FEET,
  /** 装飾品 */
  JEWELRY,
}

/** 装備スロット。防具部位 + 武器スロット */
enum EquipSlot {
  /** 頭防具 */
  HEAD,
  /** 胴防具 */
  TORSO,
  /** 腕防具 */
  ARMS,
  /** 手防具 */
  HANDS,
  /** 脚防具 */
  LEGS,
  /** 足防具 */
  FEET,
  /** 武器スロット1 */
  WEAPON1,
  /** 武器スロット2 */
  WEAPON2,
  /** 武器スロット3 */
  WEAPON3,
  /** 武器スロット4 */
  WEAPON4,
  /** 武器スロット5 */
  WEAPON5,
}

/** 弾薬タグ */
enum AmmoTag {
  /** 9mm弾 */
  `9mm`: "9mm",
  /** ライフル弾 */
  rifle,
  /** 散弾 */
  shell,
  /** エネルギーセル */
  energy_cell,
}

/** 派閥タイプ */
enum FactionMemberType {
  /** 中立派閥 */
  FactionNeutral,
}

/** 態度タイプ。エンティティの他者に対する初期態度を定義する */
enum DispositionType {
  /** 敵対態度。視界内のプレイヤーを攻撃する */
  hostile,
  /** 中立態度。攻撃されると反撃する */
  neutral,
  /** 臆病な態度。攻撃されると逃亡する */
  cowardly,
}

/** 非戦闘時の移動パターン */
enum MovementPatternType {
  /** ランダム移動 */
  random,
  /** 定点巡回 */
  patrol,
  /** 壁沿い移動 */
  wallHug,
  /** 固定。移動しない */
  stationary,
  /** 徘徊。低頻度で周辺をランダム移動する */
  wander,
  /** 縄張り。スポーン地点から一定範囲内で移動する */
  territorial,
  /** 群れ。同種族の仲間に寄る */
  swarm,
}

/** スプライト描画深度 */
enum SpriteDepth {
  /** 床レイヤー */
  Floor: 0,
  /** 敷物レイヤー */
  Rug: 1,
  /** 背の高いオブジェクトレイヤー */
  Taller: 2,
  /** プレイヤーレイヤー */
  Player: 3,
}

/** 遮蔽タイプ */
enum ShelterType {
  /** 遮蔽なし */
  None: 0,
  /** 部分遮蔽 */
  Partial: 5,
  /** 完全遮蔽 */
  Full: 10,
}

/** 水タイプ */
enum WaterType {
  /** 水なし */
  None: 0,
  /** 水辺。移動ペナルティ小 */
  Nearby: -5,
  /** 水没。移動ペナルティ大 */
  Submerged: -10,
}

/** 植生タイプ */
enum FoliageType {
  /** 植生なし */
  None: 0,
  /** 草原。視界ペナルティ小 */
  Grass: -1,
  /** 森。視界ペナルティ大 */
  Forest: -3,
}

/** 回復量の計算方式 */
enum HealingValueType {
  /** 最大値に対する割合で回復 */
  PERCENTAGE,
  /** 固定値で回復 */
  ABSOLUTE,
  /** 数値で回復 */
  NUMERAL,
}
