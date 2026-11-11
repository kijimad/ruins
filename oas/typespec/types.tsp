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

@minLength(1)
@maxLength(50)
scalar EntityID extends string;

/** テーブルエントリの重み。大きいほど選ばれやすい */
@minValue(0)
scalar EntryWeight extends float64;

/** アイテムコレクションのエントリの出現確率。百分率。書き忘れて0になった行を弾くため0は許さない */
@minValueExclusive(0)
@maxValue(100)
scalar ItemCollectionChance extends float64;

/** 部屋の役割名。interior の roleName に相当する。main/back は Go 内部専用で raw.toml には現れないため含めない */
enum RoomRole {
  bath,
  bedroom,
  coldroom,
  corridor,
  dressing,
  examination_room,
  genkan,
  kitchen,
  living,
  office,
  pharmacy,
  restroom,
  storage,
  storeroom,
  toilet,
  waiting,
}

/** 建物の損傷段階 */
enum BuildingDamage {
  intact,
  minor,
  major,
}

/** 建物の散らかり段階 */
enum BuildingClutter {
  tidy,
  messy,
  filthy,
}

/** 建物の事情。稀な1軒へ物語を与える */
enum BuildingVariant {
  standard,
  abandoned,
  under_construction,
  survivalist,
  hoarder,
}

/** 敷地の類型。前庭の深さと外構の出し分けに使う */
enum SiteType {
  /** 戸建。塀で囲い前庭に観葉 */
  detached,

  /** 商店街。街路に面して建て前庭は浅く、塀を張らず看板と自販機を出す */
  shopfront,

  /** ロードサイド。前庭を広く取り前面を駐車場にする */
  roadside,
}

/** 内装の経年や散らかりの割合。百分率 */
@minValue(0)
@maxValue(100)
scalar InteriorPercent extends integer;

/** 敷地の類型の選ばれる重み。0 ならその種類の建物では選ばれない */
@minValue(0)
scalar SiteWeight extends int32;

/** 入口側に空ける前庭の奥行き。タイル */
@minValue(0)
@maxValue(10)
scalar YardDepth extends int32;

/** 建物の家具量の倍率。10 が等倍の ×/10 */
@minValue(1)
@maxValue(30)
scalar DensityFactor extends integer;

/** 天候の種類 */
enum WeatherId {
  clear,
  cloudy,
  rain,
  snow,
  blizzard,
  cold_snap,
}

/** 季節 */
enum SeasonId {
  spring,
  summer,
  autumn,
  winter,
}

/** 時間帯。1日は夜明けで始まり、この順に巡る */
enum TimeOfDayId {
  dawn,
  morning,
  noon,
  evening,
  night,
  midnight,
}

/** 時間帯による世界温度への加算。℃ */
@minValue(-50)
@maxValue(50)
scalar TimeOfDayTempModifier extends integer;

/** 光の強さ。0 が真っ暗、1 が最大 */
@minValue(0)
@maxValue(1)
scalar LightLevel extends float64;

/** 季節の世界温度。℃ */
@minValue(-100)
@maxValue(100)
scalar SeasonTemp extends integer;

/** 天候による世界温度への加算。℃ */
@minValue(-50)
@maxValue(50)
scalar WeatherTempModifier extends integer;

/** 天候による視程レンジへの加算。タイル。負で狭くなる */
@minValue(-100)
@maxValue(100)
scalar WeatherVisionDelta extends integer;

/** 天候の寒さの厳しさ。奥地ほど厳しい天候を厚くし、冬と奥地で長引かせるのに使う */
@minValue(0)
@maxValue(10)
scalar WeatherSeverity extends integer;

/** 天候のスペルの基準長。ターン */
@minValue(1)
@maxValue(100000)
scalar SpellTurns extends integer;

/** 不調の重症度 */
enum ConditionSeverity {
  minor,
  medium,
  severe,
}

/** 最大値に対する割合 */
@minValue(0)
@maxValue(1)
scalar NeedRatio extends float64;

/** 空腹の段階 */
enum HungerStageId {
  satiated,
  normal,
  hungry,
  starving,
}

/** 疲労の段階 */
enum FatigueStageId {
  rested,
  normal,
  tired,
  exhausted,
}

/** ダンジョンのフロアを生成する間取りの種類 */
enum DungeonPlanner {
  small_room,
  big_room,
  cave,
  ruins,
  forest,
}

/** ステージの基本気温。℃ */
@minValue(-100)
@maxValue(100)
scalar StageBaseTemp extends integer;

/** 会話の1ページ。英語の msgid で、表示のとき query.T で訳す。<keyword> などのマークアップを含められる */
@minLength(1)
scalar DialoguePage extends string;

/** 鮮度の段階 */
enum FreshnessStageId {
  fresh,
  stale,
  rotten,
}

/** 病気を起こす確率。百分率 */
@minValue(0)
@maxValue(100)
scalar IllnessChancePercent extends integer;

/** 食料から得る栄養の満額に対する割合。百分率 */
@minValue(0)
@maxValue(100)
scalar NutritionPercent extends integer;

/** 材質のkgあたり燃焼熱量。0 なら不燃で燃料にならない */
@minValue(0)
@maxValue(10000)
scalar HeatPerKg extends integer;

/** 間取りテンプレの選択キー */
enum PlannerKey {
  house,
  store,
  clinic,
  bsp,
}

/** 市街地の地区 */
enum Zone {
  downtown,
  residential,
  industrial,
}

/** ランドマークの描画関数の選択キー */
enum DrawerKey {
  hut,
  open,
}

/** 耐久値。削り切られると壊れる */
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

/** 不調の id。Go の ConditionType 定数の値と同じ */
@minLength(1)
@maxLength(50)
@pattern("^[a-z][a-z0-9_]*$")
scalar ConditionTypeName extends string;

/** 部位の id */
@minLength(1)
@maxLength(50)
@pattern("^[a-z][a-z0-9_]*$")
scalar BodyPartId extends string;

/** 不調の重症度1段あたりの量。痛み、身体機能の低下、血液量の低下に使う */
@minValue(0)
@maxValue(100)
scalar ConditionAmountPerSeverity extends integer;

/** 不調の1ターンあたりの進行度の増減量 */
@minValue(1)
@maxValue(100)
scalar ConditionTimerRate extends integer;

/** 不調が未治療のときの振る舞いと治し方 */
enum RecoveryMode {
  /** 治療して初めて回復する。未治療は保持し悪化しない */
  recover_after_tend,
  /** 未治療なら悪化し続け、治療して初めて回復する */
  progress_until_tend,
  /** 未治療でも時間で自然に治る。治療すれば速まる */
  recover_over_time,
}

/** 不調を起こす仕組み。仕組みごとにちょうど1つの不調が持つ */
enum ConditionOnset {
  /** 体温が下がると TemperatureSystem が進める */
  cold,
  /** 腐敗食を食べると確率で発症する */
  rotten_food,
  /** 疲労の量から読み取り時に導出する */
  fatigue,
  /** 空腹の量から読み取り時に導出する */
  hunger,
}

/** 不調が血液量を尽きさせて倒したときの死因 */
enum ConditionDeathCause {
  frozen,
  illness,
  blood_loss,
}

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

/** キューブ展開範囲の伸び幅。縦横を一律に広げるタイル数 */
@minValue(0)
@maxValue(20)
scalar RangeBonus extends integer;

/** キューブモジュールの燃料容量ボーナス。装着するとキューブの燃料ポイントの上限がこの量だけ増える */
@minValue(0)
scalar FuelCapacityBonus extends integer;

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
@pattern("^[1-9][0-9]*d[1-9][0-9]*([+-][0-9]+)?$")
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

/** 鮮度の段階が始まる劣化量。食料の stageLength の何倍か */
@minValue(0)
scalar FreshnessStageFrom extends integer;

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
enum SkillId {
  sword,
  spear,
  fist,
  weight_bearing,
  bow,
  handgun,
  rifle,
  cannon,
  exploration,
  crafting,
  smithing,
  negotiation,
  mechanic,
  sprinting,
  stealth,
  night_vision,
  cold_resist,
  hunger_resist,
  healing,
  heavy_armor,
  fire_resist,
  thunder_resist,
  chill_resist,
  photon_resist,
}

/** スキルのカテゴリの id */
@minLength(1)
@maxLength(50)
@pattern("^[a-z][a-z0-9_]*$")
scalar SkillCategoryId extends string;

/** 熟練の効果倍率の id。技能と能力値から導く行動ごとの倍率を指す。怪我・病気・疲労・空腹による身体機能の低下は別に扱う */
enum ProficiencyId {
  fire_resist,
  thunder_resist,
  chill_resist,
  photon_resist,
  cold_progress,
  hunger_progress,
  healing_effect,
  max_weight,
  exploration,
  enemy_vision,
  night_vision,
  move_cost,
  craft_cost,
  smith_quality,
  buy_price,
  sell_price,
  heavy_armor,
  sword_damage,
  spear_damage,
  fist_damage,
  bow_damage,
  handgun_damage,
  rifle_damage,
  cannon_damage,
  sword_accuracy,
  spear_accuracy,
  fist_accuracy,
  bow_accuracy,
  handgun_accuracy,
  rifle_accuracy,
  cannon_accuracy,
}

/** 熟練の効果倍率のスキルLv1あたりの変化量。百分率 */
@minValue(-100)
@maxValue(100)
scalar ProficiencyPerLevel extends integer;

/** 能力値。スキルの成長の速さと熟練の効果倍率に効く */
enum AbilityId {
  str,
  sen,
  dex,
  agi,
  vit,
  def,
}

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
不燃の材質は係数0で燃料にならない。係数は materials の行が持つ */
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

/** 攻撃種別。武器の攻撃種別は同じ名前の武器スキルで上達する */
/** 攻撃の射程の区分 */
enum AttackRange {
  melee,
  ranged,
}

/** 射程。タイル */
@minValue(1)
@maxValue(50)
scalar RangeTiles extends integer;

/** 射程の超過1タイルあたりの命中の低下。百分率 */
@minValue(0)
@maxValue(100)
scalar RangePenaltyPercent extends integer;

enum AttackCategory {
  /** 素手 */
  fist,
  /** 剣 */
  sword,
  /** 槍 */
  spear,
  /** 拳銃 */
  handgun,
  /** ライフル */
  rifle,
  /** 砲 */
  cannon,
  /** 弓 */
  bow,
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
  /** 随伴。プレイヤーに隣接するまで1歩ずつ近づく */
  follow,
}

/** スプライト描画深度 */
enum SpriteDepth {
  /** 床レイヤー */
  Floor: 0,
  /** 床の模様レイヤー。3D では地面に寝かせて描く */
  Decal: 1,
  /** 敷物レイヤー */
  Rug: 2,
  /** 背の高いオブジェクトレイヤー */
  Taller: 3,
  /** プレイヤーレイヤー */
  Player: 4,
}

/** タイルの囲われの度合い。屋内外の判定と、世界温度の受け方に効く */
enum Shelter {
  outdoor,
  partial,
  indoor,
}

/** 世界温度を引き寄せる割合。百分率。100 なら世界温度をそのまま受ける */
@minValue(0)
@maxValue(100)
scalar ShelterInfluencePercent extends integer;

/** タイルが受ける水の影響 */
enum WaterExposure {
  /** 水のタイルの隣 */
  nearby,
  /** 水のタイルの上 */
  submerged,
}

/** タイルの植生 */
enum Foliage {
  grass,
  forest,
}

/** 水や植生がタイルの気温に足す補正。℃ */
@minValue(-50)
@maxValue(50)
scalar TileTempModifier extends integer;

/** 回復量の計算方式 */
enum HealingValueType {
  /** 最大値に対する割合で回復 */
  PERCENTAGE,
  /** 固定値で回復 */
  ABSOLUTE,
  /** 数値で回復 */
  NUMERAL,
}

/** 分解定義の道具が無いと殴っても削れないか */
scalar DisassemblyToolRequired extends boolean;

/** 分解産出の確率。百分率 */
@minValue(1)
@maxValue(100)
scalar DisassemblyChance extends integer;

/** 道具のグレード。高いほど速く壊し多く得る */
@minValue(1)
@maxValue(3)
scalar ToolGrade extends integer;

/** 道具の分類 */
enum ToolCategory {
  /** こじ開け。木製家具・棚など */
  prying,
  /** 精密。電子機器・機械など */
  precision,
  /** 切断。金属大物など */
  cutting,
  /** 伐採。木など */
  chopping,
  /** 採掘。岩など */
  mining,
}
