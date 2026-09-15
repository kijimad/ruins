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
| 危険度・世界 | 危険度→敵プール、季節+時間→世界温度 | A | 難易度カーブ（戦力比 対 日）。世界温度の季節変動 | `query/danger.go` → `balance/metrics.go`、`components.SeasonalTemperatureForDay` → `balance/climate.go` | 実装済み |
| 戦闘・装備 | 能力+武器−防御→ダメージ、命中、TTK | A | 戦力比 PowerRatio、ExpectedTTK、ExpectedDamagePerAttack、死亡確率とTTK分布 | `formula`（CalcHitRate/ApplyCritical/CalcHP）→ `balance/metrics.go`・`balance/markov.go` | 実装済み |
| サバイバル（飢え・寒さ） | 満腹→減耗、体温→低体温 | A | DaysUntilStarving/DaysUntilHungerEmpty、TurnsToHypothermia | `components/hunger.go`、`systems/temperature.go`（CalcBodyTempRate/BodyTempColdBand）→ `balance/survival.go` | 実装済み |
| サバイバル（連鎖） | 状態異常→血液→HP | B | 血液量ごとのHP減 | `components.BloodLossHPDrainRate` → `balance/survival.go` | 実装済み |
| 疲労・睡眠 | 経過→疲労、睡眠→回復 | A | 疲労/過労までの日数、満タン回復までの睡眠ターン、釣り合いに要する睡眠時間の割合 | `components.FatigueTiredRatio`・`systems.FatigueRecoverPerTurn` → `balance/survival.go` | 実装済み |
| 能力値 | VIT/STR/SEN→HP、STR/DEX/AGI→戦闘 | ― | 単独メトリクスなし。戦闘・生存の入力で、探索が動かすつまみ | `formula.CalcHP`、`formula.CalcHitRate` | 戦闘に内包 |
| 進行・成長 | 攻撃→スキル経験→スキル値 | A | Lv N 到達に要する攻撃回数 | `skill.GainExp` の反復・`skill.MaxLevel` → `balance/growth.go` | 実装済み。攻撃回数のみ、日への写像はC |
| 物流・キューブ | 燃料/(基準+kg)→航続、積載↔移動、火→暖 | A | 満載時の航続タイル、積載と航続のトレード、燃料の燃焼ターン | `query/cube.go` DriveFuelCost、`consts` DriveFuelBase/DriveFuelPerKg/CubeWeightCapacityKg、`query.HeatOf`・`query.GroundBurnEfficiency` → `balance/logistics.go` | 実装済み |
| 経済・終端 | 競売の手数料・送料→手取り、loot→手取り→探索収入 | A+C | 競売の手取り率、1個あたりの期待手取り、探索1回の期待収入(層数入力) | `query.AuctionNetProceeds`・itemTable→group→value/weight・`floorItemBase/Random` → `balance/economy.go` | 実装済み。到達層数の分布と移動コスト側のみC/設計値 |

補足。競売は毎ターン確率 0.6 で入札が延びる確率過程（`query/auction.go` AuctionBidChance）。手取りの定常近似は A で出せるが、実際の落札額分布は C で測る。

## 実装規約（死にコード・重複を防ぐ）

- **単一出典**。メトリクスは `formula`・`consts`・`query` の実式と実定数だけを参照する。戦闘式や燃費式を `balance` 内に再実装しない。式が変われば導出も自動追従する。
- **パラメータはベクトルで持つ**。バランスに効くつまみは `balance/params.go` の `Params` に1つのベクトルとして集約し、導出は Params の純関数として書く。ゲーム定数との接点は `DefaultParams` の1箇所に限り、定数変更に自動追従させる。感度・交換レート・探索は `knobRegistry` の成分を摂動する汎用機械にし、つまみの追加は成分とレジストリの1行で済ませる。つまみごとの専用配管を書かない。
- **公開窓は最小限**。world 抜きで導出するために純関数だけを公開する。既存の窓は `query.DangerLevelForDay`、`systems.CalcBodyTempRate`、`systems.BodyTempColdBand`、`components.TurnsPerDay`。窓を増やすときはこの一覧に足し、乱用しない。
- **置き場を固定**。A/B の導出は `internal/balance` に置く。ドメインごとにファイルを分ける。`metrics.go`（戦闘）、`survival.go`（生存・疲労・血液）、`logistics.go`（物流）、`economy.go`（経済の純部分）。1ドメイン1ファイル。基準値は共有定数、すなわち `BaselinePlayer` 等や `consts` の公開定数を使い各所へ直書きしない。
- **モンテカルロは C 専用**。`balance/run.go`・`combat.go` の `Simulate*` は創発ドメインの測定にだけ使う。導出可能なドメインへ拡張しない。拡張したくなったら、それは A か B で書けるはずと疑う。
- **凍結ゲート**。導出したメトリクスは `balance/targets_test.go` に現状値で pin し、`make check` で変化を検知する。目標帯そのものは assert しない。意図した調整で値が動いたら期待値を更新する。
- **raw.toml の書き戻しは yq で行う**。導出で決めた値を書き戻すときは jq 風の式で該当フィールドだけを更新する。独自ツールは作らない。編集後の正規形は `make fmt` が整える。
  ```sh
  # 武器の近接ダメージを更新する
  go run github.com/mikefarah/yq/v4@v4.53.6 -p toml -o toml -i \
    '(.items[] | select(.id=="cleaver").melee.damage) = 12' assets/metadata/entities/raw/raw.toml
  # 敵テーブルの出現重み・危険度を更新する
  go run github.com/mikefarah/yq/v4@v4.53.6 -p toml -o toml -i \
    '(.enemyTables[] | select(.id=="ruins_area").entries[] | select(.id=="slime").maxDanger) = 6' assets/metadata/entities/raw/raw.toml
  ```
- **旧経路の扱い**。`simulate-balance` cmd → `balance.json` → editor-ui の BalancePage/DPSPage は C 層として残置する。導出系（baseline.md）に役割が移ったら、editor 各ページの実依存を精査してから別 PR で整理を判断する。今は消さない。

## 導出の前提。静的な保守下限と想定プレイヤー

導出の基準は固定プレイヤーの静的スナップショット、すなわち床。HP自然回復（`systems/health_regen.go`）とスキル成長（`activity/attack.go` の `growWeaponSkill`）を含めないので、床は「少なくともこれだけは厳しい」という保守的な下限になる。

その上に、想定スキル+装備防御を織り込んだ想定プレイヤーの軌道を `balance/progression.go` が別途出す。攻撃頻度と装備は行動依存の設計仮説だが、明示すれば実プレイヤーの体験曲線が見え、主基準になる。床と想定の帯で両側から見る。

## 依存順

上流のパラメータから下流へ調整する。`ruins_balance.dot` の矢印の向き。

```
時間・季節 → 危険度・世界温度 → 体温・生存 / 敵プール・戦闘 → 経済
能力値 → HP と戦闘・生存の両方（触ると戦闘ベースラインが動く。凍結ゲートで自動検知）
```

- 能力値は戦闘と生存の共通入力。能力値を調整するなら、戦闘の凍結ゲートが差分を教える。
- 経済は最下流。戦闘結果・loot・競売の合流点なので、上流が固まってから。

## 分析ツール

導出の上に、調整を助ける分析を載せる。すべて決定論的で baseline.md に出る。個々の関数とファイルは `internal/balance/doc.go` を見る。

- 分布: 戦闘を吸収マルコフ連鎖で解き死亡確率とTTK分布を出す。期待値の戦力比が滑らかでも死亡確率は危険度の崖で跳ねる。
- 感度行列・交換レート: つまみ×メトリクスの弾力性と、同じメトリクスを保つつまみ間の相互補償。`knobRegistry` を機械的に回す。
- 境界マージン・要素制限・viability・スキル深度: 崖までの余白、要素を封じたときの劣化量、使える武器の割合と差、レベルアップの手応え。
- 進行カーブ: 床（スキル0・無装備）と想定プレイヤー（想定スキル+装備防御）の死亡確率を同じ日軸で並べる。主基準は想定プレイヤーの死亡確率 `TargetExpectedDeath`。床の戦力比目標 `TargetPowerRatio` は下限リファレンスで、敵を進行で強化する設計では床が後半に目標を下回るのが正常。

## 運用

目標帯を人間が決め、baseline.md のダッシュボード・感度行列・交換レート・凍結ゲートでつまみを引き当て、探索か手編集（yq）で値を寄せる。
