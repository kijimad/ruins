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

/** キューブモジュール設定。装着すると展開野営の範囲を縦横一律に広げる */
model CubeModule {
  rangeBonus: RangeBonus;
}

/** 据付設定。キューブの展開空間へ据えるとフィールドで設備として働くアイテムであることを表す */
model Deployable {
  /** 収納設備の容量。据えると収納として開ける */
  storage?: StorageRaw;
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
  disassembly?: Disassembly;
  disassemblyTool?: DisassemblyTool;
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
  weapon: EntityID;
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
  /** ドロップする素材名。空文字はドロップなしを意味する */
  material: string;
  weight: EntryWeight;
}

/** ドロップテーブル */
model DropTable {
  id: EntityID;
  name: EntityName;
  entries: DropTableEntry[];
}

/**
 * アイテムグループのサブタイプ
 */
enum ItemGroupSubtype {
  /** エントリ群から重み比率に基づいて1つだけ選ぶ。weightは相対比率として扱う */
  distribution,
  /** 各エントリを独立に確率判定する。weightは0-100の出現確率(%)として扱う。複数当選することもありうる */
  collection,
}

/** アイテムグループエントリ */
model ItemGroupEntry {
  id: EntityID;
  /** distribution: 相対重み。collection: 確率（0-100） */
  weight: EntryWeight;
  /** 1山あたりの個数のダイス表記 */
  pack: Dice;
}

/** アイテムグループ。アイテムの出現セットを定義する */
model ItemGroup {
  id: EntityID;
  name: EntityName;
  subtype: ItemGroupSubtype;
  entries: ItemGroupEntry[];
}

/** アイテムテーブルエントリ。アイテムグループを参照する */
model ItemTableEntry {
  /** 参照するアイテムグループの id */
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

/** 施設種別ごとの敵テーブル割り当て。市街地生成が施設で敵テーブルを切り替える。似た施設は同じ
enemyTable を指してよい。未割り当ての施設は生成側の既定テーブルへ落ちる。 */
model FacilityEnemyTable {
  facility: FacilityKind;
  /** 割り当てる敵テーブルの id。enemyTables のいずれかを指す */
  enemyTable: EntityID;
}

// ================== 内装レシピ ==================

/** 配置指示の種別。家具・戦利品・敵・装飾・罠を同じ器で扱う */
enum StuffKind {
  furniture,
  loot,
  being,
  decor,
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

/** 配置の置き方。空なら家具型の archetype 既定へ落ちる */
enum Placement {
  center,
  wall,
  full_area,
  near_door,
  far_from_door,
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
}

/** 内装レシピの1配置指示 */
model ContentStuff {
  kind: StuffKind;
  ref: EntityID;
  weight?: EntryWeight;
  /** 0..100。pick_each でこの Stuff を置く確率。省略時は常置 */
  chance?: int32;
  /** 置く個数のダイス表記 */
  amount: Dice;
  placement?: Placement;
  satellites?: ContentSatellite[];
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

/** 施設種別ごとの主室の内装変種。抽選で1つ選ぶ */
model FacilityContent {
  facility: FacilityKind;
  variants: EntityID[];
}

/** 奥室の役割名と内装レシピの対 */
model RoomContent {
  role: RoomRole;
  content: EntityID;
}

/** 施設種別ごとの奥室カタログ。役割別 content と、カタログに無い役割のフォールバック */
model FacilityRooms {
  facility: FacilityKind;
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
  shelter: ShelterType;
  water: WaterType;
  foliage: FoliageType;
}

// ================== 置物 ==================

/** 扉ローデータ */
model DoorRaw {}

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
  hp: HitPoints;
  lightSource?: LightSource;
  heatSource?: HeatSource;
  bedding?: Bedding;
  door?: DoorRaw;
  warpNextTrigger?: WarpNextTriggerRaw;
  warpPrevTrigger?: WarpPrevTriggerRaw;
  storage?: StorageRaw;
  disassembly?: Disassembly;
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

/** 分解定義。これを持つ prop・item は対応工具で分解できる */
model Disassembly {
  toolCategory: ToolCategory;
  baseAP: DisassemblyBaseAP;
  yields: DisassemblyYield[];
  bonus?: DisassemblyBonus[];
}

/** 分解工具。分類の対応とグレードを持つ */
model DisassemblyTool {
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
  itemTables?: ItemTable[];
  enemyTables?: EnemyTable[];
  facilityEnemyTables?: FacilityEnemyTable[];
  interiorContents?: InteriorContent[];
  facilityContents?: FacilityContent[];
  facilityRooms?: FacilityRooms[];
  spriteSheets?: SpriteSheet[];
  tiles?: Tile[];
  props?: Prop[];
  professions?: Profession[];
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
