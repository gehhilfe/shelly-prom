package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Port        int           `json:"port"`
	Interval    time.Duration `json:"interval_seconds"`
	ListenAddr  string        `json:"listen_addr"`
	ShellyPlugs []ShellyPlug  `json:"shelly_plugs"`
}

type ShellyPlug struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ShellyStatus struct {
	SwitchID int     `json:"id"`
	APower   float64 `json:"apower"`
}

var (
	powerMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shelly_plug_power_watts",
			Help: "Current power consumption in watts",
		},
		[]string{"device", "host"},
	)
)

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Register metrics
	prometheus.MustRegister(powerMetric)
	// Start monitoring loop
	go monitorShellyDevices(config)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf("%s:%d", config.ListenAddr, config.Port), nil)
}

func loadConfig() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// Fallback paths
		paths := []string{
			"/etc/shelly-prom/config.json",
			"config.json",
		}

		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return nil, fmt.Errorf("no configuration file found")
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config read error: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}
	return &config, nil
}

func monitorShellyDevices(config *Config) {
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(time.Duration(config.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, plug := range config.ShellyPlugs {
			go func(plug ShellyPlug) {
				power, err := getShellyPower(client, plug)
				if err != nil {
					slog.Error("Failed to get power data",
						"device", plug.Name,
						"host", plug.Host,
						"error", err)
					return
				}

				powerMetric.WithLabelValues(plug.Name, plug.Host).Set(power)
			}(plug)
		}
	}
}

func getShellyPower(client *http.Client, plug ShellyPlug) (float64, error) {
	url := fmt.Sprintf("http://%s/rpc/Shelly.GetStatus", plug.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	if plug.Username != "" && plug.Password != "" {
		req.SetBasicAuth(plug.Username, plug.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status struct {
		Switch0 struct {
			Apower float64 `json:"apower"`
		} `json:"switch:0"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return 0, err
	}

	return status.Switch0.Apower, nil
}
