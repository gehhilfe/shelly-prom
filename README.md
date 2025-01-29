# Shelly-Prom

`shelly-prom` is a lightweight Prometheus exporter that collects power consumption data from Shelly smart plugs and exposes it as metrics.

## Features

- Scrapes power usage from Shelly smart plugs
- Exposes metrics via an HTTP endpoint (`/metrics`)
- Runs as a NixOS service
- Supports multiple Shelly plugs with authentication

## Installation on NixOS

### Enable the Service

To enable `shelly-prom` on NixOS, add the following to your NixOS configuration (`configuration.nix`):

```nix
services.shelly-prom = {
  enable = true;
  listenAddr = "0.0.0.0";  # Change if necessary
  port = 50678;
  intervalSeconds = 10;
  shellyPlugs = [
    {
      name = "LivingRoom";
      host = "192.168.1.10";
      username = "admin";
      password = "SHELLY_PASS_ENV_VAR"; # Store password in an environment variable
    }
  ];
};
```

### Rebuild System

After adding the configuration, rebuild your NixOS system:

```sh
sudo nixos-rebuild switch
```

## Configuration

`shelly-prom` reads its configuration from `/etc/shelly-prom/config.json`, which is managed by NixOS. The service polls Shelly devices at a specified interval and exposes metrics in Prometheus format.

Example JSON configuration:

```json
{
  "port": 50678,
  "listen_addr": "0.0.0.0",
  "interval_seconds": 10,
  "shelly_plugs": [
    {
      "name": "LivingRoom",
      "host": "192.168.1.10",
      "username": "admin",
      "password": "${SHELLY_PASS_ENV_VAR}"
    }
  ]
}
```

## Metrics

Once running, `shelly-prom` exposes metrics on the configured port. You can test the endpoint with:

```sh
curl http://localhost:50678/metrics
```

Example output:

```
shelly_plug_power_watts{device="LivingRoom",host="192.168.1.10"} 5.3
```

## Security Considerations

- **Environment Variables for Passwords**: Store passwords in environment variables rather than hardcoding them.
- **Restrict Access**: Consider binding the service to `127.0.0.1` or using a firewall.
- **Systemd Hardening**: The service runs with security restrictions in place.

## Development

### Build from Source

To build `shelly-prom` using Nix:

```sh
nix-build
```

Or run it directly:

```sh
nix-shell --run "go run main.go"
```

### Testing

To test locally, set up a configuration file and run:

```sh
CONFIG_PATH=./config.json go run main.go
```

## License

This project is licensed under the MIT License.
