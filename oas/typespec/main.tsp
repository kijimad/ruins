import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";
import "@typespec/openapi";
import "./types.tsp";
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

@tag("ItemGroups")
@route("/api/v1/item-groups")
interface ItemGroups {
  /** アイテムグループ一覧取得 */
  @get list(): ItemGroupList | Error;
  /** アイテムグループ作成 */
  @post create(@body group: ItemGroup): { @statusCode statusCode: 201; @body body: ItemGroup; } | Error;
  /** アイテムグループ更新 */
  @put @route("/{index}") update(@path index: int32, @body group: ItemGroup): ItemGroup | Error;
  /** アイテムグループ削除 */
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
