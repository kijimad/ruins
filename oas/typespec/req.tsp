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

/** 武器マーカー */
model Weapon {}

/** 弾薬設定 */
model Ammo {
  ammoTag: AmmoTag;
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
  ammoTag: AmmoTag;
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
  name: EntityName;
  description: EntityDescription;
  spriteSheetName: SpriteSheetName;
  spriteKey: SpriteKey;
  animKeys: SpriteKey[];
  value: ItemValue;
  weight?: ItemWeight;
  inflictsDamage?: BaseDamage;
  providesNutrition?: NutritionAmount;
  stackable?: Stackable;
  consumable?: Consumable;
  providesHealing?: ProvidesHealing;
  wearable?: Wearable;
  equipBonus?: EquipBonus;
  weapon?: Weapon;
  ammo?: Ammo;
  melee?: Melee;
  fire?: Fire;
  book?: Book;
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

/** 会話データ */
model Dialog {
  messageKey: MessageKey;
}

/** メンバー */
model Member {
  name: EntityName;
  player?: IsPlayer;
  isBoss: IsBoss;
  abilities: Abilities;
  spriteSheetName: SpriteSheetName;
  spriteKey: SpriteKey;
  animKeys: SpriteKey[];
  lightSource?: LightSource;
  factionType?: FactionMemberType;
  disposition?: DispositionType;
  movementPattern?: MovementPatternType;
  viewDistance?: ViewDistance;
  dialog?: Dialog;
  commandTableName: EntityName;
  dropTableName: EntityName;
}

// ================== レシピ ==================

/** レシピ素材 */
model RecipeInput {
  name: EntityName;
  amount: MaterialAmount;
}

/** レシピ */
model Recipe {
  name: EntityName;
  inputs: RecipeInput[];
}

// ================== テーブル ==================

/** コマンドテーブルエントリ */
model CommandTableEntry {
  weapon: EntityName;
  weight: EntryWeight;
}

/** コマンドテーブル */
model CommandTable {
  name: EntityName;
  entries: CommandTableEntry[];
}

/** ドロップテーブルエントリ */
model DropTableEntry {
  material: EntityName;
  weight: EntryWeight;
}

/** ドロップテーブル */
model DropTable {
  name: EntityName;
  entries: DropTableEntry[];
}

/** アイテムテーブルエントリ */
model ItemTableEntry {
  itemName: EntityName;
  weight: EntryWeight;
  minDepth: DepthLevel;
  maxDepth: DepthLevel;
}

/** アイテムテーブル */
model ItemTable {
  name: EntityName;
  entries: ItemTableEntry[];
}

/** 敵テーブルエントリ */
model EnemyTableEntry {
  enemyName: EntityName;
  weight: EntryWeight;
  minDepth: DepthLevel;
  maxDepth: DepthLevel;
}

/** 敵テーブル */
model EnemyTable {
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

/** 扉ロックトリガー */
model DoorLockTriggerRaw {}

/** 次階層ワープトリガー */
model WarpNextTriggerRaw {}

/** 脱出ワープトリガー */
model WarpEscapeTriggerRaw {}

/** ダンジョン選択ゲートトリガー */
model DungeonGateTriggerRaw {}

/** 置物 */
model Prop {
  name: EntityName;
  description: EntityDescription;
  spriteRender: SpriteRender;
  animKeys: SpriteKey[];
  blockPass: BlocksPassage;
  blockView: BlocksView;
  lightSource?: LightSource;
  door?: DoorRaw;
  doorLockTrigger?: DoorLockTriggerRaw;
  warpNextTrigger?: WarpNextTriggerRaw;
  warpEscapeTrigger?: WarpEscapeTriggerRaw;
  dungeonGateTrigger?: DungeonGateTriggerRaw;
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
  skills: ProfessionSkill[];
  items: ProfessionItem[];
  equips: ProfessionEquip[];
}

// ================== スプライトシート ==================

/** スプライトシート */
model SpriteSheet {
  name: EntityName;
  path: ImagePath;
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
