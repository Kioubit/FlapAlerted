## Collector Protocol Documentation

This document outlines the TCP protocol for implementing the collector end of the communication.

### Connection & Initialization

1.  **Connection:** FlapAlerted (the program) connects to the collector via TCP (**optionally with TLS**) using the configured `collectorEndpoint`.
2.  **Handshake:** Upon successful connection, the program sends two lines:
    *   `HELLO <InstanceName>`
    *   `VERSION <ProgramVersion>`

### Communication Flow

Communication is a **request/response** model where the collector sends a command, and the program responds with a single line.

*   Each command and response is terminated by a newline (`\n`).
*   Maximum command length is 1024 bytes.
*   The program enforces a **rate limit** of 15 commands per minute; exceeding this will cause the program to disconnect.
*   The program uses a **read deadline** (5 minutes) to detect connection stalls and automatically disconnect.

### Commands and Responses

| Command (Collector $\to$ FlapAlerted) | Arguments                                   | FlapAlerted Response                     | Notes                                                                                                              |
|:--------------------------------------|:--------------------------------------------|:-----------------------------------------|:-------------------------------------------------------------------------------------------------------------------|
| **PING**                              | None                                        | `PONG`                                   | Keep the connection alive if no other commands are received.                                                       |
| **ACTIVE\_FLAPS**                     | None                                        | JSON string of active flaps              | Returns a JSON string of active flaps.                                                                             |
| **AVERAGE\_ROUTE\_CHANGES\_90**       | None                                        | Floating point number (2 decimal places) | Returns the current 90th percentile average route change value.                                                    |
| **CAPABILITIES**                      | None                                        | JSON string of capabilities              | Returns the settings of the program.                                                                               |
| **NOTIFY_ERROR**                      | Reconnect (Boolean), Error message (String) | `OK`                                     | Notify the user of an error condition. The boolean dictates if the program should permanently disconnect (`true`). |
| **INSTANCE**                          | None                                        | String                                   | The instance name supplied during initialization (`collectorInstanceName`).                                        |
| **VERSION**                           | None                                        | String                                   | The program version.                                                                                               |

### Error Handling (Collector Side)

The collector should handle the following scenarios:

1.  **Read/Write Errors:** Connection failure during communication.
2.  **Program Disconnection:** The program will disconnect if the rate limit is exceeded, or if configured timeouts are hit.
3.  **Program Errors:** The program responds with `ERROR:...` for malformed, unknown commands or other errors.