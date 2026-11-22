package components

// Switch は点け消しできる物の状態。持つ物だけが「使う」で点け消しでき、オフの間は働かない。
// 灯りに限らない部品で、灯りはオフの間は照らさず残量も減らない
type Switch struct {
	On bool
}
