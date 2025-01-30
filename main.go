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
	freqMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shelly_plug_frequency_hertz",
			Help: "Current frequency in hertz",
		},
		[]string{"device", "host"},
	)
	voltageMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shelly_plug_voltage_volts",
			Help: "Current voltage in volts",
		},
		[]string{"device", "host"},
	)
	currentMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shelly_plug_current_amperes",
			Help: "Current in amperes",
		},
		[]string{"device", "host"},
	)
	temperatureMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shelly_plug_temperature_celsius",
			Help: "Current temperature in Celsius",
		},
		[]string{"device", "host"},
	)
	outputMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shelly_plug_output",
			Help: "Output status of the Shelly plug (1 for on, 0 for off)",
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
	prometheus.MustRegister(freqMetric)
	prometheus.MustRegister(voltageMetric)
	prometheus.MustRegister(currentMetric)
	prometheus.MustRegister(temperatureMetric)
	prometheus.MustRegister(outputMetric)

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

	// Expand environment variables for username and password
	for i, plug := range config.ShellyPlugs {
		config.ShellyPlugs[i].Password = os.ExpandEnv(plug.Password)
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
				err := getMetrics(client, plug)
				if err != nil {
					slog.Error("Failed to get power data",
						"device", plug.Name,
						"host", plug.Host,
						"error", err)
					return
				}
			}(plug)
		}
	}
}

func getMetrics(client *http.Client, plug ShellyPlug) error {
	url := fmt.Sprintf("http://%s/rpc/Shelly.GetStatus", plug.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if plug.Username != "" && plug.Password != "" {
		req.SetBasicAuth(plug.Username, plug.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status struct {
		Switch0 struct {
			Output      bool    `json:"output"`
			Apower      float64 `json:"apower"`
			Freq        float64 `json:"freq"`
			Voltage     float64 `json:"voltage"`
			Current     float64 `json:"current"`
			Temperature struct {
				Celsius    float64 `json:"tC"`
				Fahrenheit float64 `json:"tF"`
			} `json:"temperature"`
		} `json:"switch:0"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return err
	}

	powerMetric.WithLabelValues(plug.Name, plug.Host).Set(status.Switch0.Apower)
	freqMetric.WithLabelValues(plug.Name, plug.Host).Set(status.Switch0.Freq)
	voltageMetric.WithLabelValues(plug.Name, plug.Host).Set(status.Switch0.Voltage)
	currentMetric.WithLabelValues(plug.Name, plug.Host).Set(status.Switch0.Current)
	temperatureMetric.WithLabelValues(plug.Name, plug.Host).Set(status.Switch0.Temperature.Celsius)
	if status.Switch0.Output {
		outputMetric.WithLabelValues(plug.Name, plug.Host).Set(1)
	} else {
		outputMetric.WithLabelValues(plug.Name, plug.Host).Set(0)
	}

	return nil
}
