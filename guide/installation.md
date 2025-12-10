---
url: /virga/guide/installation.md
---
# Installation

This guide provides instructions for installing and building Virga from source.

## Prerequisites

Before you begin, ensure you have the following software installed on your system:

* **Go**: Version 1.21 or higher.
* **Git**: For cloning the source code repository.
* **Make**: For using the provided `Makefile` to simplify the build process.
* **A C Compiler**: Such as GCC or Clang, which is required for CGO support (necessary for some dependencies).

```bash

# install C compiler for Windows
sudo apt install -y g++-mingw-w64-x86-64
sudo apt install -y mingw-w64-x86-64-dev

# set the default g++ compiler for Windows
sudo update-alternatives --config x86_64-w64-mingw32-g++
# output:
There are 2 choices for the alternative x86_64-w64-mingw32-g++ (providing /usr/bin/x86_64-w64-mingw32-g++).

  Selection    Path                                   Priority   Status
------------------------------------------------------------
  0            /usr/bin/x86_64-w64-mingw32-g++-win32   60        auto mode
* 1            /usr/bin/x86_64-w64-mingw32-g++-posix   30        manual mode
  2            /usr/bin/x86_64-w64-mingw32-g++-win32   60        manual mode

Press <enter> to keep the current choice[*], or type selection number: 1

# set the default gcc compiler for Windows
sudo update-alternatives --config x86_64-w64-mingw32-gcc
# output:
There are 2 choices for the alternative x86_64-w64-mingw32-gcc (providing /usr/bin/x86_64-w64-mingw32-gcc).

  Selection    Path                                   Priority   Status
------------------------------------------------------------
  0            /usr/bin/x86_64-w64-mingw32-gcc-win32   60        auto mode
* 1            /usr/bin/x86_64-w64-mingw32-gcc-posix   30        manual mode
  2            /usr/bin/x86_64-w64-mingw32-gcc-win32   60        manual mode

Press <enter> to keep the current choice[*], or type selection number: 1

```

## Building from Source

This is the standard method for installing Virga

### 1. Clone the Repository

First, clone the repository from GitHub to your local machine:

```bash
git clone https://github.com/r74tech/virga.git
cd virga
```

### 2. Build the Binaries

The project includes a `Makefile` that automates the build process.

To build all server and client components, run:

```bash
make build
```

This command compiles the following binaries and places them in the `./bin` directory:

* `virga-server`: The main C2 server.
* `virga-cli`: The command-line interface for interacting with the server.
* `virga-mcp-stdio`: An optional server for the Model Context Protocol (MCP) over standard I/O.

### 3. Install AI Dependencies (Optional)

If you plan to use the AI features, you must download the required model files and libraries. The `Makefile` provides a convenient target for this.

```bash
# This will download the model and libraries for your current OS/architecture
make download-llama-all
make deps
```

This command will populate the `internal/implant/llama/models` and `internal/implant/llama/libs` directories.

### 4. Generate Secure Encryption Keys (Recommended)

For production deployments, it's critical to generate and use your own encryption keys instead of the default ones:

```bash
make generate-key
```

This will generate secure AES-256 encryption keys and provide instructions on how to use them. You can either:

* Set the key as an environment variable: `export VIRGA_ENCRYPTION_KEY='<your-key-here>'`
* Add it to your configuration file under the `encryption` section

> **⚠️ Security Warning**: Never use the default encryption key in production environments. Always generate a unique key for each deployment.

## Verifying the Installation

After the build process is complete, you can verify that the components were built correctly.

1. **Check for Binaries**: List the contents of the `./bin` directory to ensure the server and CLI executables are present.

   ```bash
   ls -l bin/
   ```

2. **Run the Server**: Start the server with the default configuration file.

   ```bash
   ./bin/virga-server --config configs/server.yaml
   ```

   If successful, you will see a message indicating the server has started.

3. **Connect with the CLI**: In a separate terminal, connect to the running server.

   ```bash
   ./bin/virga-cli --host localhost --port 8443
   ```

   You should be greeted with the `virga>` prompt.

## Uninstallation

To uninstall Virgaply remove the project directory that you cloned from GitHub.

```bash
# Navigate out of the project directory
cd ..

# Remove the directory and all its contents
rm -rf virga/
```
