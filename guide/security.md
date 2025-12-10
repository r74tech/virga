---
url: /virga/guide/security.md
---
# Virga Security Model

This document outlines the security architecture and best practices for deploying and operating the Virgaramework. Understanding these concepts is crucial for maintaining operational security (OPSEC) and protecting both your infrastructure and your targets.

## Guiding Principles

* **Defense in Depth**: Security is implemented in layers, from network communication to implant behavior.
* **Configurability**: Security features are designed to be adaptable to different operational scenarios.
* **Transparency**: This document aims to be transparent about the current security features and their limitations.

## Communication Security

Securing the communication channel between the beacon and the C2 server is fundamental.

### Transport Layer Security

* **HTTPS Listeners**: It is strongly recommended to use `https` listeners for all production deployments. This encrypts traffic at the transport layer using TLS, protecting against eavesdropping and man-in-the-middle (MitM) attacks.
* **SSL/TLS Certificates**: Use valid, trusted SSL/TLS certificates. While self-signed certificates can work for testing, they are easily detectable. Services like Let's Encrypt provide free, trusted certificates.

### Application Layer Encryption

* **AES-256-GCM**: All beacon communication is encrypted at the application layer using AES-256-GCM. This provides an additional layer of security, ensuring that even if TLS is compromised, the C2 traffic remains confidential.
* **Encryption Key Management**:

  > **⚠️ CRITICAL WARNING:** The default configuration uses a static, publicly known encryption key (`change-this-key-in-production-environment`). You **MUST** generate a unique, strong key for each deployment. Using the default key makes your C2 traffic trivial to decrypt.

  **Generate a secure key using the provided script:**

  ```bash
  ./scripts/generate-secure-key.sh
  ```

  **Or generate manually:**

  ```bash
  # Generate a strong, random 32-byte key
  openssl rand -base64 32

  # Set as environment variable (recommended)
  export VIRGA_ENCRYPTION_KEY='your-generated-key-here'
  ```

  **Configuration example:**

  ```yaml
  listeners:
    - encryption:
        key: ${VIRGA_ENCRYPTION_KEY}  # Use environment variable
        type: aes-256
  ```

## Implant Security

The security of the beacon implant itself is vital to avoid detection.

### In-Memory Operations

* **MemDB**: The AI-powered features use an in-memory database (`go-memdb`) to store operational data. This minimizes the disk footprint on the target system, making forensic analysis more difficult.
* **Self-Deletion**: The `self_delete` option in the generator configuration, when enabled, attempts to remove the beacon binary from the disk after execution.

### Obfuscation and Evasion

* **Code Obfuscation (`obfuscation: true`)**: This feature is **planned** to automatically obfuscate the beacon's code, including string encryption and dynamic API resolution, to make reverse engineering more challenging.
* **Anti-AV (`anti_av: true`)**: This feature is **planned** to incorporate techniques to bypass or disable common antivirus products.
* **Anti-ETW (`anti_etw: true`)**: This feature is **planned** to include methods for patching Event Tracing for Windows (ETW) to reduce logging of malicious activities.

> **Note**: As of the current version, these advanced evasion features are not yet implemented. Rely on other OPSEC measures for stealth.

## Server Security

Protecting the C2 server is as important as securing the implants.

* **Admin API Access**: The admin port (default: `8443`) should be firewalled and accessible only from trusted IP addresses. Exposing it to the public internet is a significant risk.
* **Database Security**: The SQLite database (`virga.db`) contains sensitive information, including session data, target information, and task history. Ensure the file has appropriate permissions and is encrypted at rest if required by your environment.
* **Logging**: Monitor server logs (`logs/server.log`) for suspicious activity. In a production environment, consider shipping logs to a secure, centralized logging solution.

## Operational Security (OPSEC) Best Practices

* **Listener Profiles**: Customize your listener profiles to mimic legitimate web traffic. Use common `uri_path` values and consider using a redirector to hide the true location of your C2 server.
* **User-Agent**: Change the default `user_agent` in your generator settings to blend in with the target environment.
* **Jitter**: Always use a `jitter` value (e.g., 20-30%) to randomize beacon check-in intervals, making traffic analysis more difficult.
* **Legal and Ethical Use**: This tool is intended for authorized security testing and research only. Unauthorized use is illegal and unethical. Always adhere to the law and obtain explicit permission before using Virga
