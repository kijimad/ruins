package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/systems"
)

// Params はバランス導出が参照する全つまみを1つに集約したベクトル。導出はこの純関数として書き、感度・交換・探索は
// 成分を摂動する汎用機械になる。成分はゲーム定数を写した絶対値と、集計へ掛ける倍率の2種。絶対値は DefaultParams
// だけがゲーム定数から取り込み、接点を1箇所に限る。
type Params struct {
	// 時間。日換算メトリクスの共通分母
	TurnsPerDay float64

	// 生存
	MaxHunger           float64 // 最大満腹度
	HungerDrainTurns    float64 // 満腹度1点の減耗に要する期待ターン数
	HungerStarvingRatio float64 // 栄養失調に入る満腹度割合のしきい値

	// 疲労
	MaxFatigue            float64 // 最大疲労
	FatigueGainPerTurn    float64 // 起床時に毎ターン溜まる疲労量
	FatigueRecoverPerTurn float64 // 基準の寝床で毎ターン抜ける疲労量
	FatigueTiredRatio     float64 // 疲労状態に入る割合のしきい値
	FatigueExhaustedRatio float64 // 過労状態に入る割合のしきい値

	// 倍率つまみ。1が現状
	EnemyWeaponScale    float64 // 敵武器の近接ダメージ倍率
	PlayerStrengthScale float64 // 基準プレイヤーの筋力倍率
	FuelHeatScale       float64 // 燃料熱量の倍率
	LootValueScale      float64 // loot期待価値の倍率
}

// DefaultParams は現行のゲーム定数から初期化した基準ベクトルを返す。
// ゲーム定数との接点はこの関数だけに限り、定数が変われば導出も自動追従する。
func DefaultParams() Params {
	return Params{
		TurnsPerDay: float64(gc.TurnsPerDay),

		MaxHunger:           float64(gc.DefaultMaxHunger),
		HungerDrainTurns:    float64(gc.HungerDrainTurns),
		HungerStarvingRatio: gc.HungerStarvingRatio,

		MaxFatigue:            float64(gc.DefaultMaxFatigue),
		FatigueGainPerTurn:    float64(gc.FatigueGainPerTurn),
		FatigueRecoverPerTurn: float64(systems.FatigueRecoverPerTurn),
		FatigueTiredRatio:     gc.FatigueTiredRatio,
		FatigueExhaustedRatio: gc.FatigueExhaustedRatio,

		EnemyWeaponScale:    1,
		PlayerStrengthScale: 1,
		FuelHeatScale:       1,
		LootValueScale:      1,
	}
}
