# バランス調整のアプローチとメトリクス

パラメータ関係ごとに、どのアプローチで評価し、どのメトリクスを目標に置くかを定める正典。
死にコードと重複実装を防ぐため、ドメインとメトリクスとコード出典の対応をここ1箇所に固定する。

- パラメータの相互関係の全体像は `docs/balance/ruins_balance.dot`
- 生成されるベースラインは `docs/balance/baseline.md`（`make balance-report`）
- 設計の経緯は `docs/design/260913002926.md`

## アプローチの3層

導出しやすさでアプローチが決まる。ドメインを次の3層に振り分ける。

| 層 | 何か | 使いどころ | 実装 |
|---|---|---|---|
| A 閉形式 | 体験指標が少数のパラメータの純関数で書ける | 戦闘・生存タイマー・物流 | `internal/balance` の導出関数。乱数なし・決定論的・ミリ秒 |
| B 連鎖 | 複数の式を合成すれば決定論的に導ける | 空腹→血液→HP、体温→状態異常→HP | A の関数を組み合わせる。ステートの遷移を1本の式列で表す |
| C 創発 | 行動依存でランダム。閉形式は定常近似止まり | 経済ループ、複数ドメインの戦略、競売の確率入札 | モンテカルロ `simulate-balance`、将来はエージェント対戦 |

3層は対立でなく、導出しやすさに応じた使い分け。まず A、無理なら B、それでも無理なら C。

## ドメイン別の採用

`ruins_balance.dot` の列に対応する。メトリクスは体験指標、すなわち日数・タイル・戦力比・収支で宣言する。生パラメータ値は目標にしない。

| ドメイン | 主要な関係 | 層 | 採用メトリクス | コード出典（単一出典） | 状態 |
|---|---|---|---|---|---|
| 時間・進行 | 経過日→危険度、季節→世界温度 | ― | 単独メトリクスなし。他ドメインの進行軸 d に使う | `components/game_time.go` TurnsPerDay、`query/danger.go` DangerLevelForDay | 土台 |
| 危険度・世界 | 危険度→敵プール、季節+時間→世界温度 | A | 難易度カーブ（戦力比 対 日）。世界温度の季節変動 | `query/danger.go`、`query/temperature.go` | 戦闘側で実装済み |
| 戦闘・装備 | 能力+武器−防御→ダメージ、命中、TTK | A | 戦力比 PowerRatio、ExpectedTTK、ExpectedDamagePerAttack | `formula`（CalcHitRate/ApplyCritical/CalcHP）→ `balance/metrics.go` | 実装済み |
| サバイバル（飢え・寒さ） | 満腹→減耗、体温→低体温 | A | DaysUntilStarving/DaysUntilHungerEmpty、TurnsToHypothermia | `components/hunger.go`、`systems/temperature.go`（CalcBodyTempRate/BodyTempColdBand）→ `balance/survival.go` | 実装済み |
| サバイバル（連鎖） | 空腹/低体温→状態異常→血液→HP、代謝→回復 | B | 状態異常からの HP 減到達ターン、無回復での生存打数 | `components/health_status.go` BloodLossHPDrain/Metabolism | 未着手 |
| 疲労・睡眠 | 経過→疲労、睡眠→疲労回復・代謝 | A/B | 睡眠なしの限界ターン、睡眠1回の回復量 | `systems/fatigue.go`、`activity/sleep.go` | 未着手 |
| 能力値 | VIT/STR/SEN→HP、STR/DEX/AGI→戦闘 | ― | 単独メトリクスなし。戦闘・生存の入力で、探索が動かすつまみ | `formula.CalcHP`、`formula.CalcHitRate` | 戦闘に内包 |
| 物流・キューブ | 燃料/(基準+kg)→航続、積載↔移動、火→暖 | A | 満載時の航続タイル、積載と航続のトレード、燃料の燃焼ターン | `query/cube.go` DriveFuelCost、`consts` DriveFuelBase/DriveFuelPerKg/CubeWeightCapacityKg、`query.HeatOf` → `balance/logistics.go` | 実装済み |
| 経済・終端 | loot価値→売買/競売、送料・手数料→手取り | A+C | 探索1回の期待収支、競売手取り率 | 純部分は `query/auction.go`（送料25/kg・手数料0.12・集荷100・開始0.4）。行動依存は `balance/run.go` モンテカルロ | 後回し |

補足。競売は毎ターン確率 0.6 で入札が延びる確率過程（`query/auction.go` AuctionBidChance）。手取りの定常近似は A で出せるが、実際の落札額分布は C で測る。

## 実装規約（死にコード・重複を防ぐ）

- **単一出典**。メトリクスは `formula`・`consts`・`query` の実式と実定数だけを参照する。戦闘式や燃費式を `balance` 内に再実装しない。式が変われば導出も自動追従する。
- **公開窓は最小限**。world 抜きで導出するために純関数だけを公開する。既存の窓は `query.DangerLevelForDay`、`systems.CalcBodyTempRate`、`systems.BodyTempColdBand`、`components.TurnsPerDay`。窓を増やすときはこの一覧に足し、乱用しない。
- **置き場を固定**。A/B の導出は `internal/balance` に置く。ドメインごとにファイルを分ける。`metrics.go`（戦闘）、`survival.go`（生存）、物流は `logistics.go`、経済の純部分は `economy.go` を予定。1ドメイン1ファイル。
- **モンテカルロは C 専用**。`balance/run.go`・`combat.go` の `Simulate*` は創発ドメインの測定にだけ使う。導出可能なドメインへ拡張しない。拡張したくなったら、それは A か B で書けるはずと疑う。
- **凍結ゲート**。導出したメトリクスは `balance/targets_test.go` に現状値で pin し、`make check` で変化を検知する。目標帯そのものは assert しない。意図した調整で値が動いたら期待値を更新する。
- **旧経路の扱い**。`simulate-balance` cmd → `balance.json` → editor-ui の BalancePage/DPSPage は C 層として残置する。導出系（baseline.md）に役割が移ったら、editor 各ページの実依存を精査してから別 PR で整理を判断する。今は消さない。

## 依存順

上流のパラメータから下流へ調整する。`ruins_balance.dot` の矢印の向き。

```
時間・季節 → 危険度・世界温度 → 体温・生存 / 敵プール・戦闘 → 経済
能力値 → HP と戦闘・生存の両方（触ると戦闘ベースラインが動く。凍結ゲートで自動検知）
```

- 能力値は戦闘と生存の共通入力。能力値を調整するなら、戦闘の凍結ゲートが差分を教える。
- 経済は最下流。戦闘結果・loot・競売の合流点なので、上流が固まってから。

## 現状と次

- 実装済み: 戦闘（戦力比カーブ）、生存の飢え・寒さ、物流（航続・積載トレード）。凍結ゲート・感度・単変数探索。
- 次: サバイバル連鎖 B（空腹/低体温→状態異常→血液→HP）、疲労・睡眠。
- 後: 経済 A+C。
