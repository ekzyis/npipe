{
  description = "Share files between devices on the same network";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      commit = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAllSystems (pkgs: rec {
        default = npipe;

        npipe = pkgs.buildGoModule {
          pname = "npipe";
          version = commit;
          src = ./.;
          vendorHash = null;
          ldflags = [
            "-s"
            "-w"
            "-X main.buildCommit=${commit}"
          ];
          meta = {
            description = "Share files between devices on the same network";
            homepage = "https://npipe.io/";
            license = nixpkgs.lib.licenses.mit;
            mainProgram = "npipe";
          };
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
          ];
        };
      });

      formatter = forAllSystems (pkgs: pkgs.nixfmt-tree);
    };
}
