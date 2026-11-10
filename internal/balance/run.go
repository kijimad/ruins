package balance

import (
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// フロアあたりの移動歩数。50x50マップでの入口→出口＋探索を推定した値
const (
	baseStepsPerFloor   = 60
	randomStepsPerFloor = 40 // 平均80歩（60+rand(40)）
)

// RunResult は1ランの結果
type RunResult struct {
	ReachedDepth        int
	Died                bool
	HPByDepth           map[int]int    // 回復後のHP
	HPBeforeHealByDepth map[int]int    // 回復前のHP（戦闘ダメージの実態を反映する）
	WeaponByDepth       map[int]string // 各深度で使用した武器名
	WeaponDamageByDepth map[int]int    // 各深度での武器ダメージ値
	AvgKillTurnsByDepth map[int]int    // 各深度での1戦あたり平均キルターン
	HungerByDepth       map[int]int    // 各深度終了時の空腹度
	LootIncome          int            // ラン中に拾った loot の店売り手取り総額。早死にするほど少ない
}

// SimulateRun はラン全体を模擬する。maxDepth まで進み死亡で終了する。フロアのドロップでより強い武器へ切り替える。
func SimulateRun(master oapi.Raws, enemyTableName string, player CombatantStats, playerWeapon WeaponStats, maxDepth int, rng *rand.Rand) RunResult {
	result := RunResult{
		HPByDepth:           make(map[int]int),
		HPBeforeHealByDepth: make(map[int]int),
		WeaponByDepth:       make(map[int]string),
		WeaponDamageByDepth: make(map[int]int),
		AvgKillTurnsByDepth: make(map[int]int),
		HungerByDepth:       make(map[int]int),
	}
	currentHP := player.HP
	currentWeapon := playerWeapon
	currentWeaponName := "bare_hands"
	hunger := gc.NewHunger()
	needs := raw.NewNeedDefs(master)
	foodStock := 0 // 未消費の食料ストック（栄養値の合計）

	enemyTable, err := raw.GetEnemyTable(master, enemyTableName)
	if err != nil {
		result.ReachedDepth = 0
		return result
	}

	// 敵データのキャッシュ
	statsCache := make(map[string]CombatantStats)
	weaponCache := make(map[string]WeaponStats)

	for depth := 1; depth <= maxDepth; depth++ {
		// フロアの敵数を決定
		enemyCount := consts.FloorEnemyBase + rng.IntN(consts.FloorEnemyRandom)
		floorTotalTurns := 0
		floorBattleCount := 0

		for range enemyCount {
			// 敵テーブルから敵を選択
			enemyName, err := raw.SelectEnemyByWeight(enemyTable, rng, depth)
			if err != nil || enemyName == "" {
				continue
			}

			enemyStats, ok := statsCache[enemyName]
			if !ok {
				var err error
				enemyStats, err = LoadCombatantFromMember(master, enemyName)
				if err != nil {
					continue
				}
				statsCache[enemyName] = enemyStats
			}

			enemyWeapon, ok := weaponCache[enemyName]
			if !ok {
				enemyWeapon, err = LoadEnemyWeapon(master, enemyName)
				if err != nil {
					enemyWeapon = WeaponStats{}
				}
				weaponCache[enemyName] = enemyWeapon
			}

			// 現在のHPで戦闘
			combatPlayer := player
			combatPlayer.HP = currentHP

			br := SimulateBattle(combatPlayer, enemyStats, currentWeapon, enemyWeapon, rng)
			currentHP -= br.DamageTaken
			floorTotalTurns += br.Turns
			floorBattleCount++

			if currentHP <= 0 {
				result.ReachedDepth = depth
				result.Died = true
				result.HPByDepth[depth] = 0
				result.WeaponByDepth[depth] = currentWeaponName
				return result
			}
		}

		// 回復前のHPを記録する
		result.HPBeforeHealByDepth[depth] = currentHP

		// フロアの移動による空腹度減少
		steps := baseStepsPerFloor + rng.IntN(randomStepsPerFloor)
		hunger.Decrease(steps)

		// フロアのアイテムドロップから回復・武器・食料を取得する
		loot := rollFloorLoot(master, enemyTableName, depth, player.HP, rng)
		currentHP += loot.healing
		if currentHP > player.HP {
			currentHP = player.HP
		}
		if loot.weapon != nil && loot.weapon.Damage > currentWeapon.Damage {
			currentWeapon = *loot.weapon
			currentWeaponName = loot.weaponName
		}
		foodStock += loot.nutrition
		result.LootIncome += loot.income

		// 空腹（33%未満）になったら食料ストックを消費する
		if hunger.GetLevel(needs) >= gc.HungerHungry && foodStock > 0 {
			hunger.Increase(foodStock)
			foodStock = 0
		}

		result.HPByDepth[depth] = currentHP
		result.WeaponByDepth[depth] = currentWeaponName
		result.WeaponDamageByDepth[depth] = currentWeapon.Damage
		if floorBattleCount > 0 {
			result.AvgKillTurnsByDepth[depth] = floorTotalTurns / floorBattleCount
		}
		result.HungerByDepth[depth] = hunger.Current
	}

	result.ReachedDepth = maxDepth
	return result
}

// floorLoot はフロアで拾えたアイテムの結果
type floorLoot struct {
	healing    int
	nutrition  int
	weapon     *WeaponStats
	weaponName string
	income     int // 拾った loot を店で売ったときの手取り総額
}

// rollFloorLoot はフロアで拾えるアイテムを計算する。
// 回復アイテムと武器の両方を処理し、武器はフロア内で最も強いものを返す
func rollFloorLoot(master oapi.Raws, tableName string, depth int, playerMaxHP int, rng *rand.Rand) floorLoot {
	result := floorLoot{}

	itemTable, err := raw.GetItemTable(master, tableName)
	if err != nil {
		return result
	}

	itemCount := consts.FloorItemBase + rng.IntN(consts.FloorItemRandom)

	for range itemCount {
		draws, err := raw.DrawFromItemTable(master, itemTable, rng, depth)
		if err != nil {
			continue
		}
		for _, d := range draws {
			addFloorLoot(&result, master, d, playerMaxHP)
		}
	}

	return result
}

// addFloorLoot は拾った1山を回復量・栄養・最強の武器・売却額へ積む。
func addFloorLoot(result *floorLoot, master oapi.Raws, d raw.DrawnItem, playerMaxHP int) {
	spec, err := raw.NewItemSpec(master, d.Name)
	if err != nil {
		return
	}
	if spec.ProvidesHealing != nil {
		result.healing += spec.ProvidesHealing.Calc(playerMaxHP) * d.Count
	}
	if spec.ProvidesNutrition != nil {
		result.nutrition += spec.ProvidesNutrition.Amount * d.Count
	}
	if w, err := LoadWeaponFromItem(master, d.Name); err == nil && (result.weapon == nil || w.Damage > result.weapon.Damage) {
		result.weapon = &w
		result.weaponName = d.Name
	}
	// 拾った loot をすべて売った仮定の粗収入。装備・消費する品も一律に計上する上限指標
	if it, e := raw.FindItem(master, d.Name); e == nil {
		if net := query.CalculateSellPrice(it.Value); net > 0 {
			result.income += int(net) * d.Count
		}
	}
}

// RunBattles はN回の戦闘シミュレーションを実行し、結果のスライスを返す
func RunBattles(player, enemy CombatantStats, playerWeapon, enemyWeapon WeaponStats, n int, rng *rand.Rand) []BattleResult {
	results := make([]BattleResult, n)
	for i := range n {
		results[i] = SimulateBattle(player, enemy, playerWeapon, enemyWeapon, rng)
	}
	return results
}

// BattleStats はN回の戦闘結果の統計を提供する
type BattleStats struct {
	Results []BattleResult
}

// DPS は1ターンあたりのプレイヤーの平均与ダメージを返す
func (bs BattleStats) DPS() float64 {
	if len(bs.Results) == 0 {
		return 0
	}
	var totalDamage, totalTurns int
	for _, r := range bs.Results {
		totalDamage += r.DamageDealt
		totalTurns += r.Turns
	}
	if totalTurns == 0 {
		return 0
	}
	return float64(totalDamage) / float64(totalTurns)
}

// RunSimulations はN回のランシミュレーションを実行する
func RunSimulations(master oapi.Raws, enemyTableName string, player CombatantStats, playerWeapon WeaponStats, maxDepth int, n int, seed uint64) RunStats {
	results := make([]RunResult, n)
	for i := range n {
		rng := rand.New(rand.NewPCG(seed+uint64(i), 0))
		results[i] = SimulateRun(master, enemyTableName, player, playerWeapon, maxDepth, rng)
	}
	return RunStats{Results: results}
}
