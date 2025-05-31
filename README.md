# Twitch-Kick-wiever-bot

This project is a Go-based utility to simulate multiple parallel stream viewers for a given Twitch or Kick stream link using Streamlink and MPV. It optionally supports proxy usage via a file and launches multiple workers with randomized delays and dynamic window positions.

## ⚙️ Features

* Launches multiple stream viewers simultaneously.
* Supports proxy usage (`socks5://` by default).
* Dynamic MPV window geometry for visual separation.
* Graceful shutdown with Ctrl+C.
* Retry logic via Streamlink options.

## 💪 Example Use Case

This tool is primarily intended for **testing and educational purposes**, such as:

* Evaluating the behavior of stream servers under load.
* Understanding how Streamlink and proxies interact.
* Learning Go's concurrency and process management.

## ❌ Disclaimer & Ethical Notice

> ❗️ **This tool is NOT intended for viewbotting or artificially inflating viewership on streaming platforms like Twitch or Kick.**

* I, the creator, **do not condone nor accept any responsibility** for unethical, abusive, or policy-violating use of this code.
* **All liability is disclaimed** for misuse or violation of platform terms of service.
* By using this tool, you **agree to take full responsibility** for your actions and to **use it only in compliance with all relevant terms, laws, and ethical standards**.

## 📁 Setup

### 1. Requirements

* **[Streamlink](https://streamlink.github.io/)** installed on your system.
* **[MPV Media Player](https://mpv.io/)** installed.
* Go 1.18 or newer.

### 2. File Structure

```bash
.
├── main.go
└── dataa.txt  # (optional) Contains one proxy per line, supports socks5://host:port format
```

### 3. Configuration

Edit the paths at the top of `main.go` to reflect your system setup:

```go
const (
    mpvPath        = `C:\Path\To\mpv.exe`
    streamlinkPath = `C:\Path\To\streamlink.exe`
)
```

### 4. Run

```bash
go run main.go
```

You will be prompted to enter the stream URL. The tool will then begin launching viewers.

## 🔐 Proxy Usage

If you provide a file named `dataa.txt`, each line should be a proxy:

```
127.0.0.1:9050
socks5://192.168.0.1:1080
```

If no proxy is specified, it will run directly without proxy usage.

## 🔄 Concurrency

The number of parallel stream launches is controlled by:

```go
const threads = 10
```

You can adjust this value to launch more or fewer stream viewers.

## 📜 License

This project is shared for educational and learning purposes. **No license is granted for usage violating any service terms or laws**. You may **not use this project for automation of fraudulent traffic, spamming, or malicious intent.**

> 🚩 **Use responsibly.**
