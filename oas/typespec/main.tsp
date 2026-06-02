import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "@typespec/openapi";
import "./req.tsp";
import "./resp.tsp";

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

// ================== スカラー型 ==================

/** 命中率 */
scalar Accuracy extends int32;
/** 命中率補正値 */
scalar AccuracyBonus extends int32;
/** 行動コスト */
scalar ActionCost extends int32;
/** 敏捷性。行動順に影響する */
scalar Agility extends int32;
/** 防具の防御力 */
scalar ArmorDefense extends int32;
/** 1ターンあたりの攻撃回数 */
scalar AttackCount extends int32;
/** 基本ダメージ */
scalar BaseDamage extends int32;
/** 通行を妨げるかどうか */
scalar BlocksPassage extends boolean;
/** 視線を遮るかどうか */
scalar BlocksView extends boolean;
/** RGBA色チャネル値 (0-255) */
scalar ColorChannel extends uint8;
/** ダメージ補正値 */
scalar DamageBonus extends int32;
/** 防御力。被ダメージを軽減する */
scalar Defense extends int32;
/** 階層レベル */
scalar DepthLevel extends int32;
/** 器用さ。クリティカル率に影響する */
scalar Dexterity extends int32;
/** 説明文 */
scalar EntityDescription extends string;
/** エンティティ名 */
scalar EntityName extends string;
/** テーブルエントリの重み。大きいほど選ばれやすい */
scalar EntryWeight extends float64;
/** 回復固定量 */
scalar HealAmount extends int32;
/** 回復割合。0.0〜1.0 */
scalar HealRatio extends float64;
/** 画像ファイルパス */
scalar ImagePath extends string;
/** 耐寒性能 */
scalar InsulationCold extends int32;
/** 耐暑性能 */
scalar InsulationHeat extends int32;
/** ボスモンスターかどうか */
scalar IsBoss extends boolean;
/** プレイヤーキャラクターかどうか */
scalar IsPlayer extends boolean;
/** アイテム所持数 */
scalar ItemCount extends int32;
/** 売買価格 */
scalar ItemValue extends int32;
/** アイテム重量 */
scalar ItemWeight extends float64;
/** 光源が有効かどうか */
scalar LightEnabled extends boolean;
/** 光の到達半径 */
scalar LightRadius extends int32;
/** マガジン容量 */
scalar MagazineSize extends int32;
/** 素材必要数 */
scalar MaterialAmount extends int32;
/** メッセージリソースのキー */
scalar MessageKey extends string;
/** 栄養価 */
scalar NutritionAmount extends int32;
/** パレットID */
scalar PaletteId extends string;
/** 職業ID */
scalar ProfessionId extends string;
/** 読了に必要な総読書量 */
scalar ReadingEffort extends int32;
/** リロードに必要な行動力 */
scalar ReloadEffort extends int32;
/** 感覚。命中率と回避率に影響する */
scalar Sensation extends int32;
/** スキルID */
scalar SkillId extends string;
/** スキルレベル */
scalar SkillLevel extends int32;
/** スプライトキー */
scalar SpriteKey extends string;
/** スプライトシート名 */
scalar SpriteSheetName extends string;
/** スタック可能かどうか */
scalar Stackable extends boolean;
/** 筋力。物理ダメージに影響する */
scalar Strength extends int32;
/** AI視界距離（タイル単位） */
scalar ViewDistance extends int32;
/** 体力。HPに影響する */
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
