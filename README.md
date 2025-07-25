<div align="center">

# 🐳 VMDocker

**A Docker-based Virtual Machine Implementation for HyMatrix Computing Network**

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-27.3.x-blue.svg)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![HyMatrix](https://img.shields.io/badge/HyMatrix-Compatible-orange.svg)](https://hymatrix.com/)

</div>

## 📖 Overview

**VMDocker** is a high-performance, Docker-based virtual machine implementation designed for the `HyMatrix` computing network. It serves as a universal virtual machine extension that can be seamlessly `mounted` to HyMatrix nodes, enabling scalable and verifiable computation execution.

### 🌟 Key Features

- **🔌 Universal VM Interface**: Compatible with standard HyMatrix VM protocol
- **🐳 Docker-based**: Leverages Docker containers for isolated computation environments
- **🔄 Multi-Architecture Support**: Supports EVM, WASM, AO, LLM model services, and more
- **📊 Checkpoint & Restore**: Advanced state management with CRIU integration
- **⚡ High Performance**: Optimized for scalable computation workloads
- **🔗 AO Compatible**: Full support for AO protocol containers

### 🏗️ Architecture

```
┌─────────┐    ┌──────────┐    ┌───────────┐
│ HyMatrix│───▶│ VMDocker │───▶│ Container │
│  Node   │    │          │    │(EVM/WASM) │
└─────────┘    └──────────┘    └───────────┘
```

### 🔗 About HyMatrix

**HyMatrix** is an infinitely scalable decentralized computing network that decouples computation from consensus by anchoring execution logs in immutable storage (Arweave), enabling verifiable, trustless computation anywhere.

**Learn more**: 
> - 🌐 [HyMatrix Website](https://hymatrix.com/)
> - 📖 [HyMatrix Documentation](https://docs.hymatrix.com/)

### 🛠️ VM Interface

VMDocker implements the standard HyMatrix VM interface:

```go
// hymx/vmm/schema/schema.go
type Vm interface {
    Apply(from string, meta Meta) (res *Result, err error)
    Checkpoint() (data string, err error)
    Restore(data string) error
    Close() error
}
```

**Supported Container Types**:
- 🔷 **EVM**: Ethereum Virtual Machine
- 🟦 **WASM**: WebAssembly runtime
- 🟠 **AO**: Arweave AO protocol ([Container Repository](https://github.com/cryptowizard0/vmdocker_container))
- 🤖 **LLM**: Large Language Model services
- ➕ **Custom**: Any containerized computation environment

## 🚀 Getting Started

### 📋 Prerequisites

| Component | Version | Platform | Required |
|-----------|---------|----------|----------|
| **Operating System** | Linux | Any | ✅ |
| **Go** | 1.24.0+ | Any | ✅ |
| **Docker** | 27.3.x | Any | ✅ |
| **Redis** | Latest | Any | ✅ |
| **Clang/GCC** | Latest | Any | ✅ (for CGO) |
| **CRIU** | v4.1 | Linux only | ⚠️ (for checkpoint) |

> ⚠️ **Note**: CRIU is only required for checkpoint functionality and is Linux-specific. macOS users can skip CRIU installation.

### 📦 Installation

#### 1. Install System Dependencies

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install gcc build-essential redis-server
```

**CentOS/RHEL:**
```bash
sudo yum install gcc gcc-c++ make redis
```

#### 2. Clone Repository

```bash
git clone https://github.com/cryptowizard0/vmdocker.git
cd vmdocker
```

#### 3. Install Dependencies

```bash
go mod tidy
```

#### 4. Build VMDocker

```bash
go build -o ./build/hymx-node ./cmd
```


### 🔧 Optional: CRIU Installation (Linux Only)

> 📝 **Required for**: Checkpoint and restore functionality
> 🖥️ **Platform**: Linux systems only

#### Install CRIU v4.1

```bash
# Download CRIU v4.1 source code
wget https://github.com/checkpoint-restore/criu/archive/criu_v4.1.tar.gz
tar -xzf criu_v4.1.tar.gz
cd criu-criu_v4.1

# Compile and install
make
sudo make install

# Verify installation
criu check
# Expected output: "Looks good."
```

### 🐳 Docker Configuration

> ⚠️ **Important**: Docker version `27.3.x` is required for optimal compatibility.

#### Enable Experimental Features

Docker checkpoint requires experimental features to be enabled:

```bash
# Create Docker daemon configuration
sudo mkdir -p /etc/docker

# Enable experimental features
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "experimental": true
}
EOF

# Restart Docker service
sudo systemctl restart docker

# Verify experimental features are enabled
docker info | grep "Experimental"
# Expected output: "Experimental: true"
```

## ⚙️ Configuration

### 📝 Create Configuration File

> 📚 **For detailed configuration options**, see [HyMatrix Configuration Documentation](https://docs.hymatrix.com/docs/join-the-network/setup)

VMDocker uses standard HyMatrix configuration format. Create a `config.yaml` file:

```yaml
# 🌐 Node Service Configuration
port: :8080
ginMode: release  # Options: "debug", "release"

# 🔴 Redis Configuration
redisURL: redis://@localhost:6379/0

# 🌍 Storage & Network
arweaveURL: https://arweave.net
hymxURL: http://127.0.0.1:8080

# 🔐 Node Identity (Wallet)
prvKey: 0x64dd2342616f385f3e8157cf7246cf394217e13e8f91b7d208e9f8b60e25ed1b
keyfilePath:  # Optional: path to keyfile instead of prvKey

# ℹ️ Node Information
nodeName: test1
nodeDesc: first test node
nodeURL: http://127.0.0.1:8080

# 🔗 Network Participation
joinNetwork: false  # Set to true for production network
```

### 📊 Configuration Reference

| Field | Type | Description | Example |
|-------|------|-------------|----------|
| `port` | string | HTTP server port | `:8080` |
| `ginMode` | string | Gin framework mode | `release` or `debug` |
| `redisURL` | string | Redis connection URL | `redis://@localhost:6379/0` |
| `arweaveURL` | string | Arweave gateway URL | `https://arweave.net` |
| `hymxURL` | string | Local node URL for SDK calls | `http://127.0.0.1:8080` |
| `prvKey` | string | Ethereum private key (hex) | `0x64dd...` |
| `keyfilePath` | string | Alternative to prvKey | `./keyfile.json` |
| `nodeName` | string | Node identifier | `my-node` |
| `nodeDesc` | string | Node description | `Production node` |
| `nodeURL` | string | Public node URL | `https://my-node.com` |
| `joinNetwork` | boolean | Join HyMatrix network | `false` (testing), `true` (production) |



## 🔧 Module Setup

### 📦 Generate Module Configuration

1. **Configure Example Settings**

   Edit the configuration in `examples/main.go`:

   ```go
   // examples/main.go
   var (
       url    = "http://127.0.0.1:8080"  // Your node URL
       prvKey = "0x64dd2342616f385f3e8157cf7246cf394217e13e8f91b7d208e9f8b60e25ed1b"  // Your private key
       
       signer, _  = goether.NewSigner(prvKey)
       bundler, _ = goar.NewBundler(signer)
       s          = sdk.NewFromBundler(url, bundler)
   )
   ```

2. **Generate Module File**

   ```bash
   cd examples
   go run ./ module
   ```

   This will generate a `mod-xxxx.json` file containing your module configuration.

3. **Install Module**

   ```bash
   # Copy the generated module file to the modules directory
   cp mod-*.json ../cmd/mod/
   ```

## 🚀 Running VMDocker

### 1. 🔴 Start Redis Server

Ensure Redis is running before starting VMDocker:

```bash
# Ubuntu/Debian
sudo systemctl start redis-server
sudo systemctl enable redis-server

# CentOS/RHEL
sudo systemctl start redis
sudo systemctl enable redis

# macOS (with Homebrew)
brew services start redis
```

### 2. 🚀 Launch VMDocker Node

```bash
# From the project root directory
./build/hymx-node --config ./config.yaml
```

### 3. ✅ Verify Startup

Successful startup will display:

```
INFO[07-25|00:00:01] server is running   module=node-v0.0.1 wallet=0x... port=:8080
```

## 🌐 Network Participation

### 🔗 Join HyMatrix Network

To participate as a network node operator:

1. **Configure for Production**
   ```yaml
   joinNetwork: true
   nodeURL: https://your-public-domain.com  # Your public URL
   ```

2. **Stake HMX Tokens**
   - Acquire the required HMX tokens
   - Complete the staking process

3. **Complete Registration**
   - Submit node registration
   - Wait for network acceptance

### 💰 Rewards

Participating nodes earn rewards for:
- ⚡ **Computation execution**
- 📝 **Log submission**
- 🔗 **Network services**
- 🛡️ **Network security**

> 📖 **For detailed network joining instructions**, see [HyMatrix Network Documentation](https://docs.hymatrix.com/docs/category/join-the-network)

## 💻 Usage

### 🔗 AOS Client Integration

VMDocker is fully AO-compatible and can be used with the modified AOS client.

#### 1. 📥 Setup AOS Client

```bash
# Clone the modified AOS repository
git clone https://github.com/cryptowizard0/aos
cd aos

# Install Node.js dependencies
npm install
```

#### 2. 🚀 Launch AOS Client

**First-time setup:**
```bash
DEBUG=true node src/index.js \
  --cu-url=http://127.0.0.1:8080 \
  --mu-url=http://127.0.0.1:8080 \
  --scheduler=0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51 \
  test_name
```

**Reconnect to existing process:**
```bash
DEBUG=true node src/index.js \
  --cu-url=http://127.0.0.1:8080 \
  --mu-url=http://127.0.0.1:8080 \
  --scheduler=0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51 \
  {{process_id}}
```

#### 📋 Parameter Reference

| Parameter | Description | Example |
|-----------|-------------|----------|
| `cu-url` | Compute Unit URL (same as VMDocker) | `http://127.0.0.1:8080` |
| `mu-url` | Message Unit URL (same as VMDocker) | `http://127.0.0.1:8080` |
| `scheduler` | VMDocker node ID | `0x972AeD...` |
| `process_id` | Existing process ID for reconnection | `abc123...` |

> 💡 **Tip**: Save your Process ID after the first launch for future reconnections!

### 📚 Examples

Explore the `examples/` directory for reference implementations:

```bash
ls examples/
# Available examples:
# - checkpoint.go   # Checkpoint and restore functionality
# - eval.go         # Expression evaluation
# - inbox.go        # Message inbox handling
# - module.go       # Module management
# - pingpong.go     # Basic communication test
# - spawn.go        # Process spawning
# - token.go        # Token operations
# - stress.go       # Performance testing
```

#### 🏃‍♂️ Run Examples

```bash
cd examples

# Run a specific example
go run . <example_name>

# Example: Run ping-pong test
go run . pingpong
```

## 🔧 API Reference

VMDocker exposes standard HyMatrix VM interface endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/apply` | POST | Execute computation |
| `/checkpoint` | POST | Create state checkpoint |
| `/restore` | POST | Restore from checkpoint |
| `/health` | GET | Health check |

## 🐛 Troubleshooting

### Common Issues

**Redis Connection Failed**
```bash
# Check Redis status
sudo systemctl status redis-server

# Restart Redis
sudo systemctl restart redis-server
```

**Docker Permission Denied**
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Restart session or run:
newgrp docker
```

**CRIU Check Failed**
```bash
# Install missing dependencies
sudo apt-get install criu

# Verify installation
criu check
```


## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- 🌐 [HyMatrix Website](https://hymatrix.com/)
- 📖 [Documentation](https://docs.hymatrix.com/)
- 🐳 [Container Repository](https://github.com/cryptowizard0/vmdocker_container)
- 🔧 [AOS Client](https://github.com/cryptowizard0/aos)

---

<div align="center">

**Built with ❤️ for the HyMatrix ecosystem**

</div>
