namespace RuinsEditorApi;

// ================== アイテム ==================

/** 消費可能アイテムの設定 */
model Consumable {
  usableScene: UsableScene;
  targetGroup: TargetGroup;
  targetNum: TargetNum;
}

/** 回復効果 */
model ProvidesHealing {
  valueType: HealingValueType;
  amount: HealAmount;
  ratio: HealRatio;
}

/** 治療する性質。Treats に一致する不調のうち最も重い1つを治療済みにする */
model Remedy {
  treats: ConditionTypeName[];
  potency: Potency;
}

/** 装備可能設定 */
model Wearable {
  defense: ArmorDefense;
  equipmentCategory: EquipmentCategory;
  insulationCold: InsulationCold;
  insulationHeat: InsulationHeat;
}

/** 装備ボーナス */
model EquipBonus {
  vitality: Vitality;
  strength: Strength;
  sensation: Sensation;
  dexterity: Dexterity;
  agility: Agility;
}

/** 弾薬設定 */
model Ammo {
  ammoTag?: AmmoTag;
  damageBonus: DamageBonus;
  accuracyBonus: AccuracyBonus;
}

/** キューブモジュール設定。装着すると展開野営の範囲や燃料タンクの容量を広げる */
model CubeModule {
  rangeBonus: RangeBonus;
  fuelCapacityBonus: FuelCapacityBonus;
}

/** 装着設定。キューブの展開空間へ装着するとフィールドで設備として働くアイテムであることを表す */
model Deployable {
  storage?: StorageRaw;
  lightSource?: LightSource;
  bedding?: Bedding;
}

/** 近接攻撃設定 */
model Melee {
  accuracy: Accuracy;
  damage: BaseDamage;
  attackCount: AttackCount;
  element: Element;
  attackCategory: AttackCategory;
  cost: ActionCost;
  targetGroup: TargetGroup;
  targetNum: TargetNum;
}

/** 遠距離攻撃設定 */
model Fire {
  accuracy: Accuracy;
  damage: BaseDamage;
  attackCount: AttackCount;
  element: Element;
  attackCategory: AttackCategory;
  cost: ActionCost;
  targetGroup: TargetGroup;
  targetNum: TargetNum;
  magazineSize: MagazineSize;
  reloadEffort: ReloadEffort;
  ammoTag?: AmmoTag;
}

/** スキル本設定 */
model SkillBook {
  targetSkill: SkillId;
  requiredLevel: SkillLevel;
  maxLevel: SkillLevel;
}

/** 本の設定 */
model Book {
  totalEffort: ReadingEffort;
  skill?: SkillBook;
}

/** アイテム */
model Item {
  id: EntityID;
  name: EntityName;
  description: EntityDescription;
  spriteSheetName: SpriteSheetName;
  spriteKey: SpriteKey;
  animKeys?: SpriteKey[];
  value: ItemValue;
  /** アイテム重量 */
  weight?: Weight;
  inflictsDamage?: BaseDamage;
  providesNutrition?: NutritionAmount;
  stageLength?: StageLengthTurns;
  consumable?: Consumable;
  providesHealing?: ProvidesHealing;
  remedy?: Remedy;
  wearable?: Wearable;
  equipBonus?: EquipBonus;
  ammo?: Ammo;
  cubeModule?: CubeModule;
  deployable?: Deployable;
  melee?: Melee;
  fire?: Fire;
  book?: Book;
  /** 道具の性質。合う物を殴って壊すと速く多く得る */
  tool?: Tool;
  /** 携行光源。装備すると owner を照らす */
  lightSource?: LightSource;
  /** 材質。可燃性と燃焼熱量を導く。可燃な材質のアイテムは地面に置いて着火したり火にくべて燃やせる */
  material?: Material;
  /** 火種。所持していると隣接の燃焼物に着火できる */
  fireStarter?: FireStarter;
}

// ================== メンバー ==================

/** 能力値 */
model Abilities {
  vitality: Vitality;
  strength: Strength;
  sensation: Sensation;
  dexterity: Dexterity;
  agility: Agility;
  defense: Defense;
}

/** RGBA色 */
model RGBAColor {
  r: ColorChannel;
  g: ColorChannel;
  b: ColorChannel;
  a: ColorChannel;
}

/** 光源設定 */
model LightSource {
  radius: LightRadius;
  color: RGBAColor;
  enabled: LightEnabled;
}

/** 熱源設定。近接するキャラの低体温を毎ターン回復する */
model HeatSource {
  radius: HeatRadius;
  warmth: HeatWarmth;
}

/** 寝具設定。眠るときの睡眠効率を決める */
model Bedding {
  quality: BeddingQuality;
}

/** 火種設定。所持していると隣接タイルの燃焼物に火をつけられる道具を表す */
model FireStarter {}

/** 会話データ */
model Dialog {
  messageKey: MessageKey;
}

/** メンバー */
model Member {
  id: EntityID;
  name: EntityName;
  player?: IsPlayer;
  abilities: Abilities;
  spriteSheetName: SpriteSheetName;
  spriteKey: SpriteKey;
  animKeys?: SpriteKey[];
  lightSource?: LightSource;
  factionType?: FactionMemberType;
  combatPolicy?: CombatPolicyType;
  movementPattern?: MovementPatternType;
  viewDistance?: ViewDistance;
  dialog?: Dialog;
  /** プレイヤーキャラクターでは省略可能 */
  commandTableId?: EntityID;
  /** プレイヤーキャラクターでは省略可能 */
  dropTableId?: EntityID;

  /** 生成時に装備して持たせるアイテム。startingItems より先に持たせる */
  startingEquips?: ProfessionEquip[];

  /** 生成時にバックパックへ持たせるアイテム */
  startingItems?: ProfessionItem[];

  /** 店の在庫。持つ member は店を開く */
  stock?: ProfessionItem[];
}

// ================== レシピ ==================

/** レシピ素材 */
model RecipeInput {
  id: EntityID;
  amount: MaterialAmount;
}

/** レシピ */
model Recipe {
  id: EntityID;
  name: EntityName;
  inputs: RecipeInput[];
}

// ================== テーブル ==================

/** コマンドテーブルエントリ */
model CommandTableEntry {
  /** 使う武器の id */
  id: EntityID;
  weight: EntryWeight;
}

/** コマンドテーブル */
model CommandTable {
  id: EntityID;
  name: EntityName;
  entries: CommandTableEntry[];
}

/** ドロップテーブルエントリ */
model DropTableEntry {
  /** ドロップする素材の id */
  id: EntityID;
  weight: EntryWeight;
}

/** ドロップテーブル */
model DropTable {
  id: EntityID;
  name: EntityName;
  /** 何も落とさない重み。省略すると0で、必ずどれかを落とす */
  nothing?: EntryWeight;
  entries: DropTableEntry[];
}

/** アイテムグループエントリ */
model ItemGroupEntry {
  id: EntityID;
  weight: EntryWeight;
  /** 1山あたりの個数のダイス表記 */
  pack: Dice;
}

/** アイテムグループ。エントリ群から weight の比で1つだけ選ぶ */
model ItemGroup {
  id: EntityID;
  name: EntityName;
  entries: ItemGroupEntry[];
}

/** アイテムコレクションエントリ */
model ItemCollectionEntry {
  id: EntityID;
  chance: ItemCollectionChance;
  /** 1山あたりの個数のダイス表記 */
  pack: Dice;
}

/** アイテムコレクション。各エントリを chance の確率で独立に判定する。複数当選することもありうる */
model ItemCollection {
  id: EntityID;
  name: EntityName;
  entries: ItemCollectionEntry[];
}

/** アイテムテーブルエントリ。アイテムグループかアイテムコレクションを参照する */
model ItemTableEntry {
  /** 参照するアイテムグループかアイテムコレクションの id */
  id: EntityID;
  weight: EntryWeight;
  minDanger: DangerLevel;
  maxDanger: DangerLevel;
}

/** アイテムテーブル */
model ItemTable {
  id: EntityID;
  name: EntityName;
  entries: ItemTableEntry[];
}

/** 敵テーブルエントリ */
model EnemyTableEntry {
  id: EntityID;
  weight: EntryWeight;
  minDanger: DangerLevel;
  maxDanger: DangerLevel;
  /** 1群あたりの敵数のダイス表記 */
  pack: Dice;
}

/** 敵テーブル */
model EnemyTable {
  id: EntityID;
  name: EntityName;
  entries: EnemyTableEntry[];
}

/** 市街地の施設1種。isShop は看板やシャッターを出す店か */
model Facility {
  id: EntityID;
  isShop: boolean;
  planner: PlannerKey;
  enemyTable: EntityID;
  zones: FacilityZone[];
  /** 収納家具が荒らされて空になっている確率。0から100の百分率で、省略は0 */
  @minValue(0) @maxValue(100) lootedChance?: int32;
}

/** 施設が出現する地区と抽選重み。市街地の一辺が minSpan 以上のときだけ抽選対象になる */
model FacilityZone {
  zone: Zone;
  @minValue(1) weight: int32;
  minSpan: int32;
}

/** 原野の点在ランドマーク1種。hutW/hutH は drawer=hut のときだけ使う */
model Landmark {
  id: EntityID;
  @minValue(1) weight: int32;
  drawer: DrawerKey;
  hutW: int32;
  hutH: int32;
  props: PropSpot[];
}

/** 基準点から相対の位置に置くエンティティ1個 */
model PropSpot {
  name: EntityID;
  dx: int32;
  dy: int32;
}

/** 俯瞰地図の記号1種。id は地物・施設の種別 id */
model MapGlyph {
  id: EntityID;
  glyph: string;
  name: string;
  color: RGBAColor;
  order: int32;

  /** この記号で描く、持ち主の行を持たないチャンクの種類。施設と集落とランドマークの記号は持ち主の行の id で引くので省く */
  chunk?: GlyphChunk;
}

/** 開けた地形の散布ゾーン1種。grass は草を置くタイルに、soil は草の生えていない土へ soilDensity の確率で置く地面の飾り */
model ScatterZone {
  id: EntityID;
  grassDensity: float64;
  soilDensity: float64;
  propDensity: float64;
  lootGroup: EntityID;
  entries: ScatterEntry[];
  grass: GroundCoverEntry[];
  soil: GroundCoverEntry[];
}

/** 地面を覆う草や土の飾りの prop 1種と、選ばれる重み */
model GroundCoverEntry {
  ref: EntityID;
  @minValue(1) weight: int32;
}

/** 原野の小集落1種。当選したチャンクで weight の比で種類を選び、中心から相対の位置に住人の member と prop を置く。
    id は地図の記号の id を兼ねる。行の順に重みを積む */
model Settlement {
  id: EntityID;
  @minValue(1) weight: int32;

  /** 住人の member。name は members の id */
  members: PropSpot[];

  props: PropSpot[];
}

/** 散布 prop 1種。ref の空文字は置かないことを表す */
model ScatterEntry {
  ref: string;
  @minValue(1) weight: int32;
  big: boolean;
  satellites?: PropSpot[];
}

// ================== 内装レシピ ==================

/** 配置指示の種別。家具・戦利品・敵・装飾・罠を同じ器で扱う */
enum StuffKind {
  /** 家具。占有し移動を阻む */
  furniture,
  /** 戦利品。コンテナや床置き */
  loot,
  /** 敵・住人 */
  being,
  /** 装飾。通行を阻まない小物 */
  decor,
  /** 罠 */
  trap,
}

/** グループの抽選方式。保証セットとランダム充填を分ける */
enum GroupStyle {
  /** Items を全部置く。各 Stuff は chance で個別に gate する */
  pick_each,
  /** Items から重みで1つ選ぶ */
  pick_one,
  /** Items から重複なく pick 個選ぶ */
  pick_n,
}

/** 配置の置き方。空なら interiorRefs の既定、それも無ければ全域へ落ちる */
enum Placement {
  /** 部屋中心寄り。中央の島什器など */
  center,
  /** 壁際。戸口隣接面は避ける。棚・冷蔵ケース */
  wall,
  /** 全域に散らす。ゴミ・小物 */
  full_area,
  /** 入口近く。レジ・カート */
  near_door,
  /** 入口から遠い奥。冷蔵ケース・貴重品 */
  far_from_door,
  /** 棚を1列おきの平行列に置き、間の列を通路として空ける。ゴンドラ・ラック */
  row,
}

/** 相対座標。衛星の anchor からのオフセットに使う。負値は上/左方向 */
model ContentVec {
  x: int32;
  y: int32;
}

/** anchor 相対に一緒に置く衛星。机に対する椅子など。束で置いて散布事故を防ぐ */
model ContentSatellite {
  kind: StuffKind;
  ref: EntityID;
  offsets: ContentVec[];
  /** 収納家具の中身を引く itemTables の id。収納家具では必須 */
  lootTable?: EntityID;
}

/** 内装レシピの1配置指示 */
model ContentStuff {
  kind: StuffKind;
  ref: EntityID;
  weight?: EntryWeight;
  /** 置く個数のダイス表記 */
  amount: Dice;
  placement?: Placement;
  satellites?: ContentSatellite[];
  /** 収納家具の中身を引く itemTables の id。収納家具では必須 */
  lootTable?: EntityID;
}

/** 抽選単位の束 */
model ContentGroup {
  style: GroupStyle;
  /** pick_n のときの選ぶ個数。pick_each / pick_one では使わないので省略可 */
  pick?: int32;
  items: ContentStuff[];
}

/** 内装レシピ。施設まるごと、または奥室1つに対応する */
model InteriorContent {
  id: EntityID;
  groups?: ContentGroup[];
}

/** 内装レシピの Ref。spawn する prop か床 loot の item group と、既定の置き方を持つ */
model InteriorRef {
  id: EntityID;

  /** 家具と装飾として置く raw の prop。省くと prop として置かない */
  prop?: EntityID;

  /** 床 loot として引く itemGroups か itemCollections の id。省くと床 loot として置かない */
  itemGroup?: EntityID;

  /** レシピが placement を省いたときの置き方 */
  placement?: Placement;
}

/** 建物の損傷段階の重み */
model DamageChoice {
  id: BuildingDamage;
  weight: EntryWeight;

  /** 経年で戦利品を撤去する割合 */
  lootPercent: InteriorPercent;

  /** 廃墟化で壁際のタイルに瓦礫の種を撒く割合。0 から 1 */
  rubbleSeedRatio: NeedRatio;
}

/** 建物の散らかり段階の重み */
model ClutterChoice {
  id: BuildingClutter;
  weight: EntryWeight;

  /** 家具1つの隣へ小物を落とす割合。0 なら散らかりを足さない */
  furniturePercent: InteriorPercent;

  /** 家具から離れた床のタイルへ小物を散らす割合 */
  floorPercent: InteriorPercent;
}

/** 建物の変種の重み */
model VariantChoice {
  id: BuildingVariant;
  weight: EntryWeight;

  /** 引いた損傷段階を上書きする。省くと上書きしない */
  damage?: BuildingDamage;

  /** 散らかり段階をこれ以上に引き上げる。省くと引き上げない */
  minClutter?: BuildingClutter;

  /** 引いた家具量の倍率を上書きする。省くと上書きしない */
  density?: DensityFactor;
}

/** 敷地の類型の、民家と店それぞれで選ばれる重みと前庭の奥行き。行の順に重みを積む */
model SiteChoice {
  id: SiteType;
  homeWeight: SiteWeight;
  shopWeight: SiteWeight;
  yardDepth: YardDepth;

  /** 敷地を塀で囲うか。街路に面して開く敷地は囲わない */
  fence: boolean;
}

/** 街路側の前壁に付ける外皮の interiorRefs の id */
model FacadeRefs {
  window: EntityID;

  /** 大破した店が窓の代わりに下ろすシャッター */
  shutter: EntityID;

  /** 店の入口脇に立てる看板 */
  sign: EntityID;
}

/** 敷地の外構の interiorRefs の id */
model LotRefs {
  fence: EntityID;

  /** 民家の前庭に置く外構 */
  plant: EntityID;

  /** 店の前庭に置く外構 */
  vending: EntityID;
}

/** 建物の家具量の倍率の重み */
model DensityChoice {
  factor: DensityFactor;
  weight: EntryWeight;
}

/** hero 部屋の目玉の種類の重み。選ばれた種類の refs から1つを一様に引く */
model HeroChoice {
  id: EntityID;
  weight: EntryWeight;
  @minItems(1) refs: EntityID[];
}

/** 建物単位の生活感の抽選表。建物ごとに各軸を1回引き、全室へ一様に効かせる */
model InteriorProfile {
  damages: DamageChoice[];
  clutters: ClutterChoice[];
  variants: VariantChoice[];
  densities: DensityChoice[];
  heroes: HeroChoice[];
  sites: SiteChoice[];
  facade: FacadeRefs;
  lot: LotRefs;

  /** 経年で家具1つが隣へ生活痕を落とす割合 */
  wearPercent: InteriorPercent;

  /** hero 部屋の目玉を持つ建物の割合の逆数。この数の建物に1棟が目玉を持つ */
  @minValue(1)
  heroOneIn: int32;
}

/** 施設種別ごとの主室の内装変種。抽選で1つ選ぶ */
model FacilityContent {
  facility: EntityID;
  variants: EntityID[];
}

/** 奥室の役割名と内装レシピの対 */
model RoomContent {
  role: RoomRole;
  content: EntityID;
}

/** 施設種別ごとの奥室カタログ。役割別 content と、カタログに無い役割のフォールバック */
model FacilityRooms {
  facility: EntityID;
  rooms?: RoomContent[];
  fallback: EntityID;
}

// ================== タイル ==================

/** スプライトレンダー設定 */
model SpriteRender {
  spriteSheetName: SpriteSheetName;
  spriteKey: SpriteKey;
  depth: SpriteDepth;
}

/** タイル */
model Tile {
  id: EntityID;
  name: EntityName;
  description: EntityDescription;
  blockPass: BlocksPassage;
  blockView: BlocksView;
  spriteRender: SpriteRender;

  /** 水のタイル。上に立つと水中、隣に立つと水辺の気温の補正を受ける */
  water?: boolean;

  /** 植生。省くと植生なし */
  foliage?: Foliage;

  /** マップの計画からこのタイルを生成するときの仕様。省くと計画に置けない */
  spawn?: TileSpawn;
}

/** 計画のタイルを実体化する仕様 */
model TileSpawn {
  /** 生成するタイルの id。壁の計画を壁の見た目の別タイルで生成するときに使う。省くと自身 */
  tile?: EntityID;

  /** 周りのタイルを見てオートタイルの添字を計算するか。単一の絵柄のタイルは false */
  autotile: boolean;
}

/** NPC の会話。member の dialog の messageKey が id を指す。ページの順に1ページずつ見せる */
model Dialogue {
  id: MessageKey;
  @minItems(1) pages: DialoguePage[];
}

/** ダンジョンの間取りの種類と、フロアごとに選ばれる重み */
model DungeonPlannerWeight {
  id: DungeonPlanner;
  weight: EntryWeight;
}

/** フロアを生成して潜るダンジョン。id はステージの識別名で、遺跡の入口が指す */
model Dungeon {
  id: EntityID;
  name: EntityName;
  description: EntityDescription;

  /** 背景画像のスプライトキー */
  imageKey: EntityID;

  /** 敵を引く enemyTables の id */
  enemyTable: EntityID;

  /** アイテムを引く itemTables の id */
  itemTable: EntityID;

  baseTemp: StageBaseTemp;

  /** フロアの間取りの抽選表。行の順に重みを積む */
  @minItems(1)
  planners: DungeonPlannerWeight[];
}

/** 環境光の色。明るさは daylight が持ち、ここは色味だけを表す */
model AmbientColor {
  r: LightLevel;
  g: LightLevel;
  b: LightLevel;
}

/** 時間帯の表示名、世界温度への加算、地上の日照の明るさと環境光の色。
    明るさと色は時間帯の中心の値で、間は隣の時間帯の値と線形に補間する */
model TimeOfDay {
  id: TimeOfDayId;
  name: EntityName;
  tempModifier: TimeOfDayTempModifier;
  daylight: LightLevel;
  ambientColor: AmbientColor;
}

/** 季節の表示名と世界温度。春と秋は年の前半と後半の中点、夏は頂、冬は底の温度で、間を正弦波でつなぐ */
model Season {
  id: SeasonId;
  name: EntityName;
  temp: SeasonTemp;
}

/** 鮮度の段階の表示名と、食べたとき得る栄養の割合。段階の境目は Go の Perishable が持つ */
model FreshnessStage {
  id: FreshnessStageId;
  name: EntityName;
  nutritionPercent: NutritionPercent;

  /** 食べたときのログの msgid。%s は食べた者と食べ物 */
  eatLog: EntityDescription;

  /** メニューの行に鮮度を添えるか */
  marker: boolean;

  /** 食べたとき食中毒を起こす確率。百分率。省くと起こさない */
  illnessChancePercent?: IllnessChancePercent;
}

/** 材質の表示名と燃焼の熱量 */
model MaterialDef {
  id: Material;
  name: EntityName;
  heatPerKg: HeatPerKg;
}

/** 道具の分類と、合う道具で壊したとき伸びるスキル。スキルを省くと伸びない */
model ToolCategoryDef {
  id: ToolCategory;
  skill?: SkillId;
}

/** 囲われの度合いごとの、世界温度の受け方と表示名。受ける温度は anchorTemp + (世界温度 - anchorTemp) * influencePercent / 100 */
model ShelterEffect {
  id: Shelter;
  name: EntityName;
  anchorTemp: TileTempModifier;
  influencePercent: ShelterInfluencePercent;
}

/** 水がタイルの気温に足す補正 */
model WaterEffect {
  id: WaterExposure;
  tempModifier: TileTempModifier;
}

/** 植生がタイルの気温に足す補正 */
model FoliageEffect {
  id: Foliage;
  tempModifier: TileTempModifier;
}

// ================== 置物 ==================

/** 扉ローデータ */
model DoorRaw {
  horizontalOpen: SpriteKey;
  horizontalClosed: SpriteKey;
  verticalOpen: SpriteKey;
  verticalClosed: SpriteKey;
}

/** 次階層ワープトリガー */
model WarpNextTriggerRaw {}

/** 前階層ワープトリガー */
model WarpPrevTriggerRaw {}

/** 収納ローデータ */
model StorageRaw {
  /** 収納の最大格納重量 */
  maxWeight: Weight;
  /** 初期アイテムの抽選に使うItemTable名 */
  lootTableId?: EntityID;
  /** 初期アイテム数のダイス表記。省略時は1 */
  lootCount?: Dice;
}

/** 置物 */
model Prop {
  id: EntityID;
  name: EntityName;
  description: EntityDescription;
  spriteRender: SpriteRender;
  animKeys?: SpriteKey[];
  blockPass: BlocksPassage;
  blockView: BlocksView;
  passCost?: PassCost;
  /** 耐久。省略した prop は殴れない */
  hp?: HitPoints;
  lightSource?: LightSource;
  heatSource?: HeatSource;
  bedding?: Bedding;
  door?: DoorRaw;
  warpNextTrigger?: WarpNextTriggerRaw;
  warpPrevTrigger?: WarpPrevTriggerRaw;
  storage?: StorageRaw;
  disassembly?: Disassembly;

  /** 燃え尽きたときに残すアイテム。火のように燃え尽きる prop だけが持つ */
  burnoutItem?: EntityID;
}

/** 分解の産出エントリ。chance 省略は確定枠 */
model DisassemblyYield {
  id: EntityID;
  /** 産出個数のダイス表記 */
  count: Dice;
  chance?: DisassemblyChance;
}

/** 分解のボーナス産出。minSkill か minGrade の少なくとも一方を指定する。両方指定した場合は両方を満たす必要がある */
model DisassemblyBonus {
  id: EntityID;
  /** 産出個数のダイス表記 */
  count: Dice;
  /** 機械スキルがこの値以上で産出に加わる */
  minSkill?: SkillLevel;
  /** 工具グレードがこの値以上で産出に加わる */
  minGrade?: ToolGrade;
}

/** 分解定義。これを持つ prop は殴って壊すと産出を落とす。toolCategory の道具を構えて壊すと多く得る */
model Disassembly {
  toolCategory?: ToolCategory;
  toolRequired?: DisassemblyToolRequired;
  yields: DisassemblyYield[];
  bonus?: DisassemblyBonus[];
}

/** 道具の性質。道具の分類とグレードを持つ */
model Tool {
  categories: ToolCategory[];
  grade: ToolGrade;
}

// ================== 職業 ==================

/** 職業スキル初期値 */
model ProfessionSkill {
  id: SkillId;
  value: SkillLevel;
}

/** 職業初期所持アイテム */
model ProfessionItem {
  name: EntityName;
  count: ItemCount;
}

/** 職業初期装備 */
model ProfessionEquip {
  name: EntityName;
  slot: EquipSlot;
}

/** 職業 */
model Profession {
  id: ProfessionId;
  name: EntityName;
  description: EntityDescription;
  abilities: Abilities;
  skills?: ProfessionSkill[];
  items: ProfessionItem[];
  equips: ProfessionEquip[];
}

// ================== 不調と部位 ==================

/** 不調。痛みと身体機能の低下、回復の仕方、出血を持つ */
model Condition {
  id: ConditionTypeName;
  name: EntityName;
  description: EntityDescription;
  painPerSeverity: ConditionAmountPerSeverity;
  bodyFuncDropPerSeverity: ConditionAmountPerSeverity;

  /** 省くと ConditionSystem の管轄外になる */
  recovery?: RecoveryMode;

  /** progress_until_tend で未治療のとき1ターンに進行度を増やす量 */
  worsenPer?: ConditionTimerRate;

  /** 治療済みで1ターンに進行度を減らす基準量 */
  recoverPer?: ConditionTimerRate;

  bloodDropPerSeverity?: ConditionAmountPerSeverity;
  cause?: ConditionDeathCause;

  /** 起こりうる部位。省くと全部位 */
  parts?: BodyPartId[];

  /** 不調を起こす仕組み。攻撃で負う怪我と、きっかけの無い不調は省く */
  onset?: ConditionOnset;
}

/** 部位。部位の集合と身体機能との対応は Go が持ち、raw は表示名と命中の重みを持つ */
model BodyPart {
  id: BodyPartId;
  name: EntityName;

  /** 戦闘の命中抽選の重み。0 は命中先にならない */
  hitWeight: EntryWeight;
}

// ================== スキルと攻撃種別 ==================

/** スキルが左右する熟練の効果倍率 */
model SkillProficiency {
  id: ProficiencyId;
  perLevel: ProficiencyPerLevel;

  /** この倍率が受けるダメージを軽減する攻撃属性。無属性を除く属性それぞれをちょうど1つの倍率が持つ */
  resists?: Element;
}

/** スキル。同じカテゴリのスキルは行の順に表示する */
model Skill {
  id: SkillId;
  name: EntityName;
  summary: EntityDescription;
  gainedBy: EntityDescription;
  effect: EntityDescription;
  ability: AbilityId;
  category: SkillCategoryId;
  proficiencies?: SkillProficiency[];
}

/** スキルのカテゴリ。行の順に表示する */
model SkillCategory {
  id: SkillCategoryId;
  name: EntityName;
}

/** 攻撃種別。射程などの計算は Go が持ち、raw は命中で負わせる怪我を持つ */
model AttackType {
  id: AttackCategory;
  name: EntityName;
  range: AttackRange;

  /** 遠距離だけが持つ。この距離までは命中が落ちない */
  optimalRange?: RangeTiles;

  /** 遠距離だけが持つ。この距離を超えると撃てない */
  maxRange?: RangeTiles;

  /** 遠距離だけが持つ。最適射程を1タイル超えるごとに下がる命中。百分率 */
  penaltyPerTile?: RangePenaltyPercent;

  injury: ConditionTypeName;

  /** 攻撃で上達するスキル */
  skill: SkillId;

  /** ダメージを決める熟練の効果倍率。skill の行が持つ */
  damage: ProficiencyId;

  /** 命中を決める熟練の効果倍率。skill の行が持つ */
  accuracy: ProficiencyId;
}

// ================== 天候と空腹・疲労 ==================

/** 季節ごとの天候の出やすさ */
model WeatherSeasonWeight {
  id: SeasonId;
  weight: EntryWeight;
}

/** 天候。遷移の粘りの距離は行の順でなく Go の WeatherKind の並びに依る */
model Weather {
  id: WeatherId;
  name: EntityName;
  tempModifier: WeatherTempModifier;
  visionDelta: WeatherVisionDelta;
  severity: WeatherSeverity;
  spellTurnsMin: SpellTurns;
  spellTurnsMax: SpellTurns;
  seasons: WeatherSeasonWeight[];
}

/** 空腹の段階。満腹度の割合が from 以上の段階のうち from が最大のものになる */
model HungerStage {
  id: HungerStageId;
  name: EntityName;
  from: NeedRatio;

  /** 食べてこの段階に達したときのログの msgid。%s は食べた者。省くと出さない */
  reachedLog?: EntityDescription;

  /** 寝ている者を起こす段階か */
  wakesSleeper?: boolean;

  /** HUD に出すバッジの色。省くとバッジを出さない */
  badgeColor?: RGBAColor;

  /** この段階で立つ栄養失調の重症度。省くと立たない */
  severity?: ConditionSeverity;
}

/** 疲労の段階。疲労の割合が from 以上の段階のうち from が最大のものになる */
model FatigueStage {
  id: FatigueStageId;
  name: EntityName;
  from: NeedRatio;

  /** この段階で眠れるか。疲れが足りない段階は眠れない */
  canSleep: boolean;

  /** HUD に出すバッジの色。省くとバッジを出さない */
  badgeColor?: RGBAColor;

  /** この段階で立つ過労の重症度。省くと立たない */
  severity?: ConditionSeverity;
}

// ================== スプライトシート ==================

/** スプライトシート */
model SpriteSheet {
  name: EntityName;
  path: ImagePath;
}

// ================== ローデータ全体 ==================

/** ローデータ全体。TOMLファイルのルート構造を定義する */
model Raws {
  items?: Item[];
  recipes?: Recipe[];
  members?: Member[];
  commandTables?: CommandTable[];
  dropTables?: DropTable[];
  itemGroups?: ItemGroup[];
  itemCollections?: ItemCollection[];
  itemTables?: ItemTable[];
  enemyTables?: EnemyTable[];
  facilities?: Facility[];
  landmarks?: Landmark[];
  scatterZones?: ScatterZone[];
  mapGlyphs?: MapGlyph[];
  interiorContents?: InteriorContent[];
  facilityContents?: FacilityContent[];
  facilityRooms?: FacilityRooms[];

  /** 全施設共通のフレーバー装飾。施設レシピと直交し全室へ一様に重ねる。id は直接引かないので
      interiorContents プールには入れず、この専用フィールドだけに置く */
  flavorContent?: InteriorContent;
  spriteSheets?: SpriteSheet[];
  tiles?: Tile[];
  props?: Prop[];
  professions?: Profession[];
  conditions?: Condition[];
  bodyParts?: BodyPart[];
  skills?: Skill[];
  skillCategories?: SkillCategory[];
  attackTypes?: AttackType[];
  interiorRefs?: InteriorRef[];
  interiorProfile?: InteriorProfile;
  weathers?: Weather[];
  shelterEffects?: ShelterEffect[];
  waterEffects?: WaterEffect[];
  foliageEffects?: FoliageEffect[];
  hungerStages?: HungerStage[];
  fatigueStages?: FatigueStage[];
  freshnessStages?: FreshnessStage[];
  materials?: MaterialDef[];
  toolCategories?: ToolCategoryDef[];
  timesOfDay?: TimeOfDay[];
  seasons?: Season[];
  dungeons?: Dungeon[];
  settlements?: Settlement[];
  dialogues?: Dialogue[];
}

// ================== パレット ==================

/** パレットエントリ */
model PaletteEntry {
  id: PaletteId;
  tile: EntityName;
}

/** パレット */
model Palette {
  id: PaletteId;
  description: EntityDescription;
  terrain: Record<EntityName>;
  props: Record<PaletteEntry>;
  npcs: Record<PaletteEntry>;
}
