# FlapAlerted

<h3>Receives BGP Update messages by peering with your BGP daemon. Detects path changes and BGP flapping events.</h3>

| Overview Page                                                                         | Event details page                                                                    |
|---------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------|
| ![a](https://github.com/user-attachments/assets/303d9aca-b4e3-4613-91ad-891ae16bf49d) | ![b](https://github.com/user-attachments/assets/860615e2-4116-429d-ab27-f8e5e70b69a0) |

### Setup notes

The program will listen on port `1790` for incoming BGP sessions (passive mode - no outgoing connections).
It is recommended to adjust the `routeChangeCounter`, `expiryRouteChangeCounter`, `overThresholdTarget` and `underThresholdTarget` parameters (see usage) to produce the desired result.

### Basic Usage
```
-asn uint
    Your ASN number
-bgpListenAddress string
    Address to listen on for incoming BGP connections (default ":1790")
-debug
    Enable debug mode (produces a lot of output)
-disableAddPath
    Disable BGP AddPath support. (Setting must be replicated in BGP daemon)
-expiryRouteChangeCounter uint
    Minimum change per minute threshold to keep detected flaps. Defaults to the same value as 'routeChangeCounter'.
-importLimitThousands uint
    Maximum number of allowed routes per session in thousands (default 10000)
-noPathInfo
    Disable keeping path information
-overThresholdTarget uint
    Number of consecutive minutes with route change rate at or above the 'routeChangeCounter' to trigger an event (default 10)
-routeChangeCounter uint
    Minimum change per minute threshold to detect a flap. Use '0' to show all route changes. (default 600)
-routerID string
    BGP router ID for this program (default "0.0.0.51")
-underThresholdTarget uint
    Number of consecutive minutes with route change rate below 'expiryRouteChangeCounter' to remove an event (default 15)
```
#### Using environment variables
Environment variables can configure options by prefixing `FA_` to any command-line flag name (optionally in uppercase). For example, set the ASN number with `FA_ASN=<asn>` or the router ID using `FA_routerID=<router id>`.
### Example BIRD bgp daemon configuration
```
protocol bgp flapalerted {
    local fdcf:8538:9ad5:1111::3 as 4242423914; # This address cannot be ::1, it must be another address assigned to the host
    neighbor ::1 as 4242423914 port 1790;

    ipv4 {
        add paths on;
        export all;
        import none;
        extended next hop on;
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
- `/sessions`
- `/flaps/active/compact`
- `/flaps/active/roa`
- `/flaps/prefix?prefix=<cidr value>`
- `/flaps/active/history?cidr=<cidr value>`
- `/flaps/metrics/json`
- `/flaps/metrics/prometheus`
- `/flaps/avgRouteChanges90`

It also provides a user interface (on the same port) at path:
- `/`

Configuration:
```
-apiKey string
    API key to access limited endpoints, when 'limitedHttpApi' is set. Empty to disable
-httpAPIListenAddress string
    Listen address for the HTTP API (TCP address like :8699 or Unix socket path) (default ":8699")
-httpGageMaxValue uint
    HTTP dashboard Gage max value (default 400)
-limitedHttpApi
    Disable http API endpoints not needed for the user interface and activate basic scraping protection
-maxUserDefined uint
    Maximum number of user-defined tracked prefixes. Use zero to disable (default 5)
```
To disable this module, add the following tag to the `MODULES` variable in the `Makefile`: `disable_mod_httpAPI`

#### mod_log (Enabled by default)
Logs each detected active prefix to `STDOUT`.

To disable this module, add the following tag to the `MODULES` variable in the `Makefile`: `disable_mod_log`

#### mod_script (Enabled by default, except for docker builds)
Allows executing custom scripts when BGP flap events are detected. Scripts can be triggered at both the start and end of flap events.

Configuration:
- `-detectionScriptStart`: Path to script executed when a flap event starts
- `-detectionScriptEnd`: Path to script executed when a flap event ends

The scripts receive flap event data as a JSON string via command line argument.

To disable this module, add the following tag to the `MODULES` variable in the `Makefile`: `disable_mod_script`

#### mod_webhook (Enabled by default in docker builds)

Sends HTTP POST requests to specified URLs when BGP flap events are detected (at start and end).

Configuration:
- `-webhookUrlStart`: URL for when a flap event starts; can be specified multiple times
- `-webhookUrlEnd`: URL for when a flap event ends; can be specified multiple times
- `-webhookTimeout`: Timeout for HTTP requests
- `-webhookInstanceName`: Optional instance name to send as a header

Payload: Flap event data is sent as a JSON string in the request body.

To disable this module, add the following tag to the `MODULES` variable in the Makefile: `disable_mod_webhook`

#### mod_collector (Enabled by default)

Connects to a FlapAlerted collector via TCP and allows it to retrieve event information.

Configuration:
- `-collectorInstanceName`: Instance name to send to the collector
- `-collectorEndpoint`: TCP endpoint of the collector

To disable this module, add the following tag to the `MODULES` variable in the Makefile: `disable_mod_collector`

#### mod_roaFilter (Disabled by default)
Filters a ROA file in JSON format to remove flapping prefixes.
The filtered prefixes are to be re-added by the external program updating the ROA file at regular intervals.
See the command line help for required arguments.

To enable this module, add the following tag to the `MODULES` variable in the `Makefile`: `mod_roaFilter`
***

### Building

#### Manually

Install Go, then build using `make release`. The binary will be placed in the `bin/` directory.

#### Docker

Clone this repository and run `docker build .` to generate a docker image.
