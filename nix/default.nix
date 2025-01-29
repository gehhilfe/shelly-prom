{ pkgs ? import <nixpkgs> { } }:

pkgs.buildGoModule {
  pname = "shelly-prom";
  version = "0.0.1";
  src = ./.;

  vendorSha256 = pkgs.lib.fakeSha256; # Replace with actual hash after first build

  meta = with pkgs.lib; {
    description = "Scrapes information from a shelly plug and provides them as prometheus metrics";
    homepage = "https://github.com/gehhilfe/ShellyPlug";
    license = licenses.mit;
    maintainers = [ maintainers.gehhilfe ];
  };
}