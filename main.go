package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"os/signal"
	"syscall"

	tradfri "github.com/barnybug/go-tradfri"
	dashbutton "github.com/mooyoul/go-dash-button"
	"github.com/pkg/errors"
)

func main() {
	ifaceName := os.Getenv("DASH_BUTTON_IFACE")
	dashButtonMAC := os.Getenv("DASH_BUTTON_DB_MAC")
	tradfriGateway := os.Getenv("TRADFRI_IP")
	tradfriKey := os.Getenv("TRADFRI_KEY")

	var addr net.HardwareAddr

	if a, err := net.ParseMAC(dashButtonMAC); err != nil {
		panic(err)
	} else {
		addr = a
	}

	var iface *net.Interface
	if ifaceName == "" {
		ifaces, err := net.Interfaces()

		if err != nil {
			panic(err)
		}

		if len(ifaces) == 0 {
			panic("No interface found")
		}

		iface = &ifaces[0]

		fmt.Printf("Network device(%s) was selected.\n", iface.Name)
	} else {
		var err error
		iface, err = net.InterfaceByName(ifaceName)

		if err != nil {
			panic(err)
		}
	}

	inter, err := dashbutton.NewInterceptor(iface)

	if err != nil {
		panic(err)
	}

	defer inter.Close()

	client := tradfri.NewClient(tradfriGateway)

	if err := client.LoadPSK(); err != nil {
		if len(tradfriKey) == 0 {
			panic(errors.Wrap(err, "need key"))
		} else {
			client.Key = tradfriKey
		}
	}

	client.Key = tradfriKey

	if err := client.Connect(); err != nil {
		panic(err)
	}
	client.SavePSK()

	inter.Add(addr)
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGTERM, syscall.SIGHUP)
	clicks := inter.Clicks()
	prev := time.Now()
	for {
		select {
		case dev, ok := <-clicks:
			if !ok {
				fmt.Println("Shutting down")
				return
			}
			now := time.Now()
			if now.Sub(prev) < 500*time.Millisecond {
				prev = now
				continue
			}
			prev = now

			title, body := "Dash Buttonが押されました。", fmt.Sprintf("Dash Buttonが押されました。\nMAC Address: %s\nIP Address: %s\nTime Stamp: %s", dev.HardwareAddr.String(), dev.IP.String(), time.Now().String())

			fmt.Println(title, body)

			groups, err := client.ListGroups()

			if err != nil {
				panic(err)
			}

			if len(groups) == 0 {
				fmt.Println("no group found")

				continue
			}

			fmt.Println(groups)

			devices, err := client.ListDevices()

			if err != nil {
				panic(err)
			}

			d := make(map[int]*tradfri.DeviceDescription)

			for i := range devices {
				d[devices[i].DeviceID] = devices[i]
			}

			stat := true
			for _, id := range groups[0].AccessoryLink.LinkedItems.DeviceIDs {
				if _, ok := d[id]; !ok {
					continue
				}

				if d[id].ApplicationType == 0 {
					continue
				}

				for j := range d[id].LightControl {
					stat = stat && *d[id].LightControl[j].Power != 0
				}
			}

			control := tradfri.LightControl{}

			if stat {
				p := 0
				control.Power = &p
			} else {
				p := 1
				dim := tradfri.DimMax
				control.Power = &p
				control.Dim = &dim
			}

			fmt.Println("trying to toggle light ->", *control.Power != 0)

			if err := client.SetGroup(groups[0].GroupID, control); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("toggled")
			}

		case <-sigch:
			fmt.Println("Shutting down")
			return
		}
	}
}
