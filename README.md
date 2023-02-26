# FlapAlerted Pro

## Receives BGP Update messages by peering with your BGP daemon. Accurately detects path changes and BGP flapping events.

### Setup notes

The program will listen on port 1790 for incoming BGP sessions (passive mode - no outgoing connections).
Peering multiple nodes with a single FlapAlertedPro instance is also possible. The 'multihop' mode in your BGP daemon (for example BIRD) should be enabled to allow for running this program on the same host. Note that with BIRD there is a bug that requires using another IP address that belongs to the host (for example an IP from a dummy interface) for it to correctly send bgp information to a local FlapAlertedPro instance.


### Commandline arguments
The following commandline arguments are required:


    Usage: <RouteChangeCounter> <FlapPeriod> <Asn> <KeepPathInfo> <UseAddPath> <RelevantAsnPosition> <NotifyOnce> <Debug>

Explanation:

1. Number of times a route path needs to change
2. Interval in seconds within which a notification will be triggered if the number given in [1] is exceeded.
3. Your ASN
4. Recommended: 'true'. Whether to store all possible AS paths for each BGP update. Only disable if performance is a concern. Values: 'true' or 'false'
5. Recommended: 'false'. Whether BGP AddPath support should be enabled. In most cases option [6] must be set to '1' for this option to produce the intended output. Values: 'true' or 'false'
6. The position of the last static ASN (and for which to keep separate state for) in each path starting from 1. If AddPath support has been enabled that value is 1, otherwise it is 0. For special cases like route collectors the value may differ.
7. Recommended: 'false'. Whether to notify only once for each flapping event. Setting this to 'true' may break the functionality of some modules. Values: 'true' or 'false'
8. Recommended: 'false'. Enable or disable debug output. This option produces a lot of output. Values: 'true' or 'false'

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

Below modules can be enabled or disabled in the Makefile

#### mod_httpAPI
Provides the following http API endpoints on port `8699`:

- `/capabilities`
- `/flaps/active`
- `/flaps/active/compact`
- `/flaps/metrics`
- `/flaps/metrics/prometheus`

It also provides a user interface:
- `/`

#### mod_log (Disabled by default)
Simple logger to STDOUT for events.

#### mod_tcpNotify (Disabled by default)
Listens for tcp connections on port `8700` and sends a json object for every flap containing more information about the event.