package main

import (
	"log"
	"os/exec"

	"github.com/elemecca/go-hotplug"
)

type Device struct {
	VendorID  int
	ProductID int
}

var KEYBOARDS = map[string]Device{
	"rama-m60a": Device{VendorID: 0x5241, ProductID: 0x060a},
	"rama-kara": Device{VendorID: 0x5241, ProductID: 0x4b52},
}

func main() {
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
					})
					if err != nil {
						log.Println(err)
					}

					disableIntegratedKeyboard(true)
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

	cmd = exec.Command("swaymsg", "input",
		"1:1:AT_Translated_Set_2_keyboard",
		"events", status)
	cmd.Run()
	cmd = exec.Command("swaymsg", "input",
		"4012:2782:keyd_virtual_keyboard",
		"events", status)
	cmd.Run()
}