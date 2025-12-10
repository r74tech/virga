# Introduction

Welcome to Virga! This document provides a high-level overview of the framework and its capabilities.

## What is Virga

Virga lightweight, post-exploitation C2 (Command and Control) framework designed for security professionals. It is built with Go for high performance and cross-platform compatibility, supporting Windows, Linux, and macOS.

A key feature of Virgats integrated AI, powered by an embedded LLM. This allows for autonomous operations, such as reconnaissance and data analysis, directly on the target system.

## Architecture Overview

```mermaid
flowchart TB
    subgraph Operator["🖥️ Operator Workstation"]
        CLI["CLI Client<br/>(virga-cli)"]
    end
    
    subgraph Server["🌐 C2 Infrastructure"]
        API["Admin API<br/>:8443"]
        HTTP["HTTP Listener<br/>:8080"]
        HTTPS["HTTPS Listener<br/>:443"]
        Core["Server Core"]
        DB[("SQLite DB")]
        MCP["MCP Servers<br/>STDIO/SSE/Stream"]
    end
    
    subgraph Targets["🎯 Target Systems"]
        Win["Windows Beacon<br/>+ LLM"]
        Linux["Linux Beacon<br/>+ LLM"]
        Mac["macOS Beacon<br/>+ LLM"]
    end
    
    CLI -.->|"TLS"| API
    API --> Core
    Core --> DB
    Core --> MCP
    
    Win -->|"AES-256-GCM"| HTTP
    Linux -->|"AES-256-GCM"| HTTPS
    Mac -->|"AES-256-GCM"| HTTP
    
    HTTP --> Core
    HTTPS --> Core
    
    style Operator fill:#2d3748,stroke:#1a202c,color:#fff
    style Server fill:#3c8772,stroke:#2d6659,color:#fff
    style Targets fill:#e53e3e,stroke:#c53030,color:#fff
```

## Next Steps

- To install Virgaceed to the [Installation](/guide/installation) guide.
- To get hands-on experience, follow the [Quick Start](/guide/quick-start) tutorial.