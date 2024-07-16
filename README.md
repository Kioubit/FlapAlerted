# FlapAlerted

## Receives BGP Update messages by peering with your BGP daemon. Detects path changes and BGP flapping events.

<p align="center">
<img src="https://github.com/user-attachments/assets/67f83d31-0abc-48cf-a35e-efe33fc808b9" height="400">
</p>

### Setup notes

The program will listen on port 1790 for incoming BGP sessions (passive mode - no outgoing connections).
Peering multiple nodes with a single instance of the program is also possible.

### Usage
```
Usage:
  -asnPosition int
    	The position of the last static ASN (and for which to keep separate state for) in each path. If AddPath support has been enabled this value is '1', otherwise it is '0'. For special cases like route collectors the value may differ. (default -1)
  -asn int
    	Your ASN number
  -debug
    	Enable debug mode (produces a lot of output)
  -disableAddPath
    	Disable BGP AddPath support. (Setting must be replicated in BGP daemon)
  -noPathInfo
    	Disable keeping path information. (Only disable if performance is a concern)
  -period int
    	Interval in seconds within which the routeChangeCounter value is evaluated (default 60)
  -routeChangeCounter int
    	Number of times a route path needs to change to list a prefix

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


### Special mode: All BGP updates
Use the value `0` for the RouteChangeCounter [2] if all BGP updates should cause a notification from the program. 

### Building

#### Enabling or disabling modules

To enable or disable modules edit the `MODULES` variable in `Makefile` or the `Dockerfile` if you are using Docker.

#### Manual

You will need to have GO installed on your system. Then run `make release` and find the binary in the `bin/` directory.

#### Docker

Clone this repository and run `docker build .` to generate a docker image.


***

### Module Documentation

#### mod_httpAPI (Enabled by default)
Provides the following http API endpoints on port `8699`:

- `/capabilities`
- `/flaps/active`
- `/flaps/active/history?cidr=<cidr value>`
- `/flaps/active/compact`
- `/flaps/metrics/json`
- `/flaps/metrics/prometheus`

It also provides a user interface at path:
- `/`

#### mod_log (Disabled by default)
Logs each event to STDOUT.