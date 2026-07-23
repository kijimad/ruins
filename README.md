![coverage](https://kijimad.github.io/ruins/cov/coverage.svg)

![Ruins](docs/steam/generated/library_header.png)

ローグライク。

- [Steam](https://store.steampowered.com/app/4791810/Ruins/)
- [Play](https://kijimad.github.io/ruins/)
- [Test Report](https://kijimad.github.io/ruins/cov/)
- [Raw Spec](https://kijimad.github.io/ruins/raw-spec/)
- [Godoc](https://pkg.go.dev/github.com/kijimaD/ruins)

## Play Images

| | | | |
|---|---|---|---|
| <img src="internal/states/testdata/TestGolden_CharacterJob.png" width="200" /><br>CharacterJob | <img src="internal/states/testdata/TestGolden_CharacterNaming.png" width="200" /><br>CharacterNaming | <img src="internal/states/testdata/TestGolden_ComponentDebug.png" width="200" /><br>ComponentDebug | <img src="internal/states/testdata/TestGolden_CraftMenu.png" width="200" /><br>CraftMenu |
| <img src="internal/states/testdata/TestGolden_DebugMenu.png" width="200" /><br>DebugMenu | <img src="internal/states/testdata/TestGolden_Dungeon.png" width="200" /><br>Dungeon | <img src="internal/states/testdata/TestGolden_DungeonSelect.png" width="200" /><br>DungeonSelect | <img src="internal/states/testdata/TestGolden_EquipMenu.png" width="200" /><br>EquipMenu |
| <img src="internal/states/testdata/TestGolden_FormationMenu.png" width="200" /><br>FormationMenu | <img src="internal/states/testdata/TestGolden_GameOver.png" width="200" /><br>GameOver | <img src="internal/states/testdata/TestGolden_InventoryMenu.png" width="200" /><br>InventoryMenu | <img src="internal/states/testdata/TestGolden_LanguageMenu.png" width="200" /><br>LanguageMenu |
| <img src="internal/states/testdata/TestGolden_LoadMenu.png" width="200" /><br>LoadMenu | <img src="internal/states/testdata/TestGolden_LookAround.png" width="200" /><br>LookAround | <img src="internal/states/testdata/TestGolden_MainMenu.png" width="200" /><br>MainMenu | <img src="internal/states/testdata/TestGolden_MemberStatus.png" width="200" /><br>MemberStatus |
| <img src="internal/states/testdata/TestGolden_Message.png" width="200" /><br>Message | <img src="internal/states/testdata/TestGolden_Overworld.png" width="200" /><br>Overworld | <img src="internal/states/testdata/TestGolden_OverworldFrost.png" width="200" /><br>OverworldFrost | <img src="internal/states/testdata/TestGolden_PersistentMessage.png" width="200" /><br>PersistentMessage |
| <img src="internal/states/testdata/TestGolden_Pickup.png" width="200" /><br>Pickup | <img src="internal/states/testdata/TestGolden_Place.png" width="200" /><br>Place | <img src="internal/states/testdata/TestGolden_SaveMenu.png" width="200" /><br>SaveMenu | <img src="internal/states/testdata/TestGolden_SettingsMenu.png" width="200" /><br>SettingsMenu |
| <img src="internal/states/testdata/TestGolden_Shooting.png" width="200" /><br>Shooting | <img src="internal/states/testdata/TestGolden_ShopMenu.png" width="200" /><br>ShopMenu | <img src="internal/states/testdata/TestGolden_SquadMenu.png" width="200" /><br>SquadMenu | <img src="internal/states/testdata/TestGolden_Status.png" width="200" /><br>Status |
| <img src="internal/states/testdata/TestGolden_StorageMenu.png" width="200" /><br>StorageMenu | <img src="internal/states/testdata/TestGolden_TavernMenu.png" width="200" /><br>TavernMenu | <img src="internal/states/testdata/TestGolden_Town.png" width="200" /><br>Town | |


各画像はゴールデンテストで自動生成される。

## キーボード操作

### ダンジョン探索
- **W** - 上
- **S** - 下
- **A** - 左
- **D** - 右
- **C / PageDown** - ズームアウト
- **E / PageUp** - ズームイン
- **マウスホイール** - ズーム操作

### メニューナビゲーション
- **↑ / ↓** - 項目の上下移動
- **← / →** - グリッド表示時の左右移動
- **Tab** - 次の項目へ移動
- **Shift + Tab** - 前の項目へ移動
- **Enter** - 項目選択・決定
- **Escape** - キャンセル・戻る

## 開発

依存関係はDockerfileを参考にする。

```
$ make help
```

## 設計ドキュメントの状況

`docs/design` の frontmatter から自動生成される。`go run . designdoc list` で絞り込める。

| status | 件数 |
|---|---|
| in-progress | 1 |
| done | 61 |

### 進行中

| No. | ドキュメント | 進捗 | tags |
|---|---|---|---|
| [64](docs/design/20260719_64.md) | 型ユーティリティによる堅牢化・利便化の調査と適用方針 | 0/9 |  |


## Reference

ゲーム作成で参考にしたコード等。

- https://github.com/x-hgg-x/sokoban-go
  - 最初にコピペして作成をはじめ、改変していった
  - ECSの使い方まわりで参考にした
- https://github.com/x-hgg-x/goecsengine
  - ゲームステートまわりで参考にした
- https://bfnihtly.bracketproductions.com/
  - 設定ファイルによるファクトリ、ゲームログまわりを参考にした
- https://krkrz.github.io/krkr2doc/kag3doc/contents/
  - サウンドノベルに必要な記法を参考にした
- https://ebitengine.org/en/examples/raycasting.html
  - レイキャストの実装の参考にした
- https://cataclysmdda.org/
  - ローグライクシステムの参考にした
- ゲームシステム面で、KONAMIのビデオゲーム『パワプロクンポケット』シリーズ10・11・12を参考にした
  - 途中の方針転換で、あまり残っていない

使用した素材類。

フォント。

- http://jikasei.me/font/jf-dotfont/
- https://github.com/googlefonts/morisawa-biz-ud-gothic
- https://font.download/font/augustus

画像。

- https://www.pixilart.com
- https://pixabay.com/photos/forest-fog-woods-trees-mystical-3394066/
- https://pixabay.com/photos/beer-drink-alcohol-heineken-bar-5940890/
- https://pixabay.com/photos/lost-places-monastery-past-masonry-4019367/
