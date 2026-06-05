namespace RuinsEditorApi;

// ================== 一覧レスポンス ==================

/** アイテム一覧レスポンス */
model ItemList {
  data: Item[];
  totalCount: int32;
}

/** メンバー一覧レスポンス */
model MemberList {
  data: Member[];
  totalCount: int32;
}

/** レシピ一覧レスポンス */
model RecipeList {
  data: Recipe[];
  totalCount: int32;
}

/** コマンドテーブル一覧 */
model CommandTableList {
  data: CommandTable[];
  totalCount: int32;
}

/** ドロップテーブル一覧 */
model DropTableList {
  data: DropTable[];
  totalCount: int32;
}

/** アイテムテーブル一覧 */
model ItemTableList {
  data: ItemTable[];
  totalCount: int32;
}

/** 敵テーブル一覧 */
model EnemyTableList {
  data: EnemyTable[];
  totalCount: int32;
}

/** タイル一覧 */
model TileList {
  data: Tile[];
  totalCount: int32;
}

/** 置物一覧 */
model PropList {
  data: Prop[];
  totalCount: int32;
}

/** 職業一覧 */
model ProfessionList {
  data: Profession[];
  totalCount: int32;
}

/** スプライトシート一覧 */
model SpriteSheetList {
  data: SpriteSheet[];
  totalCount: int32;
}

/** パレット一覧 */
model PaletteList {
  data: Palette[];
  totalCount: int32;
}
