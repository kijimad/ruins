namespace RuinsEditorApi;

// ================== 一覧レスポンス ==================

/** アイテム一覧レスポンス */
model ItemList {
  data: Item[];
  totalCount: integer;
}

/** メンバー一覧レスポンス */
model MemberList {
  data: Member[];
  totalCount: integer;
}

/** レシピ一覧レスポンス */
model RecipeList {
  data: Recipe[];
  totalCount: integer;
}

/** コマンドテーブル一覧 */
model CommandTableList {
  data: CommandTable[];
  totalCount: integer;
}

/** ドロップテーブル一覧 */
model DropTableList {
  data: DropTable[];
  totalCount: integer;
}

/** アイテムグループ一覧 */
model ItemGroupList {
  data: ItemGroup[];
  totalCount: integer;
}

/** アイテムテーブル一覧 */
model ItemTableList {
  data: ItemTable[];
  totalCount: integer;
}

/** 敵テーブル一覧 */
model EnemyTableList {
  data: EnemyTable[];
  totalCount: integer;
}

/** タイル一覧 */
model TileList {
  data: Tile[];
  totalCount: integer;
}

/** 置物一覧 */
model PropList {
  data: Prop[];
  totalCount: integer;
}

/** 職業一覧 */
model ProfessionList {
  data: Profession[];
  totalCount: integer;
}

/** スプライトシート一覧 */
model SpriteSheetList {
  data: SpriteSheet[];
  totalCount: integer;
}

/** パレット一覧 */
model PaletteList {
  data: Palette[];
  totalCount: integer;
}
