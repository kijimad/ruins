package interior

// 依存グラフ machine は戦利品を「payload とそれを守る錠、錠を開く鍵」の三つ組にする。鍵と錠を同じ
// 建物 seed が発行し MachineID で結ぶので必ず解け、再訪で一致する。payload は最奥へ、鍵はそれより手前へ
// 置くので、プレイヤーは錠に着く前に必ず鍵を通る。
//
// 錠は幾何的には塞がないマーカーにする。塞ぐと到達性ガードが payload を永久封鎖と誤検出するため、実際の
// 封鎖は鍵を持つ gameplay 層の解釈に委ねる。組めない狭さ・部屋数なら非施錠へ決定的にフォールバックする。

// GuardMachine は施錠された戦利品1つ分の依存グラフ。最奥の payload、その戸口の錠、手前の鍵を同じ ID で結ぶ。
type GuardMachine struct {
	ID      int
	Lock    Placed // 最奥の部屋の戸口に置く錠。KindDecor のマーカーで通行は阻まない
	Key     Placed // 手前の部屋の鍵
	Payload Placed // 最奥の戦利品
}

// buildGuardMachine は rooms から施錠戦利品を1つ組む。最奥と手前で depth の異なる2部屋が要り、無ければ
// ok=false を返して呼び出し側の bare フォールバックへ委ねる。錠は最奥の戸口、payload は最奥の奥、鍵は
// 最も手前の部屋の入口近くへ置く。
func buildGuardMachine(seed uint64, rooms []Room, depths []int, payloadRef string, id int) (GuardMachine, bool) {
	deep, shallow := deepestRoom(depths), shallowestRoom(depths)
	if deep < 0 || shallow < 0 || depths[deep] == depths[shallow] || len(rooms[deep].Doorways) == 0 {
		return GuardMachine{}, false
	}

	payload := selectTiles(rooms[deep], PlaceFarFromDoor, map[Vec]bool{}, childSeed(seed, 1), 1)
	key := selectTiles(rooms[shallow], PlaceNearDoor, map[Vec]bool{}, childSeed(seed, 2), 1)
	if len(payload) == 0 || len(key) == 0 {
		return GuardMachine{}, false
	}

	door := rooms[deep].Doorways[0]
	return GuardMachine{
		ID:      id,
		Lock:    Placed{Kind: KindDecor, Ref: "shutter", Pos: Vec(door), MachineID: id},
		Key:     Placed{Kind: KindLoot, Ref: "keycard", Pos: key[0], MachineID: id},
		Payload: Placed{Kind: KindLoot, Ref: payloadRef, Pos: payload[0], MachineID: id},
	}, true
}

// guardedLoot は施錠戦利品の依存グラフを Placed 列にして返す。elaborate を組めれば錠・鍵・payload の三つ、
// 組めなければ payload だけを最奥へ非施錠で置く bare へ決定的にフォールバックする。
func guardedLoot(seed uint64, footprint Rect, rooms []Room, payloadRef string, id int) []Placed {
	depths := roomDepths(footprint, rooms)
	if m, ok := buildGuardMachine(seed, rooms, depths, payloadRef, id); ok {
		return []Placed{m.Lock, m.Key, m.Payload}
	}
	// bare フォールバック。最奥へ非施錠で payload だけ置く
	deep := deepestRoom(depths)
	if deep < 0 {
		return nil
	}
	tiles := selectTiles(rooms[deep], PlaceFarFromDoor, map[Vec]bool{}, childSeed(seed, 3), 1)
	if len(tiles) == 0 {
		return nil
	}
	return []Placed{{Kind: KindLoot, Ref: payloadRef, Pos: tiles[0]}}
}

// deepestRoom は入口から最も遠い部屋の添字を返す。到達不能(-1)は除く。同 depth は添字の小さい方。
func deepestRoom(depths []int) int {
	best := -1
	for i, d := range depths {
		if d < 0 {
			continue
		}
		if best < 0 || d > depths[best] {
			best = i
		}
	}
	return best
}

// shallowestRoom は入口に最も近い部屋の添字を返す。到達不能(-1)は除く。同 depth は添字の小さい方。
func shallowestRoom(depths []int) int {
	best := -1
	for i, d := range depths {
		if d < 0 {
			continue
		}
		if best < 0 || d < depths[best] {
			best = i
		}
	}
	return best
}
