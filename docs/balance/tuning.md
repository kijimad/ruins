# チューニング索引

バランスに効くパラメータの所在を1箇所に集める。散在する定数を探す手間を省き、どれを触ると
どのメトリクスが動くかを引けるようにする。導出の考え方は `docs/balance/approach.md`、現状値の
一望は `docs/balance/baseline.md` の目標帯ダッシュボードと感度行列。

編集経路は2種類。**raw** は `assets/metadata/entities/raw/raw.toml` を yq で編集する。**const** は
Go 定数でコード編集する。詳細は approach.md「武器・スキルの調整方針」と「raw.toml の書き戻しは yq で行う」。

## 戦闘・装備

| パラメータ | 所在 | 効くメトリクス | 経路 |
|---|---|---|---|
| 武器 melee.damage / accuracy / attackCount | raw.toml `items` | ExpectedDamagePerAttack・TTK・PowerRatio | raw |
| 敵の能力 HP/STR/DEX/AGI/DEF | raw.toml `members` | EnemyTTK・PowerRatio | raw |
| 敵テーブルの weight / minDanger / maxDanger | raw.toml `enemyTables` | PowerRatio カーブ全体 | raw |
| プレイヤー能力(ash) | raw.toml `members` | PlayerTTK・PowerRatio | raw |
| 命中・クリティカルの式 | `internal/formula` | 命中率・期待ダメージ | const |

## 危険度・気候

| パラメータ | 所在 | 効くメトリクス | 経路 |
|---|---|---|---|
| `dangerDaysPerLevel` = 3 | `query/danger.go` | DangerLevelForDay。危険度の上がる速さ | const |
| `springTemp`5 `summerPeakTemp`22 `autumnTemp`0 `winterTroughTemp`-30 `daysPerYear`32 | `components/game_time.go` | WorldTemperatureAtDay。季節の寒暖 | const |
| `latitudeColdPerChunk`1 `latitudeColdMax`40 | `systems/temperature.go` | 奥地の寒さ勾配 | const |

## 生存

| パラメータ | 所在 | 効くメトリクス | 経路 |
|---|---|---|---|
| `DefaultMaxHunger`500 `HungerDrainTurns`3 `HungerStarvingRatio`0.33 | `components/hunger.go` | DaysUntilStarving・DaysUntilHungerEmpty | const |
| `bodyTempColdBand`・体温変化率・`ambientHeatPerWarmth`30 | `systems/temperature.go` | TurnsToHypothermia・火の暖かさ | const |
| `BloodLossHPDrainRate` の段階 | `components/health_status.go` | HPDrainPerTurnAtBlood | const |
| `DefaultMaxFatigue`2000 `FatigueGainPerTurn`1 疲労比率0.3/0.5/0.8 | `components/fatigue.go` | DaysUntilTired/Exhausted・SleepTimeFraction | const |
| `fatigueRecoverPerTurn`3 | `systems/fatigue.go` | SleepTurnsToFullRecover・SleepTimeFraction | const |
| `healthRegenBasePerTurn`2 | `systems/health_regen.go` | 静的下限には非含。approach.md 参照 | const |

## 物流

| パラメータ | 所在 | 効くメトリクス | 経路 |
|---|---|---|---|
| `DriveFuelBase`10 `DriveFuelPerKg`1 `CubeWeightCapacityKg`500 | `consts/consts.go` | DriveRangeTiles・DriveRangeAllFuel | const |
| `materialHeatPerKg` OIL1000/COAL800/WOOD200/FOOD60/BONE40 | `query/fuel.go` | 航続・FuelBurnTurns | const |
| `groundBurnEfficiency`50 | `query/fire.go` | FuelBurnTurns | const |

## 経済

| パラメータ | 所在 | 効くメトリクス | 経路 |
|---|---|---|---|
| `auctionShipRatePerKg`25 `auctionFeeRate`0.12 `auctionOpeningMult`0.4 `auctionRaiseMult`0.15 `AuctionBidChance`0.6 `AuctionPickupFee`100 | `query/auction.go` | AuctionTakeHomeRate・ExpectedNetLootValue | const |
| loot の value / weight | raw.toml `items` | ExpectedLootValue・ExpectedNetLootValue | raw |
| itemTable / itemGroup の weight・帯 | raw.toml `itemTables` `itemGroups` | ExpectedLootValue | raw |

## 進行・成長

| パラメータ | 所在 | 効くメトリクス | 経路 |
|---|---|---|---|
| `growthConfig` BaseExp10 AbilBonus5 DecayPerLevel20 MaxLevel100 | `skill/growth.go` | AttacksToSkillLevel | const |
| `LevelUpExp`100 | `components/skills.go` | AttacksToSkillLevel | const |

## 時間の土台

| パラメータ | 所在 | 効くメトリクス | 経路 |
|---|---|---|---|
| `TurnsPerDay`(consts.Turn=1500) | `components/game_time.go` | 全ドメインの日数換算 | const |

## 使い方

1. baseline.md の目標帯ダッシュボードで外れたドメインを見つける。
2. 感度行列(戦闘)か、この索引の「効くメトリクス」列で、どのパラメータを引くか当たりを付ける。
3. raw は yq、const はコード編集で動かし、`make balance-report` で再導出して内へ寄せる。凍結ゲートが意図しない波及を止める。
