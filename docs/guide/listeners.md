# Listeners

Listeners are the components of the Virga server that wait for incoming connections from beacons. You can configure multiple listeners of different types to handle various communication channels.

## Key Concepts

- **Binding**: Each listener is bound to a specific IP address and port on the server.
- **Protocol**: Listeners can be configured to use different protocols, primarily HTTP and HTTPS.
- **Encryption**: All communication is encrypted at the application layer, providing an additional layer of security on top of any transport-level encryption (like TLS).

## Basic Configuration

Listeners are defined in the `listeners` array in your `config.yaml` file. Here is a basic example of an HTTP listener:

```yaml
listeners:
  - name: "default-http"
    type: "http"
    bind_address: "0.0.0.0"
    port: 8080
    uri_path: "api/updates"
    encryption:
      type: "aes-256"
      key: "your-super-secret-key"
```

### Core Fields

- `name`: A unique name for the listener.
- `type`: The protocol type. Currently, `http` and `https` are supported.
- `bind_address`: The IP address to listen on. `0.0.0.0` listens on all available network interfaces.
- `port`: The port to listen on.
- `uri_path`: The specific URL path that the beacon will connect to (e.g., `http://c2.example.com/api/updates`).

### Encryption

- `encryption.type`: The encryption algorithm. Only `aes-256` is currently supported.
- `encryption.key`: The secret key for encrypting traffic. 
    > **⚠️ CRITICAL WARNING:** You **MUST** use a unique, strong key for each deployment. See the [Security Guide](/guide/security) for more information.

### HTTPS Configuration

To use HTTPS, you must set `type: "https"` and provide paths to your SSL/TLS certificate and key files:

```yaml
listeners:
  - name: "primary-https"
    type: "https"
    bind_address: "0.0.0.0"
    port: 443
    uri_path: "api/v2/updates"
    ssl:
      cert: "/path/to/fullchain.pem"
      key: "/path/to/privkey.pem"
    encryption:
      # ... (encryption settings)
```

For a complete list of all configuration options, see the [Server Configuration Reference](/reference/server-config-reference).
