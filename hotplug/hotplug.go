package hotplug

type InterfaceClass uint

const (
	DevIfUnknown InterfaceClass = iota
	DevIfHid
	DevIfPrinter
	DevIfStorage
	DevIfStoragePartition
)

type DeviceClass uint

const (
	DevUnknown DeviceClass = iota
	DevHid
	DevUsbDevice
	DevUsbInterface
)
