# USB Display

A small Go CLI for the WeAct Display FS 0.96 inch (80x160, USB CDC).
Displays OS release information and IPv4 addresses in landscape orientation.
Linux and macOS are the implementation targets. Windows is not supported yet.

## Build and Run

Requires Go 1.25 or newer. No Python, system fonts, cgo, or USB library installation is needed.

```sh
go build -o usb-display ./cmd/usb-display
./usb-display
```

When exactly one USB CDC candidate exists, the serial port is selected automatically.
Auto mode only considers `/dev/cu.usbmodem*` on macOS and `/dev/ttyACM*` on Linux;
it does not verify USB identity. Use `--port` when other CDC devices are attached or
when using a persistent Linux `/dev/serial/by-id/...` path.

```sh
./usb-display --port /dev/cu.usbmodemaddef39a51ab1
./usb-display --port /dev/ttyACM0 --interval 5s
./usb-display --test
./usb-display --version
```

`--test` displays text and RGB bars for five seconds, then switches the backlight off.
Stop normal operation with Ctrl-C. The backlight is switched off and the serial port
is released on exit. Disconnecting the display causes an error; reconnect it and
restart the program. Other software must not own the serial port.

On Linux, grant your user access to the serial device according to your distribution
(commonly membership in `dialout` or `uucp`, followed by logging in again).
Do not run the entire application as root to work around a permission error.

## Configuration

Optional default configuration locations:

- Linux: `$XDG_CONFIG_HOME/usb-display/config.toml`, or `~/.config/usb-display/config.toml`.
- macOS: `~/Library/Application Support/usb-display/config.toml`.

An explicitly supplied `--config` must exist. Unknown TOML keys and invalid values
are errors. Configuration is read once at startup, not watched.

```toml
interval = "3s"
port = "auto"
brightness = 30
```

```sh
./usb-display --config config.example.toml --interval 1s --brightness 50
```

Priority: built-in defaults, then TOML, then explicitly supplied flags.
An explicit `--interval=3s` overrides a different value in TOML. The interval must
be positive; brightness is an integer from 0 to 100. `port` must not be empty.
Environment variable overrides are not supported.

## Displayed Data

The ordered cycle starts with the OS, then visits every non-loopback interface,
including virtual, administratively down, and unaddressed interfaces. IPv6 is not
shown. IPv4 link-local addresses (`169.254.x.x`) are retained. Multiple IPv4
addresses receive separate pages. Interfaces without IPv4 show `NO IPv4`.
An address collection failure shows `DATA ERROR`, not `NO IPv4`.

`UP`/`DOWN` is the administrative state, not proof of a connection. `LINK YES` or
`LINK NO` reports carrier for Ethernet and connection state for Wi-Fi. `LINK ?`
means unavailable or inapplicable, including virtual interfaces. When an interface
is down, the OS may be unable to report whether a cable is connected.

Linux reads carrier from sysfs only for interfaces backed by a device. macOS uses
`networksetup -listallhardwareports` to identify hardware and `ifconfig -a` to read
active/inactive status; both run within a two-second snapshot budget. Missing
status is unknown. No network configuration is changed.

Network data is refreshed before each new page. The page interval starts after
the bitmap has been written to the serial transport; OS data is read at startup.
Long names continue onto additional pages. A large number of virtual interfaces
therefore makes a full cycle longer.

Linux uses `/etc/os-release` with `/usr/lib/os-release` as fallback, preferring
`PRETTY_NAME`. macOS uses `sw_vers`. OS release files are never executed as shell
scripts. Build metadata from `--version` is separate from the displayed OS release.

## Validation

```sh
go test ./...
go test -race ./...
go vet ./...
GOOS=linux GOARCH=amd64 go build -o /tmp/usb-display-linux-amd64 ./cmd/usb-display
GOOS=linux GOARCH=arm64 go build -o /tmp/usb-display-linux-arm64 ./cmd/usb-display
```

The connected macOS device identified itself as `Display FS 0_96 Inch`,
USB `1a86:fe0c`, firmware `V1.0.0.2`. Serial communication and a diagnostic frame
were exercised with RTS/CTS enabled. Visual output still requires inspection on
the physical screen. Cross-compilation does not establish Linux hardware support;
carrier changes, USB disconnect, and permissions must also be checked on Linux.

Protocol references: [WeAct Studio System Monitor](https://github.com/WeActStudio/WeActStudio.SystemMonitor)
and [Turing Smart Screen Python](https://github.com/mathoudebine/turing-smart-screen-python).
This implementation uses uncompressed RGB565 little-endian frames. Python source,
themes, and artwork are not bundled. The Go Mono font is supplied by `golang.org/x/image`.
