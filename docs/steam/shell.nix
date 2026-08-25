# Steam アセット生成に使うツールをまとめる nix シェル。
# 使い方: nix-shell docs/steam/shell.nix
#   ImageMagick(magick) と Python 3.12 を PATH に載せる。
#   compose.sh / gen_logo.sh / gen_icon.sh は magick を使う。
#   gen_master.py はこの python3.12 で venv を作り pip-deps.txt を入れて動かす。
#   SD 依存 torch/diffusers は pip の wheel なので nix では入れない。
#
# チャンネル無しの Determinate Nix でも動くよう、nixpkgs は flake レジストリから取る。
let
  pkgs = (builtins.getFlake "nixpkgs").legacyPackages.${builtins.currentSystem};
in
pkgs.mkShell {
  packages = [
    pkgs.imagemagick
    pkgs.python312
  ];
}
