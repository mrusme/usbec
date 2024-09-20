usbec
-----

The *USB Equipment Commander* is a lightweight daemon that is able to run 
commands based on the USB equipment connected to a computer. It makes it easily 
possible to run scripts and programs when specific USB devices are being 
connected or disconnected.

*"Why, there's udev for that?!"* you might say. Sure, but udev rules come with 
all sorts of limitations, making it very cumbersome to run especially 
Wayland-related commands. Often times, rules that trigger user-specific scripts 
and tools are hacky at best, given the constrains imposed by udev.

The daemon can be fully configured using a [toml file](usbec.toml).


## Build

```sh
go build .
```


## Configure

Check out the [example config](usbec.toml) and create a similar config under any 
of the usual configuration paths (`/etc/usbec.toml`, 
`$XDG_CONFIG_HOME/usbec.toml`, `./usbec.toml`).

## Run

```sh
usbec
```


## What can I do with `usbec`?

Examples include disabling the internal keyboard when you connect an external 
keyboard to your laptop, triggering a script that runs a backup on an external 
hard drive as soon as it's being connected, opening a LUKS encrypted device when 
a USB stick containing a key was inserted and much more.
