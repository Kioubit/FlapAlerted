# FlapAlerted Pro

## Peers with your BGP daemon to receive BGP Update messages for accurate path change information

The program will listen on port 1790 for incoming BGP sessions
The following commandline paramters are required:

1. Number of times a route path needs to change
2. Interval within which a notification will be triggered if the number given in [1] is exceeded.
3. Your asn