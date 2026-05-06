import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;

@service({
  title: "Ruins Editor API",
})
@server("http://localhost:8080", "Development server")
namespace RuinsEditorApi;

// ================== 共通 ==================

/** エラーレスポンス */
@error
model Error {
  message: string;
}

// ================== アイテム ==================

/** 消費可能アイテムの設定 */
model Consumable {
  usableScene: string;
  targetGroup: string;
  targetNum: string;
}

/** 回復効果 */
model ProvidesHealing {
  valueType: string;
  amount: int32;
  ratio: float64;
}

/** 装備可能設定 */
model Wearable {
  defense: int32;
  equipmentCategory: string;
  insulationCold: int32;
  insulationHeat: int32;
}

/** 装備ボーナス */
model EquipBonus {
  vitality: int32;
  strength: int32;
  sensation: int32;
  dexterity: int32;
  agility: int32;
}

/** 武器マーカー */
model Weapon {}

/** 弾薬設定 */
model Ammo {
  ammoTag: string;
  damageBonus: int32;
  accuracyBonus: int32;
}

/** 近接攻撃設定 */
model Melee {
  accuracy: int32;
  damage: int32;
  attackCount: int32;
  element: string;
  attackCategory: string;
  cost: int32;
  targetGroup: string;
  targetNum: string;
}

/** 遠距離攻撃設定 */
model Fire {
  accuracy: int32;
  damage: int32;
  attackCount: int32;
  element: string;
  attackCategory: string;
  cost: int32;
  targetGroup: string;
  targetNum: string;
  magazineSize: int32;
  reloadEffort: int32;
  ammoTag: string;
}

/** スキル本設定 */
model SkillBook {
  targetSkill: string;
  requiredLevel: int32;
  maxLevel: int32;
}

/** 本の設定 */
model Book {
  totalEffort: int32;
  skill?: SkillBook;
}

/** アイテム */
model Item {
  name: string;
  description: string;
  spriteSheetName: string;
  spriteKey: string;
  animKeys: string[];
  value: int32;
  weight?: float64;
  inflictsDamage?: int32;
  providesNutrition?: int32;
  stackable?: boolean;
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

/** アイテム一覧レスポンス */
model ItemList {
  data: Item[];
  totalCount: int32;
}

// ================== メンバー ==================

/** 能力値 */
model Abilities {
  vitality: int32;
  strength: int32;
  sensation: int32;
  dexterity: int32;
  agility: int32;
  defense: int32;
}

/** 光源設定 */
model LightSource {
  radius: int32;
  color: RGBAColor;
  enabled: boolean;
}

/** RGBA色 */
model RGBAColor {
  r: uint8;
  g: uint8;
  b: uint8;
  a: uint8;
}

/** 会話データ */
model Dialog {
  messageKey: string;
}

/** メンバー */
model Member {
  name: string;
  player?: boolean;
  isBoss: boolean;
  abilities: Abilities;
  spriteSheetName: string;
  spriteKey: string;
  animKeys: string[];
  lightSource?: LightSource;
  factionType: string;
  dialog?: Dialog;
  commandTableName: string;
  dropTableName: string;
}

/** メンバー一覧レスポンス */
model MemberList {
  data: Member[];
  totalCount: int32;
}

// ================== レシピ ==================

/** レシピ素材 */
model RecipeInput {
  name: string;
  amount: int32;
}

/** レシピ */
model Recipe {
  name: string;
  inputs: RecipeInput[];
}

/** レシピ一覧レスポンス */
model RecipeList {
  data: Recipe[];
  totalCount: int32;
}

// ================== テーブル ==================

/** コマンドテーブルエントリ */
model CommandTableEntry {
  weapon: string;
  weight: float64;
}

/** コマンドテーブル */
model CommandTable {
  name: string;
  entries: CommandTableEntry[];
}

/** コマンドテーブル一覧 */
model CommandTableList {
  data: CommandTable[];
  totalCount: int32;
}

/** ドロップテーブルエントリ */
model DropTableEntry {
  material: string;
  weight: float64;
}

/** ドロップテーブル */
model DropTable {
  name: string;
  entries: DropTableEntry[];
}

/** ドロップテーブル一覧 */
model DropTableList {
  data: DropTable[];
  totalCount: int32;
}

/** アイテムテーブルエントリ */
model ItemTableEntry {
  itemName: string;
  weight: float64;
  minDepth: int32;
  maxDepth: int32;
}

/** アイテムテーブル */
model ItemTable {
  name: string;
  entries: ItemTableEntry[];
}

/** アイテムテーブル一覧 */
model ItemTableList {
  data: ItemTable[];
  totalCount: int32;
}

/** 敵テーブルエントリ */
model EnemyTableEntry {
  enemyName: string;
  weight: float64;
  minDepth: int32;
  maxDepth: int32;
}

/** 敵テーブル */
model EnemyTable {
  name: string;
  entries: EnemyTableEntry[];
}

/** 敵テーブル一覧 */
model EnemyTableList {
  data: EnemyTable[];
  totalCount: int32;
}

// ================== タイル ==================

/** スプライトレンダー設定 */
model SpriteRender {
  spriteSheetName: string;
  spriteKey: string;
  depth: int32;
}

/** タイル */
model Tile {
  name: string;
  description: string;
  blockPass: boolean;
  blockView: boolean;
  spriteRender: SpriteRender;
  shelter: int32;
  water: int32;
  foliage: int32;
}

/** タイル一覧 */
model TileList {
  data: Tile[];
  totalCount: int32;
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
  name: string;
  description: string;
  spriteRender: SpriteRender;
  animKeys: string[];
  blockPass: boolean;
  blockView: boolean;
  lightSource?: LightSource;
  door?: DoorRaw;
  doorLockTrigger?: DoorLockTriggerRaw;
  warpNextTrigger?: WarpNextTriggerRaw;
  warpEscapeTrigger?: WarpEscapeTriggerRaw;
  dungeonGateTrigger?: DungeonGateTriggerRaw;
}

/** 置物一覧 */
model PropList {
  data: Prop[];
  totalCount: int32;
}

// ================== 職業 ==================

/** 職業スキル初期値 */
model ProfessionSkill {
  id: string;
  value: int32;
}

/** 職業初期所持アイテム */
model ProfessionItem {
  name: string;
  count: int32;
}

/** 職業初期装備 */
model ProfessionEquip {
  name: string;
  slot: string;
}

/** 職業 */
model Profession {
  id: string;
  name: string;
  description: string;
  abilities: Abilities;
  skills: ProfessionSkill[];
  items: ProfessionItem[];
  equips: ProfessionEquip[];
}

/** 職業一覧 */
model ProfessionList {
  data: Profession[];
  totalCount: int32;
}

// ================== スプライトシート ==================

/** スプライトシート */
model SpriteSheet {
  name: string;
  path: string;
}

/** スプライトシート一覧 */
model SpriteSheetList {
  data: SpriteSheet[];
  totalCount: int32;
}

// ================== パレット ==================

/** パレットエントリ */
model PaletteEntry {
  id: string;
  tile: string;
}

/** パレット */
model Palette {
  id: string;
  description: string;
  terrain: Record<string>;
  props: Record<PaletteEntry>;
  npcs: Record<PaletteEntry>;
}

/** パレット一覧 */
model PaletteList {
  data: Palette[];
  totalCount: int32;
}

// ================== API エンドポイント ==================

@route("/api/v1/items")
interface Items {
  /** アイテム一覧取得 */
  @get list(): ItemList | Error;

  /** アイテム詳細取得 */
  @get @route("/{index}") get(@path index: int32): Item | Error;

  /** アイテム作成 */
  @post create(@body item: Item): {
    @statusCode statusCode: 201;
    @body body: Item;
  } | Error;

  /** アイテム更新 */
  @put @route("/{index}") update(@path index: int32, @body item: Item): Item | Error;

  /** アイテム削除 */
  @delete @route("/{index}") delete(@path index: int32): {
    @statusCode statusCode: 204;
  } | Error;
}

@route("/api/v1/members")
interface Members {
  @get list(): MemberList | Error;
  @get @route("/{index}") get(@path index: int32): Member | Error;
  @post create(@body member: Member): { @statusCode statusCode: 201; @body body: Member; } | Error;
  @put @route("/{index}") update(@path index: int32, @body member: Member): Member | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/recipes")
interface Recipes {
  @get list(): RecipeList | Error;
  @get @route("/{index}") get(@path index: int32): Recipe | Error;
  @post create(@body recipe: Recipe): { @statusCode statusCode: 201; @body body: Recipe; } | Error;
  @put @route("/{index}") update(@path index: int32, @body recipe: Recipe): Recipe | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/command-tables")
interface CommandTables {
  @get list(): CommandTableList | Error;
  @post create(@body table: CommandTable): { @statusCode statusCode: 201; @body body: CommandTable; } | Error;
  @put @route("/{index}") update(@path index: int32, @body table: CommandTable): CommandTable | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/drop-tables")
interface DropTables {
  @get list(): DropTableList | Error;
  @post create(@body table: DropTable): { @statusCode statusCode: 201; @body body: DropTable; } | Error;
  @put @route("/{index}") update(@path index: int32, @body table: DropTable): DropTable | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/item-tables")
interface ItemTables {
  @get list(): ItemTableList | Error;
  @post create(@body table: ItemTable): { @statusCode statusCode: 201; @body body: ItemTable; } | Error;
  @put @route("/{index}") update(@path index: int32, @body table: ItemTable): ItemTable | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/enemy-tables")
interface EnemyTables {
  @get list(): EnemyTableList | Error;
  @post create(@body table: EnemyTable): { @statusCode statusCode: 201; @body body: EnemyTable; } | Error;
  @put @route("/{index}") update(@path index: int32, @body table: EnemyTable): EnemyTable | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/tiles")
interface Tiles {
  @get list(): TileList | Error;
  @get @route("/{index}") get(@path index: int32): Tile | Error;
  @post create(@body tile: Tile): { @statusCode statusCode: 201; @body body: Tile; } | Error;
  @put @route("/{index}") update(@path index: int32, @body tile: Tile): Tile | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/props")
interface Props {
  @get list(): PropList | Error;
  @get @route("/{index}") get(@path index: int32): Prop | Error;
  @post create(@body prop: Prop): { @statusCode statusCode: 201; @body body: Prop; } | Error;
  @put @route("/{index}") update(@path index: int32, @body prop: Prop): Prop | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/professions")
interface Professions {
  @get list(): ProfessionList | Error;
  @post create(@body profession: Profession): { @statusCode statusCode: 201; @body body: Profession; } | Error;
  @put @route("/{index}") update(@path index: int32, @body profession: Profession): Profession | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/sprite-sheets")
interface SpriteSheets {
  @get list(): SpriteSheetList | Error;
  @post create(@body sheet: SpriteSheet): { @statusCode statusCode: 201; @body body: SpriteSheet; } | Error;
  @put @route("/{index}") update(@path index: int32, @body sheet: SpriteSheet): SpriteSheet | Error;
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@route("/api/v1/palettes")
interface Palettes {
  @get list(): PaletteList | Error;
  @get @route("/{id}") get(@path id: string): Palette | Error;
  @post create(@body palette: Palette): { @statusCode statusCode: 201; @body body: Palette; } | Error;
  @put @route("/{id}") update(@path id: string, @body palette: Palette): Palette | Error;
  @delete @route("/{id}") delete(@path id: string): { @statusCode statusCode: 204; } | Error;
}
