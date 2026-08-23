## hotplug

This package is derived from [go-hotplug][gh] by Sam Hanes, vendored at commit
`77bd9a5abc65` (2023-10-20). It's licensed under the Apache License 2.0, a copy
of which is in [LICENSE.txt](LICENSE.txt). That license applies to this
directory only, not to the rest of `usbec`.

Upstream has seen no releases since October 2023.

[gh]: https://github.com/elemecca/go-hotplug

## Classes

`New` takes one interface class and reports devices matching it. `DevIfHid`
covers `hidraw` nodes, `DevIfPrinter` covers `usblp` nodes, and `DevIfStorage`
and `DevIfStoragePartition` cover USB attached block devices. Passing any other
value, `DevIfUnknown` included, returns an error.

A listener handles a single class, watching several of them at once means
creating several listeners. `usbec` uses `DevIfHid` and `DevIfStorage`.

## Changes

The following changes were made to the upstream source, as required by section
4(b) of the Apache License 2.0:

Two interface classes were added, `DevIfStorage` for USB attached whole disks
and `DevIfStoragePartition` for partitions. Both match the `block` subsystem, on
the `disk` and `partition` device types respectively.

Matching a subsystem alone isn't enough for block devices, because NVMe, SATA,
virtio and zram disks all appear under `block` too. So `deviceCondition` needs
`ancestorSubsystem` and `ancestorDevtype` fields, tested with
`udev_device_get_parent_with_subsystem_devtype`, and the two storage classes use
them to require a `usb`/`usb_device` ancestor. The check runs before the
`interfaceOnly` reparenting, so it applies to the node the event is about.

The Windows implementation was removed, along with the portable indirection that
existed to support it. Upstream split every type across a portable file and a
platform-specific one, so `platformListener`, `platformDevice` and
`platformDeviceInterface` are merged into `Listener`, `Device` and
`DeviceInterface` respectively.

Fixed strings passed to libudev are allocated once into package-level variables
instead of by `C.CString` on each call, which leaked memory on every event and
every attribute read.

The close-on-exec test in `Listen` was inverted upstream, reading
`flags&unix.FD_CLOEXEC != 0`, and so only set the flag when it was already set.
The monitor file descriptor is now made close-on-exec as intended.

`Enumerate` looped forever on a list entry with no name, because the `continue`
skipped the statement advancing the iterator.

The poll loop asserted `err.(syscall.Errno)`, which panics on any error that
isn't an `Errno`. It uses `errors.Is` now.

`OnDetach` allocated its callback slice with `make([]func(), 1)`, putting a nil
at the head of every slice.

`Stop` now closes the read end of the close pipe, which upstream leaked.

`runtime.SetFinalizer` was replaced with `runtime.AddCleanup`, and the unused
`Device.path` method was dropped.
