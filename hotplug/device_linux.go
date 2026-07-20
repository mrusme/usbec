//go:build linux

package hotplug

import (
	"errors"
	"runtime"
)

/*
	#cgo pkg-config: libudev
	#include <libudev.h>
	#include <stdlib.h>
*/
import "C"

type DeviceInterface struct {
	Path   string
	Class  InterfaceClass
	Device *Device

	listener *Listener
	devpath  string
	inArrive bool
}

func (devIf *DeviceInterface) OnDetach(callback func()) error {
	if !devIf.inArrive {
		return errors.New("OnDetach must be called from the arrive callback")
	}

	devIf.listener.detachCb[devIf.devpath] = append(
		devIf.listener.detachCb[devIf.devpath],
		callback,
	)
	return nil
}

type Device struct {
	Path  string
	Class DeviceClass

	listener *Listener
	udev     *C.struct_udev_device
}

func newDevice(listener *Listener, udev *C.struct_udev_device) *Device {
	var class DeviceClass
	for maybeClass, cond := range deviceClassCondition {
		if cond.matches(udev) {
			class = maybeClass
			break
		}
	}

	dev := &Device{
		Path:     C.GoString(C.udev_device_get_syspath(udev)),
		Class:    class,
		listener: listener,
		udev:     udev,
	}

	C.udev_device_ref(udev)
	runtime.AddCleanup(dev, func(udev *C.struct_udev_device) {
		C.udev_device_unref(udev)
	}, udev)

	return dev
}

func (dev *Device) Parent() (*Device, error) {
	parent := C.udev_device_get_parent(dev.udev)
	if parent == nil {
		return nil, errors.New("no parent")
	}

	return newDevice(dev.listener, parent), nil
}

func (dev *Device) Up(class DeviceClass) (*Device, error) {
	cond := deviceClassCondition[class]
	if cond == nil {
		return nil, errors.New("unsupported DeviceClass")
	}

	parent := dev.udev
	for {
		parent = C.udev_device_get_parent(parent)
		if parent == nil {
			return nil, errors.New("no matching ancestor found")
		}

		if cond.matches(parent) {
			return newDevice(dev.listener, parent), nil
		}
	}
}

func (dev *Device) getSysAttrLong(attr *C.char, base int) (int, error) {
	val := C.udev_device_get_sysattr_value(dev.udev, attr)
	if val == nil {
		return 0, errors.New("attribute not found")
	}

	return (int)(C.strtol(val, nil, (C.int)(base))), nil
}

func (dev *Device) BusNumber() (int, error) {
	return dev.getSysAttrLong(cstrBusnum, 10)
}

func (dev *Device) Address() (int, error) {
	return dev.getSysAttrLong(cstrDevnum, 10)
}

func (dev *Device) VendorId() (int, error) {
	return dev.getSysAttrLong(cstrIdVendor, 16)
}

func (dev *Device) ProductId() (int, error) {
	return dev.getSysAttrLong(cstrIdProduct, 16)
}
