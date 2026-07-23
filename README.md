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
| in-progress | 32 |
| done | 29 |
| dropped | 1 |

### 進行中

| No. | ドキュメント | 進捗 | tags |
|---|---|---|---|
| [3](docs/design/20260124_3.md) | Stackableアイテムのリファクタリングを計画する | 2/15 |  |
| [4](docs/design/20260125_4.md) | 戦闘システムを再設計する | 12/20 |  |
| [6](docs/design/20260215_6.md) | 橋遷移システム設計 | 2/17（見送り1） |  |
| [7](docs/design/20260216_7.md) | 極寒環境サバイバルローグライク設計 | 10/29 |  |
| [8](docs/design/20260221_8.md) | ダンジョン選択システム設計 | 14/20（見送り1） |  |
| [10](docs/design/20260223_10.md) | 気温・体温システム設計 | 16/26（見送り1） |  |
| [11](docs/design/20260228_11.md) | APシステム改善設計 | 10/15（見送り1） |  |
| [13](docs/design/20260319_13.md) | スキル・職業システム設計 | 16/22（見送り2） |  |
| [16](docs/design/20260419_16.md) | RUINS - キーコンセプト「火と氷」 | 5/8 |  |
| [17](docs/design/20260419_17.md) | >オープニングの要件 | 8/9（見送り1） |  |
| [18](docs/design/20260420_18.md) | ボス戦の設計方針 | 9/14 |  |
| [20](docs/design/20260426_20.md) | ダンジョン間設計 | 7/8 |  |
| [21](docs/design/20260429_21.md) | オープニングナレーション案 | 5/8 |  |
| [22](docs/design/20260524_22.md) | ゲームバランス調整の方法論 | 12/21（見送り2） |  |
| [23](docs/design/20260528_23.md) | コンテンツ追加とSteamリリース準備 | 4/13（見送り1） |  |
| [25](docs/design/20260531_25.md) | 敵AI行動戦略の多様化 | 14/16（見送り3） |  |
| [28](docs/design/20260609_28.md) | Propメカニクス設計 | 4/7（見送り1） |  |
| [30](docs/design/20260613_30.md) | UIコンポーネント単体のゴールデンテスト | 14/15（見送り3） |  |
| [32](docs/design/20260620_32.md) | Steam ストアグラフィック制作計画 | 11/20 |  |
| [33](docs/design/20260620_33.md) | Stable Diffusion による Steam グラフィック生成手順 | 8/11（見送り1） |  |
| [40](docs/design/20260628_40.md) | 隊マネジメントシステムへの転換 | 15/25（見送り1） |  |
| [42](docs/design/20260628_42.md) | 隊員アイテム運搬システム | 0/11 |  |
| [51](docs/design/20260710_51.md) | ECSエンジンの goecs から Ark への移行 | 30/33 |  |
| [52](docs/design/20260711_52.md) | CI ベストプラクティス施策の提案（安全性・開発効率・カバレッジ精度） | 4/14 |  |
| [55](docs/design/20260713_55.md) | マクロ移動（キャラバンのルート網踏破）の実装設計 | 0/13 |  |
| [56](docs/design/20260714_56.md) | マクロ移動の廃止 ── 出口選択型ラン構造（Hades/DD型） | 0/5 |  |
| [58](docs/design/20260715_58.md) | 高速移動（走り）の実装設計 | 13/26 |  |
| [59](docs/design/20260716_59.md) | ゲームパフォーマンステスト戦略 | 9/11（見送り1） |  |
| [60](docs/design/20260717_60.md) | シームレスワールドの実装 ── スライディング帯によるチャンクストリーミング | 8/9 |  |
| [62](docs/design/20260718_62.md) | GitHub Actions CI のキャッシュ最適化（Docker / Go） | 3/4（見送り2） |  |
| [63](docs/design/20260719_63.md) | シームレスワールド Phase 7: 永続ステージと往復 | 9/18 |  |
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
