# Steam アセット生成に使うツールをまとめる nix シェル。
# 使い方: nix-shell docs/steam/shell.nix
#   ImageMagick(magick) と Python 3.12 を PATH に載せる。
#   compose.sh / gen_logo.sh / gen_icon.sh は magick を使う。
#   gen_master.py と gen_crash.py と gen_title.py はこの python3.12 の pillow で動かす。google-chrome は別に要る。
#
# チャンネル無しの Determinate Nix でも動くよう、nixpkgs は flake レジストリから取る。
let
  pkgs = (builtins.getFlake "nixpkgs").legacyPackages.${builtins.currentSystem};
in
pkgs.mkShell {
  packages = [
    pkgs.imagemagick
    (pkgs.python312.withPackages (ps: [ ps.pillow ]))
  ];
}
