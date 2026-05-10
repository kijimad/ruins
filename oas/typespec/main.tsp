import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

@service({
  title: "Ruins Editor API",
})
@server("http://localhost:8080", "Development server")
namespace RuinsEditorApi;

// ================== 共通 ==================

/** エラーレスポンス */
@error
model Error {
  /** エラーメッセージ */
  message: string;
}

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

// ================== アイテム ==================

/** 消費可能アイテムの設定 */
model Consumable {
  /** 使用可能なシーン */
  usableScene: UsableScene;
  /** 効果の対象グループ */
  targetGroup: TargetGroup;
  /** 効果の対象数 */
  targetNum: TargetNum;
}

/** 回復量の計算方式 */
enum HealingValueType {
  /** 最大値に対する割合で回復 */
  PERCENTAGE,
}

/** 回復効果 */
model ProvidesHealing {
  /** 回復量の計算方式 */
  valueType: HealingValueType;
  /** 回復固定量 */
  amount: int32;
  /** 回復割合。0.0〜1.0 */
  ratio: float64;
}

/** 装備可能設定 */
model Wearable {
  /** 防御力 */
  defense: int32;
  /** 装備する部位 */
  equipmentCategory: EquipmentCategory;
  /** 耐寒性能 */
  insulationCold: int32;
  /** 耐暑性能 */
  insulationHeat: int32;
}

/** 装備ボーナス */
model EquipBonus {
  /** 体力ボーナス */
  vitality: int32;
  /** 筋力ボーナス */
  strength: int32;
  /** 感覚ボーナス */
  sensation: int32;
  /** 器用さボーナス */
  dexterity: int32;
  /** 敏捷性ボーナス */
  agility: int32;
}

/** 武器マーカー */
model Weapon {}

/** 弾薬設定 */
model Ammo {
  /** 弾薬の種類タグ */
  ammoTag: AmmoTag;
  /** ダメージ補正値 */
  damageBonus: int32;
  /** 命中率補正値 */
  accuracyBonus: int32;
}

/** 近接攻撃設定 */
model Melee {
  /** 命中率 */
  accuracy: int32;
  /** ダメージ */
  damage: int32;
  /** 1ターンあたりの攻撃回数 */
  attackCount: int32;
  /** 攻撃属性 */
  element: Element;
  /** 攻撃種別 */
  attackCategory: AttackCategory;
  /** 行動コスト */
  cost: int32;
  /** 対象グループ */
  targetGroup: TargetGroup;
  /** 対象数 */
  targetNum: TargetNum;
}

/** 遠距離攻撃設定 */
model Fire {
  /** 命中率 */
  accuracy: int32;
  /** ダメージ */
  damage: int32;
  /** 1ターンあたりの攻撃回数 */
  attackCount: int32;
  /** 攻撃属性 */
  element: Element;
  /** 攻撃種別 */
  attackCategory: AttackCategory;
  /** 行動コスト */
  cost: int32;
  /** 対象グループ */
  targetGroup: TargetGroup;
  /** 対象数 */
  targetNum: TargetNum;
  /** マガジン容量 */
  magazineSize: int32;
  /** リロードに必要な行動力 */
  reloadEffort: int32;
  /** 使用する弾薬の種類 */
  ammoTag: AmmoTag;
}

/** スキル本設定 */
model SkillBook {
  /** 習得対象のスキルID */
  targetSkill: string;
  /** 習得に必要なスキルレベル */
  requiredLevel: int32;
  /** 習得可能な最大レベル */
  maxLevel: int32;
}

/** 本の設定 */
model Book {
  /** 読了に必要な総読書量 */
  totalEffort: int32;
  /** スキル本としての設定。未設定なら娯楽本 */
  skill?: SkillBook;
}

/** アイテム */
model Item {
  /** アイテム名 */
  name: string;
  /** アイテムの説明文 */
  description: string;
  /** スプライトシート名 */
  spriteSheetName: string;
  /** スプライトキー */
  spriteKey: string;
  /** アニメーション用スプライトキーの配列 */
  animKeys: string[];
  /** 売買価格 */
  value: int32;
  /** 重量。未設定なら0 */
  weight?: float64;
  /** 投擲ダメージ。未設定なら投擲不可 */
  inflictsDamage?: int32;
  /** 栄養価。未設定なら食べられない */
  providesNutrition?: int32;
  /** スタック可能かどうか */
  stackable?: boolean;
  /** 消費アイテム設定 */
  consumable?: Consumable;
  /** 回復効果 */
  providesHealing?: ProvidesHealing;
  /** 防具設定 */
  wearable?: Wearable;
  /** 装備ボーナス */
  equipBonus?: EquipBonus;
  /** 武器マーカー。存在すれば武器として扱う */
  weapon?: Weapon;
  /** 弾薬設定 */
  ammo?: Ammo;
  /** 近接攻撃設定 */
  melee?: Melee;
  /** 遠距離攻撃設定 */
  fire?: Fire;
  /** 本設定 */
  book?: Book;
}

/** アイテム一覧レスポンス */
model ItemList {
  /** アイテムの配列 */
  data: Item[];
  /** 総件数 */
  totalCount: int32;
}

// ================== メンバー ==================

/** 能力値 */
model Abilities {
  /** 体力。HPに影響する */
  vitality: int32;
  /** 筋力。物理ダメージに影響する */
  strength: int32;
  /** 感覚。命中率と回避率に影響する */
  sensation: int32;
  /** 器用さ。クリティカル率に影響する */
  dexterity: int32;
  /** 敏捷性。行動順に影響する */
  agility: int32;
  /** 防御力。被ダメージを軽減する */
  defense: int32;
}

/** 光源設定 */
model LightSource {
  /** 光の到達半径 */
  radius: int32;
  /** 光の色 */
  color: RGBAColor;
  /** 光源が有効かどうか */
  enabled: boolean;
}

/** RGBA色 */
model RGBAColor {
  /** 赤 (0-255) */
  r: uint8;
  /** 緑 (0-255) */
  g: uint8;
  /** 青 (0-255) */
  b: uint8;
  /** 透明度 (0-255) */
  a: uint8;
}

/** 会話データ */
model Dialog {
  /** メッセージリソースのキー */
  messageKey: string;
}

/** メンバー */
model Member {
  /** メンバー名 */
  name: string;
  /** プレイヤーキャラクターかどうか */
  player?: boolean;
  /** ボスモンスターかどうか */
  isBoss: boolean;
  /** 基本能力値 */
  abilities: Abilities;
  /** スプライトシート名 */
  spriteSheetName: string;
  /** スプライトキー */
  spriteKey: string;
  /** アニメーション用スプライトキーの配列 */
  animKeys: string[];
  /** 光源設定。未設定なら光源なし */
  lightSource?: LightSource;
  /** 所属する派閥。未設定なら敵対 */
  factionType?: FactionMemberType;
  /** 会話データ。未設定なら会話不可 */
  dialog?: Dialog;
  /** 参照するコマンドテーブル名 */
  commandTableName: string;
  /** 参照するドロップテーブル名 */
  dropTableName: string;
}

/** メンバー一覧レスポンス */
model MemberList {
  /** メンバーの配列 */
  data: Member[];
  /** 総件数 */
  totalCount: int32;
}

// ================== レシピ ==================

/** レシピ素材 */
model RecipeInput {
  /** 素材アイテム名 */
  name: string;
  /** 必要個数 */
  amount: int32;
}

/** レシピ */
model Recipe {
  /** レシピ名。生成されるアイテム名と一致する */
  name: string;
  /** 必要素材のリスト */
  inputs: RecipeInput[];
}

/** レシピ一覧レスポンス */
model RecipeList {
  /** レシピの配列 */
  data: Recipe[];
  /** 総件数 */
  totalCount: int32;
}

// ================== テーブル ==================

/** コマンドテーブルエントリ */
model CommandTableEntry {
  /** 使用する武器名 */
  weapon: string;
  /** 選択される重み。大きいほど選ばれやすい */
  weight: float64;
}

/** コマンドテーブル */
model CommandTable {
  /** テーブル名 */
  name: string;
  /** コマンドエントリのリスト */
  entries: CommandTableEntry[];
}

/** コマンドテーブル一覧 */
model CommandTableList {
  /** コマンドテーブルの配列 */
  data: CommandTable[];
  /** 総件数 */
  totalCount: int32;
}

/** ドロップテーブルエントリ */
model DropTableEntry {
  /** ドロップする素材名 */
  material: string;
  /** ドロップする重み。大きいほど選ばれやすい */
  weight: float64;
}

/** ドロップテーブル */
model DropTable {
  /** テーブル名 */
  name: string;
  /** ドロップエントリのリスト */
  entries: DropTableEntry[];
}

/** ドロップテーブル一覧 */
model DropTableList {
  /** ドロップテーブルの配列 */
  data: DropTable[];
  /** 総件数 */
  totalCount: int32;
}

/** アイテムテーブルエントリ */
model ItemTableEntry {
  /** 出現するアイテム名 */
  itemName: string;
  /** 出現する重み。大きいほど選ばれやすい */
  weight: float64;
  /** 出現する最小階層 */
  minDepth: int32;
  /** 出現する最大階層 */
  maxDepth: int32;
}

/** アイテムテーブル */
model ItemTable {
  /** テーブル名 */
  name: string;
  /** アイテムエントリのリスト */
  entries: ItemTableEntry[];
}

/** アイテムテーブル一覧 */
model ItemTableList {
  /** アイテムテーブルの配列 */
  data: ItemTable[];
  /** 総件数 */
  totalCount: int32;
}

/** 敵テーブルエントリ */
model EnemyTableEntry {
  /** 出現する敵メンバー名 */
  enemyName: string;
  /** 出現する重み。大きいほど選ばれやすい */
  weight: float64;
  /** 出現する最小階層 */
  minDepth: int32;
  /** 出現する最大階層 */
  maxDepth: int32;
}

/** 敵テーブル */
model EnemyTable {
  /** テーブル名 */
  name: string;
  /** 敵エントリのリスト */
  entries: EnemyTableEntry[];
}

/** 敵テーブル一覧 */
model EnemyTableList {
  /** 敵テーブルの配列 */
  data: EnemyTable[];
  /** 総件数 */
  totalCount: int32;
}

// ================== タイル ==================

/** スプライトレンダー設定 */
model SpriteRender {
  /** スプライトシート名 */
  spriteSheetName: string;
  /** スプライトキー */
  spriteKey: string;
  /** 描画深度。レイヤー順序を決定する */
  depth: SpriteDepth;
}

/** タイル */
model Tile {
  /** タイル名 */
  name: string;
  /** タイルの説明文 */
  description: string;
  /** 通行を妨げるかどうか */
  blockPass: boolean;
  /** 視線を遮るかどうか */
  blockView: boolean;
  /** スプライト描画設定 */
  spriteRender: SpriteRender;
  /** 遮蔽タイプ。戦闘時の命中率に影響する */
  shelter: ShelterType;
  /** 水タイプ。移動コストに影響する */
  water: WaterType;
  /** 植生タイプ。視界に影響する */
  foliage: FoliageType;
}

/** タイル一覧 */
model TileList {
  /** タイルの配列 */
  data: Tile[];
  /** 総件数 */
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
  /** 置物名 */
  name: string;
  /** 置物の説明文 */
  description: string;
  /** スプライト描画設定 */
  spriteRender: SpriteRender;
  /** アニメーション用スプライトキーの配列 */
  animKeys: string[];
  /** 通行を妨げるかどうか */
  blockPass: boolean;
  /** 視線を遮るかどうか */
  blockView: boolean;
  /** 光源設定。未設定なら光源なし */
  lightSource?: LightSource;
  /** 扉データ。設定すると開閉可能な扉として動作する */
  door?: DoorRaw;
  /** 扉ロックトリガー。設定すると鍵付き扉として動作する */
  doorLockTrigger?: DoorLockTriggerRaw;
  /** 次階層ワープトリガー。踏むと次の階層に進む */
  warpNextTrigger?: WarpNextTriggerRaw;
  /** 脱出ワープトリガー。踏むとダンジョンから脱出する */
  warpEscapeTrigger?: WarpEscapeTriggerRaw;
  /** ダンジョン選択ゲートトリガー。調べるとダンジョン選択画面を開く */
  dungeonGateTrigger?: DungeonGateTriggerRaw;
}

/** 置物一覧 */
model PropList {
  /** 置物の配列 */
  data: Prop[];
  /** 総件数 */
  totalCount: int32;
}

// ================== 職業 ==================

/** 職業スキル初期値 */
model ProfessionSkill {
  /** スキルID */
  id: string;
  /** 初期スキルレベル */
  value: int32;
}

/** 職業初期所持アイテム */
model ProfessionItem {
  /** アイテム名 */
  name: string;
  /** 所持数 */
  count: int32;
}

/** 職業初期装備 */
model ProfessionEquip {
  /** 装備アイテム名 */
  name: string;
  /** 装備するスロット */
  slot: EquipSlot;
}

/** 職業 */
model Profession {
  /** 職業ID */
  id: string;
  /** 職業名 */
  name: string;
  /** 職業の説明文 */
  description: string;
  /** 基本能力値 */
  abilities: Abilities;
  /** 初期スキルのリスト */
  skills: ProfessionSkill[];
  /** 初期所持アイテムのリスト */
  items: ProfessionItem[];
  /** 初期装備のリスト */
  equips: ProfessionEquip[];
}

/** 職業一覧 */
model ProfessionList {
  /** 職業の配列 */
  data: Profession[];
  /** 総件数 */
  totalCount: int32;
}

// ================== スプライトシート ==================

/** スプライトシート */
model SpriteSheet {
  /** シート名 */
  name: string;
  /** 画像ファイルパス */
  path: string;
}

/** スプライトシート一覧 */
model SpriteSheetList {
  /** スプライトシートの配列 */
  data: SpriteSheet[];
  /** 総件数 */
  totalCount: int32;
}

// ================== パレット ==================

/** パレットエントリ */
model PaletteEntry {
  /** エントリID */
  id: string;
  /** 対応するタイル名 */
  tile: string;
}

/** パレット */
model Palette {
  /** パレットID */
  id: string;
  /** パレットの説明文 */
  description: string;
  /** 地形マッピング。マップ文字からタイル名への対応 */
  terrain: Record<string>;
  /** 置物マッピング。マップ文字から置物エントリへの対応 */
  props: Record<PaletteEntry>;
  /** NPCマッピング。マップ文字からNPCエントリへの対応 */
  npcs: Record<PaletteEntry>;
}

/** パレット一覧 */
model PaletteList {
  /** パレットの配列 */
  data: Palette[];
  /** 総件数 */
  totalCount: int32;
}

// ================== API エンドポイント ==================

@tag("Items")
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

@tag("Members")
@route("/api/v1/members")
interface Members {
  /** メンバー一覧取得 */
  @get list(): MemberList | Error;
  /** メンバー詳細取得 */
  @get @route("/{index}") get(@path index: int32): Member | Error;
  /** メンバー作成 */
  @post create(@body member: Member): { @statusCode statusCode: 201; @body body: Member; } | Error;
  /** メンバー更新 */
  @put @route("/{index}") update(@path index: int32, @body member: Member): Member | Error;
  /** メンバー削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("Recipes")
@route("/api/v1/recipes")
interface Recipes {
  /** レシピ一覧取得 */
  @get list(): RecipeList | Error;
  /** レシピ詳細取得 */
  @get @route("/{index}") get(@path index: int32): Recipe | Error;
  /** レシピ作成 */
  @post create(@body recipe: Recipe): { @statusCode statusCode: 201; @body body: Recipe; } | Error;
  /** レシピ更新 */
  @put @route("/{index}") update(@path index: int32, @body recipe: Recipe): Recipe | Error;
  /** レシピ削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("CommandTables")
@route("/api/v1/command-tables")
interface CommandTables {
  /** コマンドテーブル一覧取得 */
  @get list(): CommandTableList | Error;
  /** コマンドテーブル作成 */
  @post create(@body table: CommandTable): { @statusCode statusCode: 201; @body body: CommandTable; } | Error;
  /** コマンドテーブル更新 */
  @put @route("/{index}") update(@path index: int32, @body table: CommandTable): CommandTable | Error;
  /** コマンドテーブル削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("DropTables")
@route("/api/v1/drop-tables")
interface DropTables {
  /** ドロップテーブル一覧取得 */
  @get list(): DropTableList | Error;
  /** ドロップテーブル作成 */
  @post create(@body table: DropTable): { @statusCode statusCode: 201; @body body: DropTable; } | Error;
  /** ドロップテーブル更新 */
  @put @route("/{index}") update(@path index: int32, @body table: DropTable): DropTable | Error;
  /** ドロップテーブル削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("ItemTables")
@route("/api/v1/item-tables")
interface ItemTables {
  /** アイテムテーブル一覧取得 */
  @get list(): ItemTableList | Error;
  /** アイテムテーブル作成 */
  @post create(@body table: ItemTable): { @statusCode statusCode: 201; @body body: ItemTable; } | Error;
  /** アイテムテーブル更新 */
  @put @route("/{index}") update(@path index: int32, @body table: ItemTable): ItemTable | Error;
  /** アイテムテーブル削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("EnemyTables")
@route("/api/v1/enemy-tables")
interface EnemyTables {
  /** 敵テーブル一覧取得 */
  @get list(): EnemyTableList | Error;
  /** 敵テーブル作成 */
  @post create(@body table: EnemyTable): { @statusCode statusCode: 201; @body body: EnemyTable; } | Error;
  /** 敵テーブル更新 */
  @put @route("/{index}") update(@path index: int32, @body table: EnemyTable): EnemyTable | Error;
  /** 敵テーブル削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("Tiles")
@route("/api/v1/tiles")
interface Tiles {
  /** タイル一覧取得 */
  @get list(): TileList | Error;
  /** タイル詳細取得 */
  @get @route("/{index}") get(@path index: int32): Tile | Error;
  /** タイル作成 */
  @post create(@body tile: Tile): { @statusCode statusCode: 201; @body body: Tile; } | Error;
  /** タイル更新 */
  @put @route("/{index}") update(@path index: int32, @body tile: Tile): Tile | Error;
  /** タイル削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("Props")
@route("/api/v1/props")
interface Props {
  /** 置物一覧取得 */
  @get list(): PropList | Error;
  /** 置物詳細取得 */
  @get @route("/{index}") get(@path index: int32): Prop | Error;
  /** 置物作成 */
  @post create(@body prop: Prop): { @statusCode statusCode: 201; @body body: Prop; } | Error;
  /** 置物更新 */
  @put @route("/{index}") update(@path index: int32, @body prop: Prop): Prop | Error;
  /** 置物削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("Professions")
@route("/api/v1/professions")
interface Professions {
  /** 職業一覧取得 */
  @get list(): ProfessionList | Error;
  /** 職業作成 */
  @post create(@body profession: Profession): { @statusCode statusCode: 201; @body body: Profession; } | Error;
  /** 職業更新 */
  @put @route("/{index}") update(@path index: int32, @body profession: Profession): Profession | Error;
  /** 職業削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("SpriteSheets")
@route("/api/v1/sprite-sheets")
interface SpriteSheets {
  /** スプライトシート一覧取得 */
  @get list(): SpriteSheetList | Error;
  /** スプライトシート作成 */
  @post create(@body sheet: SpriteSheet): { @statusCode statusCode: 201; @body body: SpriteSheet; } | Error;
  /** スプライトシート更新 */
  @put @route("/{index}") update(@path index: int32, @body sheet: SpriteSheet): SpriteSheet | Error;
  /** スプライトシート削除 */
  @delete @route("/{index}") delete(@path index: int32): { @statusCode statusCode: 204; } | Error;
}

@tag("Palettes")
@route("/api/v1/palettes")
interface Palettes {
  /** パレット一覧取得 */
  @get list(): PaletteList | Error;
  /** パレット詳細取得 */
  @get @route("/{id}") get(@path id: string): Palette | Error;
  /** パレット作成 */
  @post create(@body palette: Palette): { @statusCode statusCode: 201; @body body: Palette; } | Error;
  /** パレット更新 */
  @put @route("/{id}") update(@path id: string, @body palette: Palette): Palette | Error;
  /** パレット削除 */
  @delete @route("/{id}") delete(@path id: string): { @statusCode statusCode: 204; } | Error;
}
