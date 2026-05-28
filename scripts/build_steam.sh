#!/bin/bash
set -eux

##############################################
# Steam向けクロスコンパイルするスクリプト
# https://ebitengine.org/ja/documents/steam.html
##############################################

APP_NAME=ruins
APP_VERSION=v0.0.0
APP_COMMIT=0000000
APP_DATE=`date +%Y-%m-%d`

STEAM_RUNTIME_VERSION=3.0.20250108.112707
GO_VERSION=$(go env GOVERSION)

LDFLAGS="-X github.com/kijimaD/ruins/internal/consts.AppVersion=$APP_VERSION -X github.com/kijimaD/ruins/internal/consts.AppCommit=$APP_COMMIT -X github.com/kijimaD/ruins/internal/consts.AppDate=$APP_DATE"
# Windowsではフリーズ対策を追加する
WINDOWS_LDFLAGS="$LDFLAGS -X=runtime.godebugDefault=asyncpreemptoff=1 -H=windowsgui"

cd `dirname $0`
cd ../

# ================

function is_git_repo {
    echo `git rev-parse --is-inside-work-tree`
}

if [ $(is_git_repo) = "true" ]; then
    APP_VERSION=`git describe --tag --abbrev=0`
    APP_COMMIT=`git rev-parse --short HEAD`
    # ldflags再構築
    LDFLAGS="-X github.com/kijimaD/ruins/internal/consts.AppVersion=$APP_VERSION -X github.com/kijimaD/ruins/internal/consts.AppCommit=$APP_COMMIT -X github.com/kijimaD/ruins/internal/consts.AppDate=$APP_DATE"
    WINDOWS_LDFLAGS="$LDFLAGS -X=runtime.godebugDefault=asyncpreemptoff=1 -H=windowsgui"
fi

# ================
# Steam Runtime Sniperイメージのキャッシュとビルド

mkdir -p .cache/${STEAM_RUNTIME_VERSION}

# -C - で途中から再開、--retry-all-errors で接続切れでもリトライする
# なぜかHTTP2で失敗するので避ける
CURL="curl --http1.1 --retry 5 --retry-delay 5 --retry-all-errors -C - --location --remote-name"

SNIPER_BASE=https://repo.steampowered.com/steamrt-images-sniper/snapshots/${STEAM_RUNTIME_VERSION}
SNIPER_DIR=.cache/${STEAM_RUNTIME_VERSION}

(cd $SNIPER_DIR; $CURL ${SNIPER_BASE}/com.valvesoftware.SteamRuntime.Sdk-amd64,i386-sniper-sysroot.Dockerfile)
(cd $SNIPER_DIR; $CURL ${SNIPER_BASE}/com.valvesoftware.SteamRuntime.Sdk-amd64,i386-sniper-sysroot.tar.gz)
(cd .cache;      $CURL https://golang.org/dl/${GO_VERSION}.linux-amd64.tar.gz)

# ================
# Linux (Steam Runtime Sniper)

(cd .cache/${STEAM_RUNTIME_VERSION}; docker build -f com.valvesoftware.SteamRuntime.Sdk-amd64,i386-sniper-sysroot.Dockerfile -t steamrt_sniper_amd64:latest .)

mkdir -p $HOME/go/pkg/mod
mkdir -p $HOME/.cache/go-build

docker run --rm \
    -u "$(id -u):$(id -g)" \
    --workdir=/work \
    --volume $(pwd):/work \
    --volume $HOME/go/pkg/mod:/tmp/gopath/pkg/mod \
    --volume $HOME/.cache/go-build:/tmp/gocache \
    --tmpfs /tmp/go:exec \
    steamrt_sniper_amd64:latest /bin/sh -c "
tar -C /tmp/go -xzf .cache/${GO_VERSION}.linux-amd64.tar.gz
export PATH=/tmp/go/go/bin:\$PATH
export GOPATH=/tmp/gopath
export GOCACHE=/tmp/gocache
export CGO_CFLAGS=-std=gnu99
export CGO_ENABLED=1

go build -tags steam -o bin/${APP_NAME}_linux_amd64_steam -buildvcs=false -ldflags \"$LDFLAGS\" .
"

# ================
# Windows (既存Dockerイメージを使用)

docker build . --target base -t base

docker run \
    --rm \
    -u "$(id -u):$(id -g)" \
    -w /work \
    -v $PWD:/work \
    -v $HOME/go/pkg/mod:/go/pkg/mod \
    -v $HOME/.cache/go-build:/tmp/go-build \
    --env GOCACHE=/tmp/go-build \
    --env GOOS=windows \
    --env GOARCH=amd64 \
    --env CGO_ENABLED=0 \
    base \
    go build -tags steam -o "bin/${APP_NAME}_windows_amd64_steam.exe" -buildvcs=false -ldflags "$WINDOWS_LDFLAGS" .
