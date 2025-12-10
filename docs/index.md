---
layout: home

hero:
  name: "Virga"
  text: "C2 Framework powered by LLM models"
  tagline: "Autonomous post-exploitation capabilities powered by embedded LLM"
  image:
    src: /virga_logo.svg
    alt: Virga
  actions:
    - theme: brand
      text: Get Started
      link: /guide/introduction
    - theme: alt
      text: View on GitHub
      link: https://github.com/r74tech/virga

features:
  - icon: 🤖
    title: AI-Powered Operations
    details: Integrated LLM model enables autonomous reconnaissance, system analysis, and intelligent task execution without operator intervention.
  - icon: 🚀
    title: High Performance
    details: Built with Go for exceptional performance and cross-platform compatibility. Supports Windows, Linux, and macOS on both x86_64 and ARM64 architectures.
  - icon: 🔒
    title: Security Features
    details: Memory-based operations with MemDB, configurable logging, and customizable sleep intervals with jitter for operational security.
  - icon: 🌐
    title: Flexible Communication
    details: HTTP/HTTPS transport with AES-256-GCM encryption, configurable jitter, and custom user agents.
---


## Demo

<NuAsciinemaPlayer
  src="asciinema/index.cast"
  :preload="true"
  :cols="161"
  :rows="38"
  :auto-play="false"
  :controls="true"
  :terminal-font-size="'12px'"
  :loop="false"
/>

## Legal Notice

Virga is designed exclusively for authorized security testing, research, and educational purposes. Users must:

- Only use this framework in environments where they have explicit permission
- Comply with all applicable laws and regulations
- Understand that misuse may result in severe legal consequences

The developers assume no liability for any misuse or damage caused by this framework.
