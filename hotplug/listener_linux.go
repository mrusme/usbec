//go:build linux

package hotplug

import (
	"errors"
	"runtime"

	"golang.org/x/sys/unix"
)

/*
	#cgo pkg-config: libudev
	#include <libudev.h>
	#include <string.h>
*/
import "C"

type ListenerCallback func(devIf *DeviceInterface)

type Listener struct {
	class    InterfaceClass
	callback ListenerCallback

	condition *deviceCondition
	udev      *C.struct_udev
	monitor   *C.struct_udev_monitor
	closeChan chan struct{}
	closePipe []int
	deviceFd  int
	detachCb  map[string][]func()
}

func New(class InterfaceClass, callback ListenerCallback) (*Listener, error) {
	condition := interfaceClassCondition[class]
	if condition == nil {
		return nil, errors.New("unsupported InterfaceClass")
	}

	udev := C.udev_new()
	if udev == nil {
		return nil, errors.New("failed to create udev context")
	}

	l := &Listener{
		class:     class,
		callback:  callback,
		condition: condition,
		udev:      udev,
		deviceFd:  -1,
		detachCb:  make(map[string][]func()),
	}

	runtime.AddCleanup(l, func(udev *C.struct_udev) {
		C.udev_unref(udev)
	}, udev)

	return l, nil
}

func (l *Listener) Listen() error {
	if l.monitor != nil {
		return errors.New("listener is already listening")
	}

	monitor := C.udev_monitor_new_from_netlink(l.udev, cstrUdev)
	if monitor == nil {
		return errors.New("failed to create udev monitor")
	}

	deviceFd, closePipe, err := startMonitor(monitor, l.condition)
	if err != nil {
		C.udev_monitor_unref(monitor)
		return err
	}

	l.monitor = monitor
	l.deviceFd = deviceFd
	l.closePipe = closePipe
	l.closeChan = make(chan struct{})

	go l.eventPump()
	return nil
}

func startMonitor(
	monitor *C.struct_udev_monitor,
	cond *deviceCondition,
) (int, []int, error) {
	res := C.udev_monitor_filter_add_match_subsystem_devtype(
		monitor,
		cond.subsystem,
		cond.devtype,
	)
	if res < 0 {
		return -1, nil, errors.New("failed to add udev filter")
	}

	if C.udev_monitor_enable_receiving(monitor) < 0 {
		return -1, nil, errors.New("failed to enable udev monitor")
	}

	deviceFd := (int)(C.udev_monitor_get_fd(monitor))
	if deviceFd < 0 {
		return -1, nil, errors.New("failed to get udev monitor fd")
	}

	if err := prepareFd(deviceFd); err != nil {
		return -1, nil, err
	}

	closePipe := make([]int, 2)
	if err := unix.Pipe(closePipe); err != nil {
		return -1, nil, err
	}

	return deviceFd, closePipe, nil
}

func prepareFd(fd int) error {
	flags, err := unix.FcntlInt((uintptr)(fd), unix.F_GETFD, 0)
	if err != nil {
		return err
	}
	if flags&unix.FD_CLOEXEC == 0 {
		_, err = unix.FcntlInt((uintptr)(fd), unix.F_SETFD, flags|unix.FD_CLOEXEC)
		if err != nil {
			return err
		}
	}

	flags, err = unix.FcntlInt((uintptr)(fd), unix.F_GETFL, 0)
	if err != nil {
		return err
	}
	if flags&unix.O_NONBLOCK == 0 {
		_, err = unix.FcntlInt((uintptr)(fd), unix.F_SETFL, flags|unix.O_NONBLOCK)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Listener) Stop() error {
	if l.monitor == nil {
		return errors.New("listener is not listening")
	}

	if err := unix.Close(l.closePipe[1]); err != nil {
		return err
	}

	<-l.closeChan

	unix.Close(l.closePipe[0])

	l.closeChan = nil
	l.closePipe = nil

	C.udev_monitor_unref(l.monitor)
	l.monitor = nil
	l.deviceFd = -1

	return nil
}

func (l *Listener) eventPump() {
	fds := []unix.PollFd{
		{Fd: (int32)(l.closePipe[0]), Events: unix.POLLHUP},
		{Fd: (int32)(l.deviceFd), Events: unix.POLLIN},
	}

	for {
		_, err := unix.Poll(fds, -1)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			break
		}

		if fds[0].Revents != 0 {
			break
		}

		if fds[1].Revents != 0 {
			dev := C.udev_monitor_receive_device(l.monitor)
			if dev == nil {
				continue
			}

			action := C.udev_device_get_action(dev)
			if action != nil {
				if C.strcmp(action, cstrAdd) == 0 {
					l.handleArrive(dev)
				} else if C.strcmp(action, cstrRemove) == 0 {
					l.handleRemove(dev)
				}
			}

			C.udev_device_unref(dev)
		}
	}

	close(l.closeChan)
}

func (l *Listener) Enumerate() error {
	enumerator := C.udev_enumerate_new(l.udev)
	if enumerator == nil {
		return errors.New("failed to create udev enumerator")
	}
	defer C.udev_enumerate_unref(enumerator)

	res := C.udev_enumerate_add_match_subsystem(enumerator, l.condition.subsystem)
	if res < 0 {
		return errors.New("failed to add udev subsystem filter")
	}

	if l.condition.devtype != nil {
		res = C.udev_enumerate_add_match_property(
			enumerator,
			cstrDevtype,
			l.condition.devtype,
		)
		if res < 0 {
			return errors.New("failed to add udev devtype filter")
		}
	}

	if C.udev_enumerate_scan_devices(enumerator) < 0 {
		return errors.New("failed to perform udev enumeration")
	}

	entry := C.udev_enumerate_get_list_entry(enumerator)
	for ; entry != nil; entry = C.udev_list_entry_get_next(entry) {
		path := C.udev_list_entry_get_name(entry)
		if path == nil {
			continue
		}

		dev := C.udev_device_new_from_syspath(l.udev, path)
		if dev == nil {
			continue
		}

		l.handleArrive(dev)
		C.udev_device_unref(dev)
	}

	return nil
}

func (l *Listener) handleArrive(dev *C.struct_udev_device) {
	if !l.condition.matches(dev) {
		return
	}

	devnode := C.udev_device_get_devnode(dev)
	if devnode == nil {
		return
	}
	goDevnode := C.GoString(devnode)

	devpath := C.udev_device_get_devpath(dev)
	if devpath == nil {
		return
	}
	goDevpath := C.GoString(devpath)

	if l.condition.interfaceOnly {
		dev = C.udev_device_get_parent(dev)
		if dev == nil {
			return
		}
	}

	goDevIf := &DeviceInterface{
		Path:     goDevnode,
		Class:    l.class,
		Device:   newDevice(l, dev),
		listener: l,
		devpath:  goDevpath,
		inArrive: true,
	}

	l.callback(goDevIf)

	goDevIf.inArrive = false
}

func (l *Listener) handleRemove(dev *C.struct_udev_device) {
	devpath := C.udev_device_get_devpath(dev)
	if devpath == nil {
		return
	}
	goDevpath := C.GoString(devpath)

	for _, callback := range l.detachCb[goDevpath] {
		if callback != nil {
			callback()
		}
	}
	delete(l.detachCb, goDevpath)
}
