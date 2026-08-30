package components

// Burning は今まさに燃えているエンティティに付く状態。地面のタイルに宿り、燃料を収納として抱える。
// Remaining は今燃えている最上段の燃料の残り。0 以下で収納の次の燃料へ移り、次が無ければ外れて火が消える。
// Burning がある間だけ同じエンティティの HeatSource が暖房として効く。
type Burning struct {
	Remaining int // 今燃やしている最上段の燃料の残り。0 以下で収納の次へ移る。次が無ければ鎮火
}
