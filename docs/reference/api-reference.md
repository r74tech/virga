# Virga API Reference

This document provides a reference for the Virgaerver's API endpoints. The API is divided into two main categories:

1.  **Beacon API**: Endpoints used by the beacon implants to communicate with the server.
2.  **Admin API**: RESTful endpoints used by clients like the official CLI to manage the server, sessions, and listeners.

## 1. Beacon API

This is the primary endpoint for beacon check-ins. The exact path is configurable.

### `POST /{uri_path}`

This single endpoint handles all incoming communication from a beacon, including initial registration, task result submission, and requests for new tasks.

- **Path**: The path is defined by the `uri_path` parameter in the listener's configuration (e.g., `/api/updates`).
- **Method**: `POST`
- **Content-Type**: `text/plain` (The body is a Base64-encoded, AES-encrypted JSON payload).

#### Request Body (Decrypted JSON)

The decrypted JSON payload sent by the beacon has the following structure, which corresponds to the `protocol.AgentMessage` struct in the source code.

```json
{
  "agent_id": "unique-beacon-identifier",
  "hostname": "TARGET-PC",
  "username": "target-user",
  "os": "windows",
  "version": "1.0.0",
  "ip": "192.168.1.10",
  "boot_time": 1672531200,
  "environment": {
    "arch": "amd64",
    "process_id": 1234,
    "working_dir": "C:\\Users\\target-user"
  },
  "task_results": [
    {
      "task_id": "task-uuid-123",
      "output": "...command output...",
      "exit_code": 0,
      "error": "",
      "timestamp": 1672531260
    }
  ]
}
```

#### Response Body (Decrypted JSON)

The server responds with a JSON object containing new tasks for the beacon, corresponding to the `protocol.AgentResponse` struct.

<!-- TODO: The Task object in the response has both `id` and `task_id`, and `payload` and `arguments`. We should standardize on using `id` and `arguments` and deprecate the others to clean up the API. Consider implementing a tool to automatically generate API documentation from the Go source code to prevent drift. -->

```json
{
  "status": "OK",
  "tasks": [
    {
      "id": "new-task-uuid-456",
      "task_id": "new-task-uuid-456",
      "type": "shell",
      "session_id": "unique-beacon-identifier",
      "arguments": {
        "command": "whoami /all"
      },
      "created_at": "2023-01-01T00:01:05Z",
      "sent_time": "2023-01-01T00:01:05Z",
      "status": "pending"
    }
  ],
  "interactive": false,
  "reconnect_time": 30
}
```
- **interactive**: A boolean flag that tells the beacon to switch to a faster check-in interval.
- **reconnect_time**: The number of seconds the beacon should wait before its next check-in.


## 2. Admin API

These endpoints are served on the `admin_port` (default: 8443) and are used to manage the C2 server.

### Session Management

#### List Sessions

Retrieves a list of all active sessions.

- **Endpoint**: `GET /api/sessions`
- **Success Response**: `200 OK`

```json
{
  "success": true,
  "sessions": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "remote_addr": "192.168.1.10:12345",
      "hostname": "TEST-PC",
      "username": "test-user",
      "os": "windows",
      "last_activity": "2023-01-01T12:00:00Z",
      "environment": {
        "arch": "amd64",
        "process_id": 1234
      }
    }
  ]
}
```

#### Add a Task to a Session

Queues a new task for a specific session.

- **Endpoint**: `POST /api/sessions/{id}/tasks`
- **URL Parameters**: `id` (string, required) - The ID of the session.
- **Request Body**:
  ```json
  {
    "type": "shell",
    "payload": {
      "command": "net user"
    }
  }
  ```
- **Success Response**: `200 OK`
  ```json
  {
    "success": true,
    "task_id": "new-task-uuid-789",
    "message": "Task added to queue"
  }
  ```

#### Get All Task Results for a Session

Retrieves all completed task results for a session.

- **Endpoint**: `GET /api/sessions/{id}/tasks`
- **Success Response**: `200 OK`

#### Get a Specific Task Result

Retrieves the result for a single task.

- **Endpoint**: `GET /api/sessions/{id}/tasks/{taskID}`
- **Success Response**: `200 OK`

#### Set Interactive Mode

Switches a session to interactive mode (faster check-ins).

- **Endpoint**: `POST /api/sessions/{id}/interactive`
- **Request Body**:
  ```json
  {
    "interactive": true
  }
  ```
- **Success Response**: `200 OK`

### Listener Management

#### List Listeners

Retrieves a list of all configured listeners.

- **Endpoint**: `GET /api/listeners`
- **Success Response**: `200 OK`

```json
{
    "success": true,
    "listeners": [
        {
            "name": "default-http",
            "type": "http",
            "bind_address": "0.0.0.0",
            "port": 8080,
            "status": "running"
        }
    ]
}
```

> **Note**: Creating and deleting listeners via the API is not yet implemented.

```