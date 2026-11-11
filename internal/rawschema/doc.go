// Package rawschema は raw のデータの型を持つ。
//
// # 単一の出どころ
//
// raw のデータの形、値の制約、行の間の参照は、このディレクトリの CUE のファイルに書く。Go の型は
// gen がそこから models_gen.go へ生成し、手で書かない。CUE を変えたら make generate で作り直す。
//
// # 使い分け
//
// このパッケージは生成した型と、CUE のファイルを埋め込んだ SchemaFiles だけを持ち、CUE のライブラリを
// import しない。ゲームの本体に CUE を持ち込まないためである。CUE での検証は rawcheck が行い、テストだけが
// それを使う。raw は埋め込みのデータなので、検証は CI のテストで保証すれば足りる。
//
// # CUE の書き方
//
// gen は CUE を構文として読み、限られた書き方だけを Go へ写す。写し方と属性 @rawgen は gen の doc にある。
package rawschema
