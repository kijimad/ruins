#!/bin/bash
set -eux

###########################
# 各ステートのキャプチャを取る。
###########################
# Goテスト内で条件を変えてスクショしたいのだが、Ebitenの制限で1プロセス内で2回起動ができなかった。シェルスクリプトで別プロセスで実行することにした。各ステートで簡単にスクショを取るだけだが、ないよりはマシだろう...

cd `dirname $0`
cd ../

RUN=${RUN:-"go run ."}

cmd() {
    state=$1

    $RUN screenshot $state
}

cmd CraftMenu
cmd DebugMenu
cmd Dungeon
cmd EquipMenu
cmd FieldInfo
cmd GameOver
cmd InventoryMenu
cmd LoadMenu
cmd MainMenu
cmd Message
cmd SaveMenu
cmd ShopMenu
cmd Town
