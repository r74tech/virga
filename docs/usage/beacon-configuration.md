# Beacon Configuration

While you can generate beacons using command-line flags, the recommended way to create a customized beacon is through a YAML configuration file. This approach provides fine-grained control over every aspect of the beacon, from its communication behavior to its AI capabilities.

To use a configuration file, pass the `--config` flag to the `generate beacon` command:

```bash
virga> generate beacon --config /path/to/your/beacon.yaml
```

You can use the `configs/beacon.yaml` file in the root of the project as a starting template.

## Core Configuration Blocks

A beacon configuration file is structured into several main blocks:

### `beacon`

This block defines the beacon's identity and how it communicates with the C2 server.

```yaml
beacon:
  c2:
    host: "127.0.0.1"
    port: 8080
    protocol: "http"
  behavior:
    sleep_time: 10
    jitter: 20
  target:
    os: "windows"
    arch: "amd64"
```

- **`c2`**: Specifies the C2 server's connection details.
- **`behavior`**: Controls the beacon's sleep and jitter cycle.
- **`target`**: Defines the target operating system and architecture.

### `llama`

This block configures the embedded AI features. You can enable or disable the AI, tune the model's parameters, and define autonomous tasks.

```yaml
llama:
  enabled: true
  model:
    temperature: 0.7
  autonomous:
    enabled: true
    initial_tasks:
      - type: "system_reconnaissance"
        description: "Complete system analysis and environment mapping"
```

### `output`

This block specifies where to save the generated beacon file.

```yaml
output:
  path: "dist/beacon.exe"
```

## Overriding with Flags

Note that any command-line flags you provide will override the values in the configuration file. For example, the following command will generate a beacon for Linux, even if the config file specifies Windows:

```bash
virga> generate beacon --config beacon.yaml --os linux
```

For a complete list of all available options, please refer to the [Beacon Config Reference](../reference/beacon-config-reference.md).
