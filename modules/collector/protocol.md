## Collector Protocol Documentation

This document outlines the TCP protocol for implementing the collector end of the communication.

### Connection & Initialization

1.  **Connection:** The monitor connects to the collector via TCP (**optionally with TLS**) using the configured `collectorEndpoint`.
2.  **Handshake:** Upon successful connection, the monitor sends two lines:
    *   `HELLO <InstanceName>`
    *   `VERSION <ProgramVersion>`

### Communication Flow

Communication is a **request/response** model where the collector sends a command, and the monitor responds with a single line.

*   Each command and response is terminated by a newline (`\n`).
*   Maximum command length is 1024 bytes.
*   The monitor enforces a **rate limit** of 15 commands per minute; exceeding this will cause the monitor to disconnect.
*   The monitor uses a **read deadline** (5 minutes) to detect connection stalls and automatically disconnect.

### Commands and Responses

| Command (Collector $\to$ Monitor) | Arguments | Monitor Response                         | Notes                                                                       |
|:----------------------------------|:----------|:-----------------------------------------|:----------------------------------------------------------------------------|
| **PING**                          | None      | `PONG`                                   | Keep the connection alive.                                                  |
| **ACTIVE\_FLAPS**                 | None      | JSON string of active flaps              | Returns a JSON string of active flaps.                                      |
| **AVERAGE\_ROUTE\_CHANGES\_90**   | None      | Floating point number (2 decimal places) | Returns the current 90th percentile average route change value.             |
| **INSTANCE**                      | None      | String                                   | The instance name supplied during initialization (`collectorInstanceName`). |
| **VERSION**                       | None      | String                                   | The program version.                                                        |
| *Unknown Command*                 | Any       | `ERROR: received unknown command`        | Error response for undefined commands.                                      |
| *Empty Command*                   | None      | `ERROR: empty command`                   | Error response for an empty input line.                                     |

### Error Handling (Collector Side)

The collector should handle the following scenarios:

1.  **Read/Write Errors:** Connection failure during communication.
2.  **Monitor Disconnection:** The monitor will disconnect if the rate limit is exceeded, or if configured timeouts are hit.
3.  **Monitor Errors:** The monitor responds with `ERROR: ...` for malformed, unknown commands or other errors.