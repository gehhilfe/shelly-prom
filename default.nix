{ pkgs ? import <nixpkgs> { } }:

pkgs.buildGoModule {
  pname = "shelly-prom";
  version = "0.0.1";
  src = ./.;

  vendorHash = "sha256-NnvB20rORPS5QF5enbb5KpWaKZ70ybSgfd7wjk21/Cg="; # Replace with actual hash after first build

  meta = with pkgs.lib; {
    description = "Scrapes information from a shelly plug and provides them as prometheus metrics";
    homepage = "https://github.com/gehhilfe/ShellyPlug";
    license = licenses.mit;
    maintainers = [ maintainers.gehhilfe ];
  };
}