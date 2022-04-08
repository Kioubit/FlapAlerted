# FlapAlerted Pro

## Peers with your BGP daemon to receive BGP Update messages for accurate path change information


### Setup notes

The program will listen on port 1790 for incoming BGP sessions (passive mode - no outgoing connections).
Peering multiple nodes with a single FlapAlertedPro instance is also possible in most cases. The 'multihop' mode in your BGP daemon (i.e bird) should be enabled to allow for running this program on the same host.


### Commandline arguments

The following commandline arguments are required:

1. Number of times a route path needs to change
2. Interval in seconds within which a notification will be triggered if the number given in [1] is exceeded.
3. Your asn
4. Recommended: 'false'. Whether AddPath support should be enabled. In most cases option [5] must be enabled as well for this option to produce the intended output. Values: 'true' or 'false'
5. Recommended: 'false'. Whether separate state should be kept for each eBGP peer. Requires iBGP peering with FlapAlertedPro. (The first ASN in each path must be the eBGP peer) Values: 'true' or 'false'
6. Recommended: 'false'. Whether to notify only once for each flapping event. Values: 'true' or 'false'
7. Recommended: 'false'. Enable or disable debug output. Values: 'true' or 'false'

### Building

You will need to have GO installed on your system. Then run `make release` and find the binary in the `bin/` directory.

#### Enabling or disabling modules

To enable or disable modules edit the `MODULES` variable in `Makefile`.

***

### Module Documentation

#### mod_log
Simple logger to STDOUT for events.

#### mod_jsonapi
Provides the following JSON http endpoints on port `8699`:

- `/flaps/active`
- `/flaps/metrics`
- `/flaps/metrics/prometheus`