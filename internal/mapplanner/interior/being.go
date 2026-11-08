package interior

// beingRefs は部屋の定義が置ける敵の Ref。occupant は施設の敵の表から引く住人。
// 表に無い Ref は in-game で spawn されない。loot の lootRaw と対称に保つ。
var beingRefs = map[string]bool{
	"occupant": true,
}

// IsBeingRef は being の Ref が spawn されるかを返す。
func IsBeingRef(ref string) bool {
	return beingRefs[ref]
}
