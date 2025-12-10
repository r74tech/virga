# Beacon Config File Reference

This document provides a complete reference for all options available in the beacon configuration YAML file. You can use `configs/beacon.yaml` as a template.

## Root Level

- `beacon` (object, required): Contains all beacon-related configuration.
- `implant` (object, optional): Configures the implant's internal logging.
- `llama` (object, optional): Configures the AI features.
- `output` (object, required): Defines the output file path.
- `advanced` (object, optional): Advanced compilation options.

---

## `beacon`

| Key        | Type   | Description                               |
| ---------- | ------ | ----------------------------------------- |
| `c2`       | object | (Required) C2 server connection settings. |
| `behavior` | object | (Required) Beacon operational behavior.   |
| `target`   | object | (Required) Target platform definition.    |

### `beacon.c2`

| Key        | Type    | Default       | Description                                     |
| ---------- | ------- | ------------- | ----------------------------------------------- |
| `host`     | string  | `127.0.0.1`   | The IP address or hostname of the C2 server.    |
| `port`     | integer | `8080`        | The port number of the C2 listener.             |
| `protocol` | string  | `http`        | The communication protocol (`http` or `https`). |
| `uri_path` | string  | `api/updates` | The URI path for the beacon to check in to.     |

### `beacon.behavior`

| Key          | Type    | Default        | Description                                                  |
| ------------ | ------- | -------------- | ------------------------------------------------------------ |
| `sleep_time` | integer | `60`           | The base sleep time in seconds between check-ins.            |
| `jitter`     | integer | `20`           | The percentage of jitter to apply to the sleep time (0-100). |
| `user_agent` | string  | (Browser-like) | The User-Agent string to use for HTTP/S requests.            |

### `beacon.target`

| Key      | Type   | Default   | Description                                                 |
| -------- | ------ | --------- | ----------------------------------------------------------- |
| `os`     | string | `windows` | The target operating system (`windows`, `linux`, `darwin`). |
| `arch`   | string | `amd64`   | The target architecture (`amd64`, `arm64`).                 |
| `format` | string | `exe`     | The output file format (`exe`, `dll`, `elf`, etc.).         |

### `beacon.shell`

| Key                      | Type   | Default                         | Description                                                                  |
| ------------------------ | ------ | ------------------------------- | ---------------------------------------------------------------------------- |
| `powershell.amsi_bypass` | string | Built-in obfuscated AMSI bypass | Optional override injected before embedded PowerShell payloads are executed. |

### `beacon.payloads`

Bundled payloads let you ship scripts/tools with the beacon binary so you can execute them later using the `shell` command without touching the network.

| Key      | Type   | Description                                                    |
| -------- | ------ | -------------------------------------------------------------- |
| `name`   | string | Operator-facing identifier used with `shell @name ...`.        |
| `source` | string | Local path or HTTPS URL to pull into the beacon at build time. |
| `type`   | string | Execution runtime (currently only `powershell` is supported).  |

> **Usage tip:** Once a payload named `powerview` is defined you can execute it via `shell> @powerview Get-NetUser` and virga will stream the embedded script plus your command into PowerShell with the AMSI bypass applied automatically.

---

## `implant`

| Key             | Type    | Default       | Description                                           |
| --------------- | ------- | ------------- | ----------------------------------------------------- |
| `log_enabled`   | boolean | `false`       | Enables detailed logging within the beacon itself.    |
| `log_file_path` | string  | `implant.log` | The path to save the log file to on the target.       |
| `log_level`     | string  | `info`        | The logging level (`debug`, `info`, `warn`, `error`). |

---

## `llama`

| Key            | Type    | Description                                        |
| -------------- | ------- | -------------------------------------------------- |
| `enabled`      | boolean | (Required) Enables or disables all AI features.    |
| `log_enabled`  | boolean | Enables detailed logging for the LLM engine.       |
| `model`        | object  | Configures the AI model's inference parameters.    |
| `prompt`       | object  | Defines the initial instruction set for the model. |
| `autonomous`   | object  | Configures the autonomous operation loop.          |
| `task_prompts` | object  | A map of custom prompts for specific tasks.        |

### `llama.model`

| Key           | Type    | Default | Description                                         |
| ------------- | ------- | ------- | --------------------------------------------------- |
| `context`     | integer | `8192`  | The maximum token context size for the model.       |
| `gpu_layers`  | integer | `0`     | Number of model layers to offload to the GPU.       |
| `threads`     | integer | `4`     | Number of CPU threads to use for inference.         |
| `temperature` | float   | `0.7`   | Controls the creativity of the model (0.0-1.0).     |
| `top_k`       | integer | `40`    | Vocabulary sampling parameter.                      |
| `top_p`       | float   | `0.95`  | Nucleus sampling parameter.                         |
| `max_tokens`  | integer | `2048`  | Maximum number of tokens to generate in a response. |

### `llama.prompt`

| Key      | Type   | Default    | Description                                                                        |
| -------- | ------ | ---------- | ---------------------------------------------------------------------------------- |
| `preset` | string | `enhanced` | The initial system prompt preset (`default`, `enhanced`, `stealth`, `aggressive`). |

### `llama.autonomous`

| Key               | Type    | Description                                                      |
| ----------------- | ------- | ---------------------------------------------------------------- |
| `enabled`         | boolean | Enables the autonomous execution loop.                           |
| `initial_tasks`   | array   | A list of tasks for the AI to perform upon startup.              |
| `max_iterations`  | integer | The maximum number of command-execution loops for a single task. |
| `timeout_minutes` | integer | The timeout in minutes for the entire autonomous operation.      |
| `report_interval` | integer | The interval in seconds for reporting progress.                  |

---

## `output`

| Key    | Type   | Default           | Description                                 |
| ------ | ------ | ----------------- | ------------------------------------------- |
| `path` | string | `dist/beacon.exe` | The path to save the generated beacon file. |

---

## `advanced`

| Key             | Type    | Default | Description                                                         |
| --------------- | ------- | ------- | ------------------------------------------------------------------- |
| `strip_symbols` | boolean | `true`  | Strips debugging symbols from the compiled binary.                  |
| `compress`      | boolean | `false` | Compresses the final binary (e.g., with UPX). (Not yet implemented) |
| `anti_debug`    | boolean | `false` | Includes anti-debugging techniques. (Not yet implemented)           |
