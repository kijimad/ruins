package consts

// Danger は危険度で数える量。1 始まりで大きいほど厳しい。北進度と経過日数から決まり、
// 敵・アイテムの生成テーブルを minDanger/maxDanger 帯で絞る難易度軸。
//
// 生の int と混ぜないための意味型で取り違えをコンパイラが弾く。生成テーブルの oapi.DangerLevel は
// 別レイヤの int エイリアスなので、raw のデータ層と往復するときだけ int へ明示変換する。
type Danger int
