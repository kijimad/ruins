namespace RuinsEditorApi;

// ================== スカラー型 ==================

/** 命中率。0で必中なし、100で必中 */
@minValue(0)
@maxValue(100)
scalar Accuracy extends integer;

/** 命中率補正値 */
@minValue(-100)
@maxValue(100)
scalar AccuracyBonus extends integer;

/** 行動コスト */
@minValue(0)
@maxValue(100)
scalar ActionCost extends integer;

/** 敏捷性。行動順に影響する */
@minValue(-100)
@maxValue(100)
scalar Agility extends integer;

/** 防具の防御力 */
@minValue(0)
@maxValue(999)
scalar ArmorDefense extends integer;

/** 1ターンあたりの攻撃回数 */
@minValue(1)
@maxValue(10)
scalar AttackCount extends integer;

/** 基本ダメージ */
@minValue(0)
@maxValue(9999)
scalar BaseDamage extends integer;

/** 通行を妨げるかどうか */
scalar BlocksPassage extends boolean;

/** 視線を遮るかどうか */
scalar BlocksView extends boolean;

/** RGBA色チャネル値 (0-255) */
scalar ColorChannel extends uint8;

/** ダメージ補正値 */
@minValue(-100)
@maxValue(100)
scalar DamageBonus extends integer;

/** 防御力。被ダメージを軽減する */
@minValue(-100)
@maxValue(100)
scalar Defense extends integer;

/** 危険度。経過日数から決まる */
@minValue(0)
@maxValue(100)
scalar DangerLevel extends integer;

/** 器用さ。クリティカル率に影響する */
@minValue(-100)
@maxValue(100)
scalar Dexterity extends integer;

/** 説明文 */
@minLength(1)
@maxLength(200)
scalar EntityDescription extends string;

/** エンティティ名 */
@minLength(1)
@maxLength(50)
scalar EntityName extends string;

/** エンティティの英語 id */
@minLength(1)
@maxLength(50)
scalar EntityID extends string;

/** テーブルエントリの重み。大きいほど選ばれやすい */
@minValue(0)
scalar EntryWeight extends float64;

/** 耐久値。設定すると破壊可能になる */
@minValue(1)
@maxValue(9999)
scalar HitPoints extends integer;

/** 回復固定量 */
@minValue(0)
@maxValue(9999)
scalar HealAmount extends integer;

/** 回復割合。0.0〜1.0 */
@minValue(0)
@maxValue(1)
scalar HealRatio extends float64;

/** 治療の質。基準100の倍率。100が標準、150で回復1.5倍 */
@minValue(0)
@maxValue(9999)
scalar Potency extends integer;

/** 治療できる不調の種類ID。骨折や切り傷などの ConditionType 値 */
@minLength(1)
scalar ConditionTypeName extends string;

/** 画像ファイルパス */
@minLength(1)
@maxLength(200)
@pattern("^[a-zA-Z0-9_/\\-]+\\.[a-zA-Z0-9]+$")
scalar ImagePath extends string;

/** 耐寒性能 */
@minValue(-100)
@maxValue(100)
scalar InsulationCold extends integer;

/** 耐暑性能 */
@minValue(-100)
@maxValue(100)
scalar InsulationHeat extends integer;

/** プレイヤーキャラクターかどうか */
scalar IsPlayer extends boolean;

/** アイテム所持数 */
@minValue(1)
@maxValue(9999)
scalar ItemCount extends integer;

/** 売買価格 */
@minValue(0)
@maxValue(99999)
scalar ItemValue extends integer;

/** 重量。単位付き文字列で指定する。例: "500 g" "2 kg" "1 mg" */
@pattern("^[0-9]+(\\.[0-9]+)? (mg|g|kg)$")
scalar Weight extends string;

/** ダイス表記の個数指定。固定値も 1d1 のように書く。consts.ParseDice でパースする */
@pattern("^[0-9]+d[0-9]+([+-][0-9]+)?$")
@example("1d3+1")
scalar Dice extends string;

/** 光源が有効かどうか */
scalar LightEnabled extends boolean;

/** 光の到達半径（タイル単位） */
@minValue(1)
@maxValue(100)
scalar LightRadius extends integer;

/** 熱の到達半径。チェビシェフ距離、タイル単位 */
@minValue(1)
@maxValue(100)
scalar HeatRadius extends integer;

/** 熱源が毎ターン下げる低体温タイマーの量 */
@minValue(0)
scalar HeatWarmth extends float64;

/** 寝具の睡眠効率。100が基準で、高いほど短く深く眠れる */
@minValue(0)
scalar BeddingQuality extends integer;

/** マガジン容量 */
@minValue(1)
@maxValue(100)
scalar MagazineSize extends integer;

/** 素材必要数 */
@minValue(1)
@maxValue(99)
scalar MaterialAmount extends integer;

/** 通行コスト加算値。0で変化なし、50でベースコスト+50 */
@minValue(0)
@maxValue(9999)
scalar PassCost extends integer;

/** メッセージリソースのキー */
@minLength(1)
@maxLength(100)
@pattern("^[a-z][a-z0-9_]*$")
scalar MessageKey extends string;

/** 栄養価 */
@minValue(0)
@maxValue(9999)
scalar NutritionAmount extends integer;

/** 1段階の長さ。この経過ターンごとに新鮮→劣化→腐敗と進む。省略すると腐敗しない */
@minValue(1)
@maxValue(100000)
scalar StageLengthTurns extends integer;

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
scalar ReadingEffort extends integer;

/** リロードに必要な行動力 */
@minValue(1)
@maxValue(100)
scalar ReloadEffort extends integer;

/** 感覚。命中率と回避率に影響する */
@minValue(-100)
@maxValue(100)
scalar Sensation extends integer;

/** スキルID */
@minLength(1)
@maxLength(50)
@pattern("^[a-z][a-z0-9_]*$")
scalar SkillId extends string;

/** スキルレベル */
@minValue(0)
@maxValue(100)
scalar SkillLevel extends integer;

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

/** 筋力。物理ダメージに影響する */
@minValue(-100)
@maxValue(100)
scalar Strength extends integer;

/** AI視界距離（タイル単位） */
@minValue(1)
@maxValue(100)
scalar ViewDistance extends integer;

/** 体力。HPに影響する */
@minValue(-100)
@maxValue(100)
scalar Vitality extends integer;

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

/** 材質。可燃性と燃焼熱量の算出に使う。燃料熱量は材質のkgあたり熱量へ重量を掛けて導く。
不燃の材質は係数0で燃料にならない。係数は balance 値なので Go 側が持つ */
enum Material {
  /** 木材 */
  WOOD,
  /** 紙 */
  PAPER,
  /** 布 */
  CLOTH,
  /** 革 */
  LEATHER,
  /** 植物繊維。縄・藁・乾いた草 */
  PLANT,
  /** 食料 */
  FOOD,
  /** 骨 */
  BONE,
  /** 油 */
  OIL,
  /** 石炭 */
  COAL,
  /** 樹脂・プラスチック */
  PLASTIC,
  /** 金属。不燃 */
  METAL,
  /** 石。不燃 */
  STONE,
  /** ガラス。不燃 */
  GLASS,
  /** 結晶・宝石。不燃 */
  CRYSTAL,
  /** 陶磁器。不燃 */
  CERAMIC,
  /** 液体。水や薬品。不燃。油は OIL で別に扱う */
  LIQUID,
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

/** 戦闘ポリシー。エンティティの戦闘時の行動方針を定義する */
enum CombatPolicyType {
  /** 攻撃。視界内の敵を攻撃する */
  attack,
  /** 回避。敵から距離を取って逃げる */
  evade,
  /** 無関心。戦闘に反応しない。被ダメージで攻撃に変化する */
  ignore,
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

/** 分解の基礎工数。100が標準1ターンに相当する */
@minValue(1)
@maxValue(9999)
scalar DisassemblyBaseAP extends integer;

/** 分解産出の確率。百分率 */
@minValue(1)
@maxValue(100)
scalar DisassemblyChance extends integer;

/** 分解工具のグレード。高いほど速く多く得る */
@minValue(1)
@maxValue(3)
scalar ToolGrade extends integer;

/** 分解工具の分類 */
enum ToolCategory {
  /** こじ開け。木製家具・棚など */
  prying,
  /** 精密。電子機器・機械など */
  precision,
  /** 切断。金属大物など */
  cutting,
}
