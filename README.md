usbec
-----

The *USB Equipment Commander* is a lightweight daemon that is able to run 
commands based on the USB equipment connected to a computer. *"Why, there's udev 
for that?!"* you might say. Sure, but udev rules come with all sorts of 
limitations, making it very cumbersome to run especially Wayland-related 
commands.

**INFO:** So far this daemon is very much hardcoded for my own use cases and I 
haven't found the time nor motivation yet to make it more configurable. Unless 
you own the exact same hardware, this daemon will be of little use for you. 
However, feel free to PR any changes that would make it usable to your use case, 
if you feel like.

