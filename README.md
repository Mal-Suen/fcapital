# fcapital

<div align="center">

```
  ███████╗ ██████╗ █████╗ ██╗     ███████╗██╗██████╗ ███████╗
  ██╔════╝██╔════╝██╔══██╗██║     ██╔════╝██║██╔══██╗██╔════╝
  █████╗  ██║     ███████║██║     █████╗  ██║██║  ██║█████╗
  ██╔══╝  ██║     ██╔══██║██║     ██╔══╝  ██║██║  ██║██╔══╝
  ██║     ╚██████╗██║  ██║███████╗███████╗██║██████╔╝███████╗
  ╚═╝      ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝╚═╝╚═════╝ ╚══════╝
```

**A Comprehensive Penetration Testing Framework**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## ⚠️ Disclaimer

**fcapital is designed for authorized security testing and educational purposes only.**

Unauthorized use of this tool against systems you do not own or have explicit permission to test is **ILLEGAL**. By using fcapital, you agree to:

1. Only test systems you own or have written authorization to test
2. Comply with all applicable laws and regulations
3. Accept full responsibility for your actions

---

## 📖 Overview

fcapital is a penetration testing framework that integrates multiple security tools with a unified interface. It provides both **interactive menu** and **command-line interface** for various security testing tasks.

### Key Features

- 🎯 **Unified Interface** - Single entry point for multiple tools
- 🔧 **Tool Management** - Automatic detection and installation of dependencies
- 🖥️ **Dual Mode** - Interactive menu and CLI support
- ⚡ **Go Performance** - Fast and efficient execution
- 📦 **Easy Integration** - Seamlessly integrates with popular security tools

---

## 🛠️ Supported Tools

| Tool | Category | Description | Kali |
|------|----------|-------------|------|
| nmap | Port Scan | Network Security Scanner | ✅ |
| dirsearch | Web Scan | Web Path Scanner | ✅ |
| dirb | Web Scan | Web Content Scanner | ✅ |
| gobuster | Web Scan | Directory/File/DNS Busting Tool | ✅ |
| ffuf | Web Scan | Fast Web Fuzzer | ✅ |
| sqlmap | Vuln Scan | Automatic SQL Injection Tool | ✅ |
| wpscan | Web Scan | WordPress Security Scanner | ✅ |
| hydra | Password | Network Logon Cracker | ✅ |
| nuclei | Vuln Scan | Vulnerability Scanner | ❌ |
| subfinder | Subdomain | Subdomain Discovery Tool | ❌ |
| httpx | Recon | HTTP Toolkit | ❌ |
| dnsx | Recon | DNS Toolkit | ❌ |

---

## 🚀 Installation

### Prerequisites

- Go 1.21 or higher
- Git

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourname/fcapital.git
cd fcapital

# Install dependencies
make install

# Build
make build

# Run
./build/fcapital
```

### Using Go Install

```bash
go install github.com/yourname/fcapital/cmd/fcapital@latest
```

---

## 📚 Usage

### Interactive Mode

```bash
fcapital
```

### CLI Mode

```bash
# Check dependencies
fcapital deps check

# List supported tools
fcapital deps list

# HTTP probe
fcapital recon http -t example.com

# Subdomain enumeration
fcapital subdomain passive -d example.com

# Port scan
fcapital portscan quick -t 192.168.1.1
fcapital portscan full -t 192.168.1.1
fcapital portscan custom -t 192.168.1.1 -p 80,443,8080

# Directory scan
fcapital webscan dir -t https://example.com

# Vulnerability scan
fcapital vulnscan nuclei -t https://example.com
fcapital vulnscan sqlmap -t https://example.com?id=1
```

---

## 📁 Project Structure

```
fcapital/
├── cmd/
│   └── fcapital/          # Main entry point
│       └── main.go
├── internal/
│   ├── cli/               # CLI commands
│   │   ├── root.go
│   │   ├── banner.go
│   │   ├── interactive.go
│   │   ├── deps.go
│   │   ├── recon.go
│   │   ├── subdomain.go
│   │   ├── portscan.go
│   │   ├── webscan.go
│   │   └── vulnscan.go
│   ├── core/
│   │   └── toolmgr/       # Tool manager
│   │       ├── manager.go
│   │       └── runner.go
│   └── modules/           # Feature modules
│       ├── recon/
│       ├── subdomain/
│       ├── portscan/
│       ├── webscan/
│       ├── vulnscan/
│       ├── password/
│       └── utils/
├── configs/
│   ├── config.yaml        # Main config
│   ├── tools.yaml         # Tools config
│   └── wordlists/         # Wordlists
├── docs/                  # Documentation
├── scripts/               # Helper scripts
├── Makefile
├── go.mod
└── README.md
```

---

## ⚙️ Configuration

Configuration file is located at `~/.fcapital/config.yaml` or `./configs/config.yaml`.

```yaml
# Output settings
output:
  format: text  # text, json, csv, html
  color: true
  verbose: false

# Tool settings
tools:
  local_path: "~/.fcapital/tools"
  timeout: 10m

# Module defaults
modules:
  webscan:
    default_tool: "dirsearch"
    wordlist: "configs/wordlists/dirs.txt"
```

---

## 🔧 Development

```bash
# Run tests
make test

# Lint code
make lint

# Format code
make fmt

# Development mode with hot reload
make dev

# Cross-compile
make cross
```

---

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

Inspired by:
- [fsociety](https://github.com/Manisso/fsociety) - A Penetration Testing Framework
- [ProjectDiscovery](https://github.com/projectdiscovery) - httpx, subfinder, nuclei, dnsx, ffuf

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request
