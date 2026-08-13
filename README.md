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
| <img src="internal/states/testdata/TestGolden_Character.png" width="200" /><br>Character | <img src="internal/states/testdata/TestGolden_CharacterJob.png" width="200" /><br>CharacterJob | <img src="internal/states/testdata/TestGolden_CharacterNaming.png" width="200" /><br>CharacterNaming | <img src="internal/states/testdata/TestGolden_ChoiceMenuHeaders.png" width="200" /><br>ChoiceMenuHeaders |
| <img src="internal/states/testdata/TestGolden_ChoiceMenuMany.png" width="200" /><br>ChoiceMenuMany | <img src="internal/states/testdata/TestGolden_ComponentDebug.png" width="200" /><br>ComponentDebug | <img src="internal/states/testdata/TestGolden_CraftMenu.png" width="200" /><br>CraftMenu | <img src="internal/states/testdata/TestGolden_CubePanel.png" width="200" /><br>CubePanel |
| <img src="internal/states/testdata/TestGolden_DebugMenu.png" width="200" /><br>DebugMenu | <img src="internal/states/testdata/TestGolden_Dungeon.png" width="200" /><br>Dungeon | <img src="internal/states/testdata/TestGolden_EquipSelect.png" width="200" /><br>EquipSelect | <img src="internal/states/testdata/TestGolden_GameOver.png" width="200" /><br>GameOver |
| <img src="internal/states/testdata/TestGolden_ItemAction.png" width="200" /><br>ItemAction | <img src="internal/states/testdata/TestGolden_ItemActionDetail.png" width="200" /><br>ItemActionDetail | <img src="internal/states/testdata/TestGolden_LanguageMenu.png" width="200" /><br>LanguageMenu | <img src="internal/states/testdata/TestGolden_LoadMenu.png" width="200" /><br>LoadMenu |
| <img src="internal/states/testdata/TestGolden_LookAround.png" width="200" /><br>LookAround | <img src="internal/states/testdata/TestGolden_MainMenu.png" width="200" /><br>MainMenu | <img src="internal/states/testdata/TestGolden_Message.png" width="200" /><br>Message | <img src="internal/states/testdata/TestGolden_Overworld.png" width="200" /><br>Overworld |
| <img src="internal/states/testdata/TestGolden_OverworldFrost.png" width="200" /><br>OverworldFrost | <img src="internal/states/testdata/TestGolden_OverworldMap.png" width="200" /><br>OverworldMap | <img src="internal/states/testdata/TestGolden_PersistentMessage.png" width="200" /><br>PersistentMessage | <img src="internal/states/testdata/TestGolden_SaveMenu.png" width="200" /><br>SaveMenu |
| <img src="internal/states/testdata/TestGolden_SettingsMenu.png" width="200" /><br>SettingsMenu | <img src="internal/states/testdata/TestGolden_Shooting.png" width="200" /><br>Shooting | <img src="internal/states/testdata/TestGolden_ShopMenu.png" width="200" /><br>ShopMenu | <img src="internal/states/testdata/TestGolden_StorageMenu.png" width="200" /><br>StorageMenu |


各画像はゴールデンテストで自動生成される。

## キーボード操作

### ダンジョン探索

**移動**

| キー | 動作 |
| --- | --- |
| ↑ / ↓ / ← / → | 上下左右の移動 |
| Shift + 矢印2方向 | 斜め移動 |
| . (ピリオド) | その場で待機 |

**行動**

| キー | 動作 |
| --- | --- |
| Enter | 足元・隣接の対象と相互作用する |
| Space | インタラクションメニューを開く |
| G | 足元のアイテムを拾う |
| F | 射撃モード |
| 1〜5 | 武器スロットの切り替え |

**動詞コマンド**（アイテム操作画面を開く）

| キー | 動作 |
| --- | --- |
| Shift + X | 調べる |
| D | 置く |
| E | 食べる・飲む |
| R | 読む |
| T | 使う |

**画面・情報**

| キー | 動作 |
| --- | --- |
| M | ダンジョンメニュー |
| L | フィールド情報 |
| N | オーバーワールド地図（オーバーワールドにいるときのみ） |

**ズーム**

| キー | 動作 |
| --- | --- |
| PageDown | ズームアウト |
| PageUp | ズームイン |
| マウスホイール | ズーム操作 |

### メニューナビゲーション

| キー | 動作 |
| --- | --- |
| ↑ / ↓ | 項目の上下移動 |
| ← / → | グリッド表示時の左右移動 |
| Tab | 次の項目へ移動 |
| Shift + Tab | 前の項目へ移動 |
| Enter | 項目選択・決定 |
| Escape | キャンセル・戻る |

## 開発

依存関係はDockerfileを参考にする。

```
$ make help
```

## 設計ドキュメントの状況

`docs/design` の frontmatter から自動生成される。`go run . designdoc list` で絞り込める。

| status | ドキュメント | 進捗 | tags |
|---|---|---|---|
| in-progress | [activity パッケージのわかりやすさ改善](docs/design/260809163430.md) | 8/8（見送り2） | refactor |
| in-progress | [一覧のセルをアイコンを持てる型にし、アイテム・装備にアイコンを表示する](docs/design/260810093849.md) | 7/7 | ui, item |
| in-progress | [隊員（NPC 仲間）を削除する](docs/design/260810222240.md) | 3/4（見送り1） | gamedesign, member |
| in-progress | [食料の腐敗](docs/design/260813080544.md) | 6/11（見送り2） | item, gamedesign |
| draft | [難所調査から抽出した自走可能な改善のバックログ](docs/design/260724224417.md) | 0/11（見送り3） | refactor, ci, ecs, combat, meta |
| draft | [施設内装の生成 —— doc 260725201431.md の未着手バックログ](docs/design/260731225939.md) | 1/33 | worldgen |
| draft | [OSS 調査 2026-08](docs/design/260801002222.md) | 0/5 | meta, worldgen, combat, ui |
| draft | [コード監査: ダンジョン生成・リロード・セーブ 2026-08-03](docs/design/260803144226.md) | 5/5 | worldgen, combat, save, ecs |
| draft | [天気システムを設計する](docs/design/260805144630.md) | 0/7 | gamedesign, worldgen, ecs |
| draft | [コード監査: ダンジョン生成・戦闘継続・UI描画 2026-08-10](docs/design/260810001803.md) | 0/5 | worldgen, combat, item, ui, ecs |


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
