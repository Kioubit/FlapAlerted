# FlapAlerted Pro

## Peers with your BGP daemon to receive BGP Update messages for accurate path change information


### Setup notes

The program will listen on port 1790 for incoming BGP sessions (passive mode - no outgoing connections).
Peering multiple nodes with a single FlapAlertedPro instance is also possible in most cases. The 'multihop' mode in your BGP daemon (i.e BIRD) should be enabled to allow for running this program on the same host. Note that with BIRD there is a bug that requires using another IP address that belongs to the host (for example an IP from a dymmy interface) for it to correctly send bgp information to a local FlapAlertedPro instance.


### Commandline arguments

The following commandline arguments are required:

1. Number of times a route path needs to change
2. Interval in seconds within which a notification will be triggered if the number given in [1] is exceeded.
3. Your asn
4. Recommended: 'true'. Whether to store all possible AS paths for a flap event. Only disable if performance is a concern. Values: 'true' or 'false'
5. Recommended: 'false'. Whether AddPath support should be enabled. In most cases option [6] must be enabled as well for this option to produce the intended output. Values: 'true' or 'false'
6. Recommended: 'false'. Whether separate state should be kept for each eBGP peer. Requires iBGP peering with FlapAlertedPro. (The first ASN in each path must be the eBGP peer) Values: 'true' or 'false'
7. Recommended: 'false'. Whether to notify only once for each flapping event. Values: 'true' or 'false'
8. Recommended: 'false'. Enable or disable debug output. Values: 'true' or 'false'

### Building

You will need to have GO installed on your system. Then run `make release` and find the binary in the `bin/` directory.

#### Enabling or disabling modules

To enable or disable modules edit the `MODULES` variable in `Makefile`.

***

### Module Documentation

#### mod_log
Simple logger to STDOUT for events.

#### mod_httpAPI
Provides the following http API endpoints on port `8699`:

- `/capabilities`
- `/flaps/active`
- `/flaps/metrics`
- `/flaps/metrics/prometheus`

#### mod_tcpNotify (Disabled by default)
Listens for tcp connections on port `8700` and sends a json object for every flap containing more information about the event.

#### core_doubleAddPath (Disabled by default)
Only for route collectors. Support double add path scenarios. (When peers also supply add path information)