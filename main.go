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
	"github.com/brutella/hc/service"
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

	go update(ctx, c, time.Duration(interval)*time.Second, url)

	log.Println("Starting transport...")
	t.Start()
}

func update(ctx context.Context, c *characteristic.Int, interval time.Duration, url string) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Error fetching energy usage: %v", err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Printf("Invalid status code: %d", resp.StatusCode)
				continue
			}

			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading energy usage: %v", err)
				continue
			}

			parts := strings.Split(string(data), " ")
			if len(parts) != 2 || parts[1] != "kW" {
				log.Printf("Unexpected energy usage output: %q", data)
				continue
			}

			kw, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				log.Printf("Unable to parse energy usage: %q", data)
				continue
			}

			c.SetValue(int(kw * 1000))

		case <-ctx.Done():
			return
		}
	}
}
