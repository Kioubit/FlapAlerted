# FlapAlerted

<h3>Receives BGP Update messages by peering with your BGP daemon. Detects path changes and BGP flapping events.</h3>

| Overview Page | Event details page |
| ------------- | ------------------ |
| ![a](https://github.com/user-attachments/assets/303d9aca-b4e3-4613-91ad-891ae16bf49d) | ![b](https://github.com/user-attachments/assets/860615e2-4116-429d-ab27-f8e5e70b69a0) |

### Setup notes

The program will listen on port 1790 for incoming BGP sessions (passive mode - no outgoing connections).
Peering multiple nodes with a single instance of the program is supported. It is recommended to adjust
the `routeChangeCounter` and `minimumAge` parameters (see usage) to produce the desired result.

### Usage
```
Usage:
  -asn int
        Your ASN number
  -asnPosition int
        The position of the last static ASN (and for which to keep separate state for) in each path. Use of this parameter is required for special cases such as when connected to a route collector. (default -1)
  -debug
        Enable debug mode (produces a lot of output)
  -disableAddPath
        Disable BGP AddPath support. (Setting must be replicated in BGP daemon)
  -minimumAge int
        Minimum age in seconds a prefix must be active to be listed. Has no effect if the routeChangeCounter is set to zero (default 540)
  -noPathInfo
        Disable keeping path information. (only disable if memory usage is a concern)
  -pathInfoActiveOnly
        Keep path information only for active prefixes (reduces memory usage)
  -period int
        Interval in seconds within which the routeChangeCounter value is evaluated. Higher values increase memory consumption. (default 60)
  -routeChangeCounter int
        Number of times a route path needs to change to list a prefix. Use '0' to show all route changes. (default 700)
  -routerID string
        BGP Router ID for this program (default "0.0.0.51")
```

### Example BIRD bgp daemon configuration
```
protocol bgp FLAPALERTED {
    local fdcf:8538:9ad5:1111::3 as 4242423914; # This address cannot be ::1, it must be another address assigned to the host
    neighbor ::1 as 4242423914 port 1790;

    ipv4 {
        add paths on;
        export all;
        import none;
    };

    ipv6 {
        add paths on;
        export all;
        import none;
    };
}
```

### Module Documentation
The program supports additional modules that can be customized at build-time.

#### mod_httpAPI (Enabled by default)
Provides the following http API endpoints on port `8699`:

- `/capabilities`
- `/flaps/active/compact`
- `/flaps/prefix?prefix=<cidr value>`
- `/flaps/active/history?cidr=<cidr value>`
- `/flaps/metrics/json`
- `/flaps/metrics/prometheus`

It also provides a user interface (on the same port) at path:
- `/`

To disable this module, add the following tag to the `MODULES` variable in the `Makefile`: `disable_mod_httpAPI`

#### mod_log (Enabled by default)
Logs each time a prefix exceeds the defined `routeChangeCounter` within the defined `period` to STDOUT.

To disable this module, add the following tag to the `MODULES` variable in the `Makefile`: `disable_mod_log`

***

### Building

#### Manually

You will need to have GO installed on your system. Then run `make release` and find the binary in the `bin/` directory.

#### Docker

Clone this repository and run `docker build .` to generate a docker image.
