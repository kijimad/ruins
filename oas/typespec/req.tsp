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

/** 移動拠点キューブの内部からの退場トリガー */
model WarpCubeExitTriggerRaw {}

/** 移動拠点キューブのコントロールパネルトリガー */
model CubePanelTriggerRaw {}

/** 通信販売の出荷場所ローデータ。収納の中身を集荷対象にし、出荷場所メニューを開く相互作用が付く。積載量は storage で持つ */
model ShippingStationRaw {}

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
  hp?: HitPoints;
  lightSource?: LightSource;
  heatSource?: HeatSource;
  bedding?: Bedding;
  door?: DoorRaw;
  warpNextTrigger?: WarpNextTriggerRaw;
  warpPrevTrigger?: WarpPrevTriggerRaw;
  warpCubeExitTrigger?: WarpCubeExitTriggerRaw;
  cubePanelTrigger?: CubePanelTriggerRaw;
  storage?: StorageRaw;
  shippingStation?: ShippingStationRaw;
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
