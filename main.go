package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	hclog "github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// Newly generated UUID for Powerley Energy Bridge
	// If there is a better, existing service type, use that instead.
	typePowerMonitor = "0A32858F-6CF9-4354-9B82-438D0261B7E2"

	// Power consumption, used by the Eve app.  See
	// https://gist.github.com/gomfunkel/b1a046d729757120907c and
	// https://gist.github.com/simont77/3f4d4330fa55b83f8ca96388d9004e7d
	// for more info.
	consumptionUUID = "E863F10D-079E-48FF-8F27-9C2605A29F52"
)

func main() {
	var (
		ip       string
		version  int
		interval int
	)

	flag.StringVar(&ip, "ip", "", "IP address of energy bridge")
	flag.IntVar(&version, "version", 2, "Energy bridge version")
	flag.IntVar(&interval, "interval", 5, "Interval to update from energy bridge, in seconds")
	flag.Parse()

	if ip == "" {
		log.Fatal("-ip must be provided")
	}

	var url string
	switch version {
	case 1:
		url = fmt.Sprintf("http://%s/instantaneousdemand", ip)
	case 2:
		url = fmt.Sprintf("http://%s:8888/zigbee/se/instantaneousdemand", ip)
	default:
		log.Fatal("Support versions: 1, 2")
	}

	if x := os.Getenv("HC_DEBUG"); x != "" {
		hclog.Debug.Enable()
	}

	cfg := hc.Config{
		Pin:         "00102003",
		StoragePath: filepath.Join(os.Getenv("HOME"), ".homecontrol", "energybridge"),
	}

	info := accessory.Info{
		Name:         "Energy Bridge",
		Manufacturer: "Powerley",
	}

	acc := accessory.New(info, accessory.TypeSensor)
	svc := service.New(typePowerMonitor)
	c := characteristic.NewInt(consumptionUUID)
	c.Format = characteristic.FormatUInt16
	c.Perms = characteristic.PermsRead()
	c.Unit = "W"

	svc.AddCharacteristic(c.Characteristic)
	acc.AddService(svc)

	t, err := hc.NewIPTransport(cfg, acc)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc.OnTermination(func() {
		cancel()
		<-t.Stop()
	})

	gauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerley_energybridge_instantaneous_demand",
		Help: "Current power demand in watts.",
	})

	go update(ctx, c, gauge, time.Duration(interval)*time.Second, url)
	go promExporter(ctx, url)

	log.Println("Starting transport...")
	t.Start()
}

func update(ctx context.Context, c *characteristic.Int, gauge prometheus.Gauge, interval time.Duration, url string) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			p, err := getPower(url)
			if err != nil {
				log.Printf("Unable to get power reading: %v", err)
				continue
			}

			c.SetValue(p)
			gauge.Set(float64(p))

		case <-ctx.Done():
			return
		}
	}
}

func promExporter(ctx context.Context, url string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	mux.Handle("/metrics", promhttp.Handler())

	log.Printf("Starting Prometheus exporter on :9525")

	s := http.Server{
		Addr:    ":9525",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		s.Shutdown(context.Background())
	}()

	if err := s.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			return
		}

		log.Fatalf("cannot start Prometheus exporter: %v", err)
	}
}

func getPower(url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Invalid status code: %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	parts := strings.Split(string(data), " ")
	if len(parts) != 2 || parts[1] != "kW" {
		return 0, fmt.Errorf("Unexpected energy usage output: %q", data)
	}

	kw, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}

	return int(kw * 1000), nil

}
