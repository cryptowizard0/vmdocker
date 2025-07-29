<div align="center">

# 🐳 VMDocker

**A Docker-based Virtual Machine Implementation for HyMatrix Computing Network**

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-27.3.x-blue.svg)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![HyMatrix](https://img.shields.io/badge/HyMatrix-Compatible-orange.svg)](https://hymatrix.com/)

</div>

## 📖 Overview

**VMDocker** is a high-performance, Docker-based virtual machine implementation designed for the HyMatrix computing network. It serves as a universal virtual machine extension that can be seamlessly mounted to HyMatrix nodes, enabling scalable and verifiable computation execution.

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
│ HyMatrix│───▶│VMDocker  │───▶│Container  │
│  Node   │    │ Manager  │    │(EVM/WASM) │
└─────────┘    └──────────┘    └───────────┘
```

### 🔗 About HyMatrix

**HyMatrix** is an infinitely scalable decentralized computing network that decouples computation from consensus by anchoring execution logs in immutable storage (Arweave), enabling verifiable, trustless computation anywhere.

🌐 **Learn more**: [https://hymatrix.com/](https://hymatrix.com/)

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

#### 1. Clone Repository

```bash
git clone https://github.com/cryptowizard0/vmdocker.git
cd vmdocker
```

#### 2. Install Dependencies

```bash
go mod tidy
```

#### 3. Build VMDocker

```bash
go build -o ./build/hymx-node ./cmd
```

#### 4. Install System Dependencies

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install gcc build-essential redis-server
```

**CentOS/RHEL:**
```bash
sudo yum install gcc gcc-c++ make redis
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

> 📚 **For detailed configuration options**, see [HyMatrix Configuration Documentation](https://docs.hymatrix.com/docs/join-the-network/setup)

## 📋 Module Configuration

### 🏷️ Module Format Requirements

VMDocker modules must follow specific format requirements to ensure proper container execution:

#### **ModuleFormat Specification**
- **Required Prefix**: `web.vmdocker-`
- **Format Pattern**: `web.vmdocker-{runtime}-{version}`
- **Examples**:
  - `web.vmdocker-golua-ao.v0.0.1`
  - `web.vmdocker-wasm-ao.v1.0.0`
  - `web.vmdocker-evm-ao.v2.1.0`

#### **Required Tags**

Every VMDocker module **MUST** include the following tags:

| Tag Name | Description | Example |
|----------|-------------|----------|
| `Image-Name` | Docker image name and tag | `chriswebber/docker-golua:v0.0.2` |
| `Image-ID` | Docker image SHA256 digest | `sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b` |

> ⚠️ **Important**: Both `Image-Name` and `Image-ID` tags are mandatory. Missing either tag will cause module validation to fail.

#### **Create Your Own Module**

Follow these steps to create and deploy your custom VMDocker module:

**Step 1: Modify Module Configuration**

Edit the `examples/module.go` file and fill in your module information:

```go
// examples/module.go
item, _ := s.GenerateModule([]byte{}, schema.Module{
    Base:         schema.DefaultBaseModule,
    ModuleFormat: "web.vmdocker-golua-ao.v0.0.1",  // Must start with "web.vmdocker-"
    Tags: []arSchema.Tag{
			{Name: "Image-Name", Value: "chriswebber/docker-golua:v0.0.2"},
			{Name: "Image-ID", Value: "sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b"},
		},
})
```

**Step 2: Generate Module File**

Run the command in the `examples` directory to generate the module configuration file:

```bash
cd examples
go run ./ module
```

This will generate a `mod-xxxx.json` file containing your module configuration.

**Step 3: Deploy Module**

Copy the generated module file to the VMDocker modules directory:

```bash
# Copy the generated module file to cmd/mod/ directory
cp mod-*.json ../cmd/mod/
```

**Step 4: Verify Deployment**

Check if the module file is correctly deployed:

```bash
ls ../cmd/mod/mod-*.json
```

Now your custom module is ready to use in VMDocker!

#### **Validation Process**

VMDocker automatically validates modules using the `checkModule` function:

1. ✅ **ModuleFormat Check**: Verifies format starts with `web.vmdocker-`
2. ✅ **Image-Name Check**: Ensures `Image-Name` tag exists and is not empty
3. ✅ **Image-ID Check**: Ensures `Image-ID` tag exists and is not empty

If any validation fails, the module will be rejected and container creation will fail.

#### **Getting Image SHA256**

To obtain the correct `Image-ID` value:

```bash
# Pull the image
docker pull chriswebber/docker-golua:v0.0.2

# Get the SHA256 digest
docker inspect chriswebber/docker-golua:v0.0.2 --format='{{.Id}}'
# Output: sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b
```

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

## 使用

### Run AOS Client

vmdocker is an AO-compatible system. Use the modified AOS to connect to vmdocker.

1. Clone AOS repository:
   ```bash
   git clone https://github.com/cryptowizard0/aos
   ```

2. Install Node.js dependencies:
   ```bash
   npm install
   ```

3. Start AOS client:
    - `cu-url` and `mu-url` should be the same as the vmdocker node url
    - `scheduler` is the vmdocker node id
   ```bash
   DEBUG=true node src/index.js \
    --cu-url=http://127.0.0.1:8080 \
    --mu-url=http://127.0.0.1:8080 \
    --scheduler=0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51 \
    test_name
   ``` 

   After the first launch, please record your Process ID. To reconnect to the specific process later, use the following command:

   ```bash
   DEBUG=true node src/index.js \
    --cu-url=http://127.0.0.1:8080 \
    --mu-url=http://127.0.0.1:8080 \
    --scheduler=0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51 \
    {{pricessid}}
   ```

### Examples

Reference implementations are available in the `examples` directory.
