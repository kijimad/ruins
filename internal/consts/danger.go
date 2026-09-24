package consts

// Danger は生成の危険度。1 始まりで大きいほど厳しく、北進度と経過日数から決まる難易度軸。
// 敵・アイテムを minDanger/maxDanger 帯で絞る。日数や深度など他の int と型で区別する。
type Danger int
