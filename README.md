# FlapAlerted

## Receives BGP Update messages by peering with your BGP daemon. Detects path changes and BGP flapping events.

### Setup notes

The program will listen on port 1790 for incoming BGP sessions (passive mode - no outgoing connections).
Peering multiple nodes with a single instance of the program is also possible.

### Commandline arguments
The following commandline arguments are required:


    Usage: ./FlapAlerted <RouteChangeCounter> <FlapPeriod> <Asn> <KeepPathInfo> <UseAddPath> <RelevantAsnPosition> <Debug>

Explanation:

1. Number of times a route path needs to change 
2. Interval in seconds within which a notification will be triggered if the number given in [1] is exceeded.
3. Your ASN
4. Recommended: `true`. Whether to store all possible AS paths for each BGP update. Only disable if performance is a concern.
5. Recommended: `false`. Whether BGP AddPath support should be used. It must be set on the BGP daemon as well.
6. Recommended: `auto`. The position of the last static ASN (and for which to keep separate state for) in each path starting from 1. If AddPath support has been enabled that value is '1', otherwise it is '0'. For special cases like route collectors the value may differ.
7. Recommended: `false`. Enable or disable debug output. This option produces a lot of output.

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

#### mod_tcpNotify (Disabled by default)
Listens for tcp connections on port `8700`. Sends a json string to each client for every event.