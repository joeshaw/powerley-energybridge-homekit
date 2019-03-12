package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	hclog "github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"github.com/eclipse/paho.mqtt.golang"
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
	var ip, addr string

	flag.StringVar(&ip, "ip", "", "IP address of energy bridge")
	flag.StringVar(&addr, "addr", ":9525", "Address to listen on for Prometheus exporter")
	flag.Parse()

	if ip == "" {
		log.Fatal("-ip must be provided")
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://" + ip + ":2883")
	opts.SetUsername("admin")
	opts.SetPassword("trinity")
	opts.SetClientID("powerley-energybridge-homecontrol")

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("unable to connect to %s: %v", ip, token.Error())
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
	char := characteristic.NewInt(consumptionUUID)
	char.Format = characteristic.FormatUInt16
	char.Perms = characteristic.PermsRead()
	char.Unit = "W"

	svc.AddCharacteristic(char.Characteristic)
	acc.AddService(svc)

	t, err := hc.NewIPTransport(cfg, acc)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc.OnTermination(func() {
		cancel()
		c.Disconnect(250)
		<-t.Stop()
	})

	gauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerley_energybridge_instantaneous_demand_watts",
		Help: "Current power demand in watts.",
	})

	token := c.Subscribe("#", 0, func(c mqtt.Client, m mqtt.Message) {
		if x := os.Getenv("HC_DEBUG"); x != "" {
			fmt.Println(m.Topic(), string(m.Payload()))
		}

		switch m.Topic() {
		case "announce":
			var j struct {
				EBOSVersion string `json:"eb_os_version"`
				Serial      string `json:"serial"`
			}
			if err := json.Unmarshal(m.Payload(), &j); err != nil {
				log.Printf("unable to unmarshal message payload: %v", err)
				return
			}

			acc.Info.FirmwareRevision.SetValue(j.EBOSVersion)
			acc.Info.SerialNumber.SetValue(j.Serial)

		case "_zigbee_metering/event/metering/instantaneous_demand":
			var j struct {
				Demand int `json:"demand"`
			}
			if err := json.Unmarshal(m.Payload(), &j); err != nil {
				log.Printf("unable to unmarshal message payload: %v", err)
				return
			}

			char.SetValue(j.Demand)
			gauge.Set(float64(j.Demand))
		}
	})
	token.Wait()
	if err := token.Error(); err != nil {
		log.Fatalf("unable to subscribe to MQTT messages: %v", err)
	}

	go loopRefresh(ctx, c)
	go promExporter(ctx, addr)

	log.Println("Starting transport...")
	t.Start()
}

func loopRefresh(ctx context.Context, c mqtt.Client) {
	if err := refresh(c); err != nil {
		log.Printf("Unable to refresh subscription: %v", err)
	}

	// Instantaneous readings expire after 5 minutes, so
	// refresh every 3.
	t := time.NewTicker(180 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			if err := refresh(c); err != nil {
				log.Printf("Unable to refresh subscription: %v", err)
			}
		}
	}
}

func refresh(c mqtt.Client) error {
	payload := fmt.Sprintf(`{"request_id":"%x"}`, time.Now().UnixNano())
	token := c.Publish("_zigbee_metering/request/is_app_open", 0, false, []byte(payload))
	token.Wait()
	return token.Error()
}

func promExporter(ctx context.Context, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	mux.Handle("/metrics", promhttp.Handler())

	log.Printf("Starting Prometheus exporter on %s", addr)

	s := http.Server{
		Addr:    addr,
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
