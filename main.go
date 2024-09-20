package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/elemecca/go-hotplug"
	"github.com/mrusme/usbec/config"
)

type Device struct {
	VendorID   int
	ProductID  int
	PrettyName string
}

var ATTACHED_DEVICES map[string]config.Device
var cfg config.Config

func main() {
	var err error

	if cfg, err = config.Cfg(); err != nil {
		panic(err)
	}

	ATTACHED_DEVICES = make(map[string]config.Device)

	listener, _ := hotplug.New(
		hotplug.DevIfHid,
		func(devIf *hotplug.DeviceInterface) {
			var err error
			var errs []error

			usb, err := devIf.Device.Up(hotplug.DevUsbDevice)
			if err != nil {
				puts("%s", err)
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
					puts("%s", err)
				}
			}

			for _, device := range cfg.Devices {
				if device.VendorID == vendorId && device.ProductID == productId {
					if _, isAttached := ATTACHED_DEVICES[device.ID]; isAttached {
						puts("'%s' already attached, skipping\n", device.ID)
						continue
					}

					puts(
						"Attached '%s' bus=%d address=%d vid=%04x pid=%04x dev=%s\n",
						device.ID, busNumber, address, vendorId, productId, devIf.Path,
					)

					err = devIf.OnDetach(func() {
						puts(
							"Detached '%s' bus=%d address=%d vid=%04x pid=%04x dev=%s\n",
							device.ID, busNumber, address, vendorId, productId, devIf.Path,
						)

						errs := runCommands(device.On.Detach)
						delete(ATTACHED_DEVICES, device.ID)

						text := "The " + device.PrettyName + " has been detached."
						estr := errstrFromErrs(device.On.Detach, errs)
						if estr != "" {
							text += "\nThe following commands returned errors:\n" + estr
						}
						notify(device.NotificationIcon,
							device.PrettyName+" detached!",
							text,
						)
					})
					if err != nil {
						puts("%s", err)
						continue
					}

					errs := runCommands(device.On.Attach)
					ATTACHED_DEVICES[device.ID] = device

					text := "The " + device.PrettyName + " has been attached."
					estr := errstrFromErrs(device.On.Attach, errs)
					if estr != "" {
						text += "\nThe following commands returned errors:\n" + estr
					}
					notify(device.NotificationIcon,
						device.PrettyName+" attached!",
						text,
					)
				}
			}

		},
	)

	err = listener.Listen()
	if err != nil {
		panic(err)
	}

	select {}
}

func runCommands(cmds []config.Cmd) []error {
	var errs []error

	for _, cmd := range cmds {
		ec := exec.Command(cmd.Command, cmd.Args...)
		err := ec.Run()
		errs = append(errs, err)
	}

	return errs
}

func notify(icon, title, text string) {
	if cfg.Notifications == false {
		return
	}

	var re = regexp.MustCompile(`(?m)\$\{{0,1}(\w+)\}{0,1}`)
	for _, match := range re.FindAllStringSubmatch(icon, -1) {
		fullvar := match[0]
		varname := match[1]

		icon = strings.Replace(icon, fullvar, os.Getenv(varname), 1)
	}

	puts("Running notify-send with -i %s ...", icon)

	cmd := exec.Command("notify-send",
		"-i", icon,
		"-a", "usbec",
		title, text)
	cmd.Env = os.Environ()
	cmd.Run()
}

func puts(format string, v ...any) {
	if cfg.Debug {
		log.Printf(format+"\n", v)
	}
}

func errstrFromErrs(cmds []config.Cmd, errs []error) string {
	errsstr := ""
	for i, err := range errs {
		if err != nil {
			errsstr = fmt.Sprintf("%s`%s` (%v)\n", errsstr, cmds[i].Command, cmds[i].Args)
		}
	}

	return errsstr
}
