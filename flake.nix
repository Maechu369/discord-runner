{
  description = "zero settings";
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/26.05";
    flake-parts.url = "github:hercules-ci/flake-parts";
    git-hooks-nix = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" ];
      imports = [
        inputs.git-hooks-nix.flakeModule
      ];
      perSystem =
        {
          config,
          self',
          inputs',
          pkgs,
          ...
        }:
        {
          pre-commit = {
            settings.hooks = {
              nixfmt.enable = true;
              git-secrets = {
                enable = true;
                name = "git-secrets";
                description = "Prevent committing secrets with git-secrets";
                entry = "${pkgs.git-secrets}/bin/git-secrets scan";
                pass_filenames = false;
              };
            };
          };
          devShells.default = pkgs.mkShell {
            shellHook = ''
              ${config.pre-commit.shellHook}
              ${pkgs.git-secrets}/bin/git-secrets --register-aws
              ${pkgs.git-secrets}/bin/git-secrets --add-provider -- cat ${./secrets-patterns.txt}
              echo "regenerate .pre-commit-config.yaml"
            '';
            packages = config.pre-commit.settings.enabledPackages ++ [ pkgs.go ];
          };
        };
    };
}
