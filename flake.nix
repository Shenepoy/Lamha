{
  description = "Lamha — a GTK4, desktop-portal screenshot tool";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in {
      devShells = forAllSystems (system:
        let pkgs = import nixpkgs { inherit system; };
        in {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gopls
              pkgs.gcc
              pkgs.gtk4
              pkgs.gdk-pixbuf
              pkgs.librsvg
              pkgs.gobject-introspection
              pkgs.pkg-config
              pkgs.gnome-screenshot
              pkgs.noto-fonts
              pkgs.fontconfig
            ];
          };
        });
    };
}
