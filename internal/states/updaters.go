package states

import w "github.com/kijimaD/ruins/internal/world"

// runUpdaters は指定した updater を world.Updaters から名前で引いて順に回す。
// メニュー state が HUD 用の再計算 system を毎フレーム回すのに使う。渡すインスタンスは
// 名前の取得にだけ使い、状態を持つ登録済みインスタンスの実体を実行する。
func runUpdaters(world w.World, updaters ...w.Updater) error {
	for _, updater := range updaters {
		sys, ok := world.Updaters[updater.String()]
		if !ok {
			continue
		}
		if err := sys.Update(world); err != nil {
			return err
		}
	}
	return nil
}
