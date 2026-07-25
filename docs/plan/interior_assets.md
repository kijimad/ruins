# 施設内装アセット 充足監査と要アート manifest

`20260725_70`(施設内装の生成)の前提タスク「施設 prop とタイル資産を先に揃える」への回答。人の目での「設計され感」
評価は、置ける prop とタイルの有無に律速される。ここで**何が既に揃っていて、何をアートで描き足すべきか**を精密に
scope する。

## 要約 —— 監査の結論

- **prop は既にアート充足**。`assets/metadata/entities/raw/raw.toml` に **77 個の `[[props]]`** が定義済みで、
  `single.json`(660 スプライト)の家具はほぼ prop 化されている。**未 wire の描画可能な家具は実質ゼロ**。つまり
  「既存スプライトを wire する」余地は無い。描ける家具は既に器に載っている。
- **不足はアートそのもの**。穴は2つ。①**タイルの材質・condition バリエーション**、②**一部の施設固有 prop**。
  いずれも新規ピクセルアートが要る。ここに列挙する。
- **評価はすぐ始められる施設がある**。下のマトリクスで「評価可」の施設は、既存 prop だけで placement ロジックの
  eyeball 評価に入れる。アート待ちで足踏みする必要はない。
- **不足分は文字入りダミーで用意済み**。要アートの prop 25・床材 tile 5 を、32x32 の色分けラベル placeholder として
  生成し、`raw.toml` に wire・`make aseprite` でパック済み。実アートが描かれ次第、`assets/file/textures/{single,tiles}/`
  の source PNG を差し替えて再パックするだけで本番に載る。placement の実装・評価をアート待ちせず始められる。
  なお **condition 変種は当面やらない**。

## 施設別 充足マトリクス

既存 prop で各施設の骨格がどこまで組めるか。評価可=既存 prop で「らしく」見せられる、部分=骨格は組めるが施設固有品が
欠ける、要拡充=不足が目立つ。

| 施設 | 使える既存 prop | 評価 | 主な不足(要アート) |
|---|---|---|---|
| コンビニ | register, goods_shelf, dish_shelf, iron_shelf, refrigerator, microwave, coffee_maker, book_showcase, carpet | 評価可 | 展示冷蔵ケース, 雑誌ラック, 自販機, 買い物カゴ, ゴンドラ/エンドキャップ, レジカウンター |
| 民家・宿 | bed, ベッドサイド, bed_lamp, sofa, 二人がけのソファ, table, chair, closet, ドレッサー, mirror, tall_lamp, laundry, bathtub, toilet, sink, carpet, clock, houseplants | 評価可 | 布団(和室), フロント受付, 鍵掛け |
| 倉庫 | crate, 木箱, wood_chest, barrel系, ドラム缶系, forklift, iron_shelf, generator, gauge_machine, furnace | 評価可 | パレット, ケージ/檻, 産業ラック(iron_shelf 流用可) |
| オフィス | desk, 仕事机, chair, 黒いチェア, desktop_pc, drawer_chest, electric_locker, clock, bookshelf各種, gauge_machine, desk_light | 部分 | パーティション/キュービクル, サーバラック, プリンタ/コピー機, ホワイトボード, ウォーターサーバー |
| 診療所 | bed, ベッドサイド, sink, drawer_chest, desk, chair, mirror, closet, desktop_pc | 部分 | 診察台/病院ベッド, IV スタンド, 車椅子, ストレッチャー, 薬品キャビネット, 受付カウンター, 医療モニタ |
| 神社・祠 | candle, lantern, stone_pillar, stone_tablet, flag_pot, mirror, bonfire, 黒い花瓶 | 要拡充 | 鳥居(小), 賽銭箱, 狛犬, 絵馬掛け, 香炉, 注連縄 |

## 要アート manifest —— prop

優先度は `20260725_70` のロードマップ順、すなわち「まず評価に入る施設」を高くする。既存 prop 流用で代替できる物は
その旨を記す。

**優先度 高**（評価対象の施設で施設性を決める品）
- コンビニ: 展示冷蔵ケース(オープン型), 雑誌ラック, ゴンドラ什器(中央島), レジカウンター
- 診療所: 診察台/病院ベッド, 薬品キャビネット, 受付カウンター
- オフィス: パーティション(キュービクル), サーバラック

**優先度 中**（施設の密度・風味を上げる）
- コンビニ: 自販機, 買い物カゴ/カート
- 診療所: IV スタンド, 車椅子, ストレッチャー, 医療モニタ
- オフィス: プリンタ/コピー機, ホワイトボード, ウォーターサーバー
- 倉庫: パレット, ケージ/檻

**優先度 低**（後回し可・authored 断片で代替も可）
- 神社: 鳥居(小), 賽銭箱, 狛犬, 絵馬掛け, 香炉, 注連縄
- 民家: 布団, フロント受付, 鍵掛け

## 要アート manifest —— タイル

現状の `[[tiles]]` は **dirt / floor / wall / dwall / void の5種のみ**。床は単一の `floor`、壁は autotile(`wall_N`)。
`20260725_70` の床材・condition 軸を目に見える形にするには、以下の新規タイルアートが要る。

**床材**（施設の材質感。ダミー用意済み）
- リノリウム(コンビニ・病院), タイル(浴室・厨房), コンクリ(倉庫・地下), フローリング(民家・オフィス), タタミ(和室)

condition 変種(略奪・汚損・水没・瓦礫)は当面やらない。materialの床材だけを用意する。壁材変種も今回は見送る。

## ダミーで用意済みのアセット

要アートの prop・tile を、施設カテゴリで色分けした文字入り 32x32 placeholder として生成し、`raw.toml` に wire 済み。
source PNG は `assets/file/textures/{single,tiles}/`、dist は `make aseprite` でパック済み。実アートはこの source を
差し替えて再パックする。`raw.toml` 内は `# === ダミー内装アセット ===` コメントで一括識別できる。

- **コンビニ**(琥珀): display_cooler, magazine_rack, vending_machine, shopping_basket, gondola_shelf, checkout_counter
- **診療所**(teal): exam_bed, iv_stand, wheelchair, stretcher, medicine_cabinet, reception_counter, medical_monitor
- **オフィス**(slate): cubicle_partition, server_rack, printer, whiteboard, water_cooler
- **倉庫**(rust): pallet, cage
- **神社**(vermilion): torii_small, offering_box, komainu, ema_rack, incense_burner
- **床材 tile**(grey): floor_linoleum, floor_tile, floor_concrete, floor_wood, floor_tatami

## アセットが届いたら —— 追加手順

新規スプライトが Aseprite で描かれ次第、定義を足すだけで器に載る。人手はアートに集中でき、配線は機械的。

1. スプライトを Aseprite ソースへ追加し `make aseprite` でパッキングする。`single.json`(prop)/`tiles.json`(tile)に
   スプライト名が出る。
2. `raw.toml` に定義を足す。prop の最小スキーマ:
   ```toml
   [[props]]
   blockPass = false   # 通行を阻むか。棚・什器は true が多い
   blockView = false
   description = "展示冷蔵ケース"
   hp = 20
   name = "display_cooler"

   [props.spriteRender]
   depth = 0
   spriteKey = "display_cooler"   # スプライト名
   spriteSheetName = "field"
   ```
   tile の最小スキーマ:
   ```toml
   [[tiles]]
   blockPass = false
   blockView = false
   description = "リノリウム床"
   foliage = 0
   name = "floor_linoleum"
   shelter = 0
   water = 0

   [tiles.spriteRender]
   depth = 0
   spriteKey = "floor_linoleum"
   spriteSheetName = "tile"
   ```
3. `make generate` で再生成し、`make check` で検証する。

## doc 70 との接続

本 manifest は `20260725_70` 進捗の先頭タスク「施設 prop とタイル資産を先に揃える」の scope 出力である。既存 prop で
「評価可」の施設(コンビニ・民家・倉庫)は、アート待ちせず placement 語彙(`20260725_70`)の実装と eyeball 評価に入れる。
「部分・要拡充」の施設は、上の高優先アートが揃った時点で評価対象に昇格する。
