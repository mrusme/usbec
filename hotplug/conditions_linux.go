//go:build linux

package hotplug

/*
	#cgo pkg-config: libudev
	#include <libudev.h>
	#include <string.h>
*/
import "C"

var (
	cstrUdev      = C.CString("udev")
	cstrAdd       = C.CString("add")
	cstrRemove    = C.CString("remove")
	cstrDevtype   = C.CString("DEVTYPE")
	cstrBusnum    = C.CString("busnum")
	cstrDevnum    = C.CString("devnum")
	cstrIdVendor  = C.CString("idVendor")
	cstrIdProduct = C.CString("idProduct")

	cstrHidraw       = C.CString("hidraw")
	cstrUsbmisc      = C.CString("usbmisc")
	cstrUsblp        = C.CString("usblp")
	cstrHid          = C.CString("hid")
	cstrUsb          = C.CString("usb")
	cstrUsbDevice    = C.CString("usb_device")
	cstrUsbInterface = C.CString("usb_interface")
	cstrBlock        = C.CString("block")
	cstrDisk         = C.CString("disk")
	cstrPartition    = C.CString("partition")
)

type deviceCondition struct {
	subsystem *C.char
	devtype   *C.char
	driver    *C.char

	ancestorSubsystem *C.char
	ancestorDevtype   *C.char

	interfaceOnly bool
}

func (cond *deviceCondition) matches(dev *C.struct_udev_device) bool {
	subsystem := C.udev_device_get_subsystem(dev)
	if subsystem == nil || C.strcmp(cond.subsystem, subsystem) != 0 {
		return false
	}

	if cond.devtype != nil {
		devtype := C.udev_device_get_devtype(dev)
		if devtype == nil || C.strcmp(cond.devtype, devtype) != 0 {
			return false
		}
	}

	if cond.ancestorSubsystem != nil {
		ancestor := C.udev_device_get_parent_with_subsystem_devtype(
			dev,
			cond.ancestorSubsystem,
			cond.ancestorDevtype,
		)
		if ancestor == nil {
			return false
		}
	}

	if cond.interfaceOnly {
		dev = C.udev_device_get_parent(dev)
		if dev == nil {
			return false
		}
	}

	if cond.driver != nil {
		driver := C.udev_device_get_driver(dev)
		if driver == nil || C.strcmp(cond.driver, driver) != 0 {
			return false
		}
	}

	return true
}

var interfaceClassCondition = map[InterfaceClass]*deviceCondition{
	DevIfHid: {
		subsystem:     cstrHidraw,
		interfaceOnly: true,
	},
	DevIfPrinter: {
		subsystem:     cstrUsbmisc,
		driver:        cstrUsblp,
		interfaceOnly: true,
	},
	DevIfStorage: {
		subsystem:         cstrBlock,
		devtype:           cstrDisk,
		ancestorSubsystem: cstrUsb,
		ancestorDevtype:   cstrUsbDevice,
	},
	DevIfStoragePartition: {
		subsystem:         cstrBlock,
		devtype:           cstrPartition,
		ancestorSubsystem: cstrUsb,
		ancestorDevtype:   cstrUsbDevice,
	},
}

var deviceClassCondition = map[DeviceClass]*deviceCondition{
	DevHid: {
		subsystem: cstrHid,
	},
	DevUsbDevice: {
		subsystem: cstrUsb,
		devtype:   cstrUsbDevice,
	},
	DevUsbInterface: {
		subsystem: cstrUsb,
		devtype:   cstrUsbInterface,
	},
}
