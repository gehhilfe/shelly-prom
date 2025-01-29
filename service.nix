{ config, lib, pkgs, ... }:

let
  cfg = config.services.shelly-prom;
  package = pkgs.callPackage ./default.nix {};
in {
  options.services.shelly-prom = {
    enable = lib.mkEnableOption "shelly-prom";
    listenAddr = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1";
      description = "Address to listen on";
    };
    port = lib.mkOption {
      type = lib.types.int;
      default = 50678;
      description = "Port to listen on";
    };
    intervalSeconds = lib.mkOption {
      type = lib.types.int;
      default = 10;
      description = "Polling interval in seconds";
    };
    shellyPlugs = lib.mkOption {
      type = lib.types.listOf (lib.types.submodule {
        options = {
          name = lib.mkOption {
            type = lib.types.str;
            description = "Descriptive name for the plug";
          };
          host = lib.mkOption {
            type = lib.types.str;
            description = "IP/hostname of Shelly Plug";
          };
          username = lib.mkOption {
            type = lib.types.str;
            default = "";
            description = "Authentication username";
          };
          password = lib.mkOption {
            type = lib.types.str;
            default = "";
            description = "Environment variable name containing password";
          };
        };
      });
      default = [];
      description = "List of Shelly Plugs to monitor";
    };
  };

  config = lib.mkIf cfg.enable {
    users.users.shelly-prom = {
      isSystemUser = true;
      group = "shelly-prom";
      description = "Shelly Prom service user";
      home = "/var/lib/shelly-prom";
      createHome = true;
    };
    users.groups.shelly-prom = {};

    systemd.services.shelly-prom = {
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        User = "shelly-prom";
        Group = "shelly-prom";
        ExecStart = "${package}/bin/shelly-prom";
        WorkingDirectory = "/var/lib/shelly-prom";
        Restart = "on-failure";
        RestartSec = "5s";
        EnvironmentFile = "/etc/shelly-prom/config.env";
        # Security settings
        CapabilityBoundingSet = "";
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        RestrictAddressFamilies = [ "AF_INET" "AF_INET6" ];
        RestrictNamespaces = true;
        RestrictRealtime = true;
        SystemCallFilter = [ "~@cpu-emulation @debug @keyring @mount @obsolete @privileged @setuid" ];
      };
    };

    environment.etc."shelly-prom/config.json".text = builtins.toJSON {
      port = cfg.port;
      listen_addr = cfg.listenAddr;
      interval_seconds = cfg.intervalSeconds;
      shelly_plugs = map (plug: {
        name = plug.name;
        host = plug.host;
        username = plug.username;
        password = "\${${plug.password}}";
      }) cfg.shellyPlugs;
    };
  };
}