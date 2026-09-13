# バランス調整のアプローチとメトリクス

パラメータ関係ごとに、どのアプローチで評価し、どのメトリクスを目標に置くかを定める正典。
死にコードと重複実装を防ぐため、ドメインとメトリクスとコード出典の対応をここ1箇所に固定する。

- パラメータの相互関係の全体像は `docs/balance/ruins_balance.dot`
- 生成されるベースラインは `docs/balance/baseline.md`（`make balance-report`）
- パラメータの所在と編集経路の索引は `docs/balance/tuning.md`
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

## 導出の前提。静的な保守下限

導出は固定プレイヤーの静的スナップショットで、実プレイを楽にする次の2つを意図的に含めない。含めないぶん導出値は「少なくともこれだけは厳しい」という保守的な下限になる。目標帯を下限で満たせば、実プレイはそれ以上に快適になる。

- **HP自然回復**。`systems/health_regen.go` は基準代謝で毎ターン2HP回復し、戦闘ターンでも走る。`balance/metrics.go` の `ExpectedTTK` はこれを引かないので、EnemyTTK は実際より短く、すなわち難しく出る。敵の期待ダメージが回復量に近い序盤ほど乖離が大きい。含めるなら正味ダメージ、すなわち期待ダメージ−回復で TTK を両側対称に計算するが、全戦力比が動き再調整が要るため今は下限のまま置く。
- **スキル成長**。`activity/attack.go` の `growWeaponSkill` で味方の武器スキルは戦闘で育つが敵は育たない。導出は成長前の固定スキルで見る。日が進むほど実プレイヤーは強くなるので後半は導出より楽になる。成長を織り込むには経過日から想定スキル値への軌道を仮定する必要があり、行動依存なので下限には載せない。

## 武器・スキルの調整方針

- **武器**。武器ステータスは戦闘メトリクスの入力そのもの。`formula` 経由で `ExpectedDamagePerAttack` に効くので、raw の `melee.damage` 等を yq で変え、戦力比カーブと感度表・Pareto 前線で目標帯へ寄せる。武器調整は戦闘ドメインの既存ワークフローで完結する。
- **スキル**。スキル成長は戦闘の下限には含めないが、成長そのものを別ドメイン「進行・成長」で導出する。`balance/growth.go` の `AttacksToSkillLevel` が実物 `skill.GainExp` の反復で Lv N 到達に要する攻撃回数を返す。`skill/growth.go` の BaseExp・AbilBonus・DecayPerLevel・MaxLevel を変えるとこの攻撃回数が動くので、目標、例えば主要武器スキルが想定攻撃回数で程よく上がるに対して合わせる。経過日 d への写像は攻撃頻度という行動依存量が要るので C 層に残す。設計は `docs/design/260913164022.md`。

## 依存順

上流のパラメータから下流へ調整する。`ruins_balance.dot` の矢印の向き。

```
時間・季節 → 危険度・世界温度 → 体温・生存 / 敵プール・戦闘 → 経済
能力値 → HP と戦闘・生存の両方（触ると戦闘ベースラインが動く。凍結ゲートで自動検知）
```

- 能力値は戦闘と生存の共通入力。能力値を調整するなら、戦闘の凍結ゲートが差分を教える。
- 経済は最下流。戦闘結果・loot・競売の合流点なので、上流が固まってから。

## 現状と次

- 実装済み: 戦闘・生存（飢え/寒さ/血液→HP/疲労/睡眠回復）・物流・経済（競売手取り・1個あたり期待手取り）・進行成長・気候の各導出。凍結ゲート・感度・交換レート・単変数探索・多目的Pareto探索。全ドメインが baseline.md に載る。
- 期待値だけでなく分布: 戦闘は `balance/markov.go` の `CombatDistribution` が吸収マルコフ連鎖で死亡確率とTTK分布を厳密に解く。先攻固定の交互攻撃なので自分と敵の撃破攻撃数は独立で、勝敗は Kp<=Ke、死亡確率は P(Ke<Kp) に分解できる。`rollAttack` のダメージを PMF へ写して残HPを状態とする1次元DPで撃破攻撃数分布を出す。閉形式が実戦闘と一致することは markov_test.go で MC と突き合わせて担保する。戦力比の期待値が滑らかでも死亡確率は危険度の崖で跳ねるので、突然死の検知に効く。baseline.md に死亡確率カーブとして載る。
- 境界マージンと要素制限: 学術・業界手法の調査結果は `docs/balance/research-auto-balancing.md`。置き換え候補はなく、閉形式の上に載る補完を採った。`balance/boundary.go` の `BoundaryMargins` は各つまみを動かして目標帯を割るまでの余白を二分探索で解き、凍結ゲートの点固定を崖までの距離の監視へ拡張する。BBExplorer の境界発見の定式化を決定論のまま移植したもの。`balance/restriction.go` の `WeaponRestrictionValues` は武器を素手へ制限したときの死亡確率の劣化量を測り、要素の必須度と死にコンテンツを暴く。Jaffe らの Restricted Play を対戦でないサバイバルへ翻案し、勝率でなく死亡確率の劣化量で定義したもの。どちらも baseline.md に載る。
- 目標帯: 戦闘の日次カーブに加え、生存・物流・経済・進行成長のスカラー指標に横断の目標帯を置いた（`balance/targets.go` の `DomainTargets`）。baseline.md 冒頭のダッシュボードで内外を一望する。帯は設計仮説で、静的下限がこの範囲に収まればという下限側の目安。
- 感度と索引: ドメイン横断の感度行列（つまみ×メトリクスの弾力性、`CrossDomainSensitivity`）を baseline.md に出す。つまみは `knobRegistry` の全成分を機械的に回すので、Params に成分を足せば行列へ自動で載る。ほぼブロック対角でドメインは疎結合、横断するのは1日ターン数のような共通分母だけ、と可視化する。外れをどのつまみで戻すかの手掛かりになる。全パラメータの所在と編集経路は `docs/balance/tuning.md` に索引化した。
- ビルド帯: 想定攻撃回数→スキル値→熟練度倍率で代表ビルドを作り、段階別(序盤/中盤/終盤)のブレ幅と許容帯を baseline.md に出す(`balance/build.go`)。倍率は恣意でなくスキル成長と熟練度の実システム由来。設計は `docs/design/260913182537.md`(done)。
- 残る発展: 探索1回の期待収入は閉形式(層数入力)、生存込み分布は simulate-balance の stdout に MC で出す。残るのは移動燃料コスト側(移動タイル数という設計値)と、定数側を数値摂動する窓の拡充。
- 運用: パラメータ調整は baseline.md の目標帯ダッシュボードと感度行列、差分、凍結ゲートで回す。目標帯を人間が決め、tuning.md でつまみを引き当て、探索や手編集で値を寄せる。
