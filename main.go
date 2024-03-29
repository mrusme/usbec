package main

import (
	"log"
	"os"
	"os/exec"

	"github.com/elemecca/go-hotplug"
)

type Device struct {
	VendorID   int
	ProductID  int
	PrettyName string
}

// swaymsg -t get_inputs | jq '.[].identifier'

var INTEGRATED_KEYBOARDS = []string{
	"1:1:AT_Translated_Set_2_keyboard",
	"4012:2782:keyd_virtual_keyboard",
}

var KEYBOARDS = map[string]Device{
	"rama-m60a": Device{VendorID: 0x5241, ProductID: 0x060a,
		PrettyName: "RAMA M60-A"},
	"rama-kara": Device{VendorID: 0x5241, ProductID: 0x4b52,
		PrettyName: "RAMA KARA"},
	"corne-v3": Device{VendorID: 0x4653, ProductID: 0x0001,
		PrettyName: "Corne V3"},
}

var ATTACHED_KEYBOARDS map[string]Device

func main() {
	ATTACHED_KEYBOARDS = make(map[string]Device)

	listener, _ := hotplug.New(
		hotplug.DevIfHid,
		func(devIf *hotplug.DeviceInterface) {
			var err error
			var errs []error

			usb, err := devIf.Device.Up(hotplug.DevUsbDevice)
			if err != nil {
				log.Println(err)
				return
			}

			busNumber, err := usb.BusNumber()
			errs = append(errs, err)

			address, err := usb.Address()
			errs = append(errs, err)

			vendorId, err := usb.VendorId()
			errs = append(errs, err)

			productId, err := usb.ProductId()
			errs = append(errs, err)

			for _, err = range errs {
				if err != nil {
					log.Println(err)
				}
			}

			for name, device := range KEYBOARDS {
				if device.VendorID == vendorId && device.ProductID == productId {
					if _, isAttached := ATTACHED_KEYBOARDS[name]; isAttached {
						log.Printf("'%s' already attached, skipping\n", name)
						continue
					}

					log.Printf(
						"Attached '%s' bus=%d address=%d vid=%04x pid=%04x dev=%s\n",
						name, busNumber, address, vendorId, productId, devIf.Path,
					)

					err = devIf.OnDetach(func() {
						log.Printf(
							"Detached '%s' bus=%d address=%d vid=%04x pid=%04x dev=%s\n",
							name, busNumber, address, vendorId, productId, devIf.Path,
						)

						disableIntegratedKeyboard(false)
						delete(ATTACHED_KEYBOARDS, name)

						notify("keyboard.svg",
							device.PrettyName+" detached!",
							"The "+device.PrettyName+
								" has been detached and the integrated keyboard enabled.")
					})
					if err != nil {
						log.Println(err)
						continue
					}

					disableIntegratedKeyboard(true)
					ATTACHED_KEYBOARDS[name] = device

					notify("keyboard.svg",
						device.PrettyName+" attached!",
						"The "+device.PrettyName+
							" has been attached and the integrated keyboard disabled.")
				}
			}

		},
	)

	err := listener.Listen()
	if err != nil {
		panic(err)
	}

	select {}
}

func disableIntegratedKeyboard(disable bool) {
	var cmd *exec.Cmd
	var status string = "enabled"

	if disable {
		status = "disabled"
	}

	for _, ikbd := range INTEGRATED_KEYBOARDS {
		cmd = exec.Command("swaymsg", "input",
			ikbd,
			"events", status)
		cmd.Run()
	}
}

func notify(icon, title, text string) {
	cmd := exec.Command("notify-send",
		"-i", os.Getenv("ICONS_PATH")+"/"+icon,
		"-a", "usbec",
		title, text)
	cmd.Run()
}
