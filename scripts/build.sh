#!/bin/bash
set -eux

##################################
# クロスコンパイルするスクリプト
##################################

# 定数的
SOURCE=app
BUILD_STAGE_TARGET=base
BUILDER_IMAGE_NAME=base
APP_NAME=ruins

# 変数的
APP_VERSION=v0.0.0  # タグ
APP_COMMIT=0000000  # 短縮ハッシュ
APP_DATE=`date +%Y-%m-%d` # 日付

cd `dirname $0`
cd ../

# ================

function is_git_repo {
    echo `git rev-parse --is-inside-work-tree`
}

if [ $(is_git_repo) = "true" ]; then
    APP_VERSION=`git describe --tag --abbrev=0`
    APP_COMMIT=`git rev-parse --short HEAD`
fi

# ================

# cmd <output> <GOOS> <GOARCH> <CGO>
cmd() {
    output=$1
    goos=$2
    goarch=$3
    cgo=$4

    docker run \
           --rm \
           -u "$(id -u):$(id -g)" \
           -w /work \
           -v $PWD:/work \
           -v $HOME/go/pkg/mod:/go/pkg/mod \
           -v $HOME/.cache/go-build:/tmp/go-build \
           --env GOCACHE=/tmp/go-build \
           --env GOOS=$goos \
           --env GOARCH=$goarch \
           --env CGO_ENABLED=$cgo \
           $BUILDER_IMAGE_NAME \
           go build -o $output -buildvcs=false -ldflags "-X github.com/kijimaD/ruins/internal/consts.AppVersion=$APP_VERSION -X github.com/kijimaD/ruins/internal/consts.AppCommit=$APP_COMMIT -X github.com/kijimaD/ruins/internal/consts.AppDate=$APP_DATE" .
}

start() {
    # Docker内でコンパイルするのでホストマシンにGo処理系は必要ではないのだが、キャッシュディレクトリをマウントするので先に存在する必要がある
    mkdir -p $HOME/go/pkg/mod
    mkdir -p $HOME/.cache/go-build

    docker build . --target $BUILD_STAGE_TARGET -t $BUILDER_IMAGE_NAME

    cmd "bin/${APP_NAME}_linux_amd64" linux amd64 1
    cmd "bin/${APP_NAME}_windows_amd64" windows amd64 0

    # no such instruction になる...
    # cmd "${APP_NAME}_linux_arm64" linux arm64 1
    # cmd "${APP_NAME}_windows_arm64" windows arm64 0

    cmd "wasm/game.wasm" js wasm 0
}

start
