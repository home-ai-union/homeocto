# 🐾 HomeOcto

[![License](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://github.com/home-ai-union/homeocto/blob/main/LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/home-ai-union/homeocto)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20ARM-lightgrey.svg)](https://github.com/home-ai-union/homeocto)

[🇨🇳 中文文档](README-zh.md)

# HomeOcto: Understands People, Understands Home Better.

Say goodbye to tedious configuration, control your entire home with just one sentence.
Zero threshold: Control home appliances like chatting.
Localized: Works even offline, privacy stays at home.
Fully compatible: Connect devices from any brand.

---

## ✨ Core Features

| Feature | Description |
|:---|:---|
| 🌐 **Minimal Integration, Natural Conversation** | Native support for major ecosystems like Mi Home, Tuya, HomeKit, Matter. Discover and control devices through natural conversation without complex learning costs. |
| 🧠 **Intelligence Equality** | Devices only need to provide control interfaces. Thinking, scheduling, and linkage are all managed by the `HomeOcto` brain. "Smart" devices like smart locks, feeders, and robot vacuums are completely "de-brained" to focus on execution. |
| 💰 **Extremely Low Deployment Cost** | Perfectly adapted for Raspberry Pi, NAS, side routers, old phones, and other idle hardware. Requires only 50MB memory without large models, and 8GB for smooth operation with local models. |
| 🔒 **Localized Security** | Built-in local large model support. All data, reasoning, and commands are completed within the LAN. Cloud models can be called on demand, with absolutely controllable privacy. |
| 🚀 **Infinite Scenario Expansion** | From supervising homework, fire and theft prevention, to automatic feeding and smart cooking. Supports spatial perception control, habit learning, and dynamic device addition with boundless scenarios. |

---

## 🚀 Quick Start

### 📦 Deployment

For detailed installation steps, please refer to the [Installation Guide](./doc/install-step.md).

### 💻 Hardware Requirements

| Mode | Memory | Storage | Use Case |
|:---|:---|:---|:---|
| Pure Local Control | ≥ 50 MB | ≥ 100 MB | Basic device linkage, IM/speaker control |
| Local Large Model | ≥ 8 GB | ≥ 20 GB | Complete AI decision-making, complex scenario generation, privacy-sensitive environments |

### 📱 Supported Platforms

HomeOcto supports deployment on various hardware platforms:

- **Raspberry Pi**: Perfectly adapted for Raspberry Pi 4/5, low-power operation
- **Mobile Phones**: Transform old phones into smart home hubs
- **Linux**: Supports major distributions like Ubuntu, Debian, CentOS
- **macOS**: Supports both Intel and Apple Silicon chips
- **Windows**: Supports Windows 10/11
- **NAS**: Synology, QNAP, and other NAS devices
- **ARM Devices**: Various ARM architecture development boards

---

## 🎯 Typical Scenarios

### ⚡ Lightning Installation · One-Sentence Sync, Enjoy Premium Smart Home Instantly

- **Lightning Installation**: One-click APP installation, first-time wizard automatically completes model configuration and account authorization, syncs all devices
- **Incremental Sync**: After new devices join the network, one sentence "scan new devices" triggers AI to infer ownership, automatically identifying rooms and purposes
- **Cross-Platform Access**: WeChat, DingTalk, Telegram, speakers, send commands from any channel, unified response

---

### 🌫️ Real-Time Perception · Seamless Decision-Making, Blends Into Life Like Air

- **Lights on when people arrive, off when they leave, on at dusk, off at dawn**: No instructions needed, just natural
- **Rainy day reminders, warm greetings when returning home**: Intimately perceives life rhythm, says what needs to be said
- **Forgot to turn off stove or water, automatic detection, timely reminders**: Abnormalities don't stay overnight, safety isn't absent
- **Cats and dogs, scheduled feeding, pet zone temperature control, careful care**: As attentive as family, never misses anything

---

### 🧠 Self-Evolution · Truly Understands Your Premium Smart Home, Reflects Daily and Monthly

- **Remembers everything you say, automatically adapts next time**: Preference accumulation, no need to repeat
- **Daily review, monthly reflection, identify deviations, correct rules**: More accurate with use, quietly evolves
- **Multiple residents, permission isolation, individual habits, no interference**: Shared by family, different experiences
- **First day is assistant, first month is butler, first year is companion**: Truly understands you, walks with you

---

## 🔌 Supported Brands

### 📊 Integration Progress

| Brand | Device Addition | Cloud Control | Local Control | Local Video |
|:---|:---|:---|:---|:---|
| Xiaomi | ✅ Testing Complete | ✅ Testing Complete | ⏳ Not Started | ✅ Testing Complete |
| Tuya | Development Complete | Development Complete | ⏳ Not Started | ⏳ Not Started |
| HomeKit | Development Complete | N/A | Development Complete | ⏳ Not Started |
| Matter Protocol | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started |
| HomeAssistant | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started |
| Haier | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started |
| Midea | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started |
| Gree | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started |
| Wyze | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started |
| Roborock | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started | ⏳ Not Started |

---

## 💬 Community & Feedback

- 📮 **Issue Reporting**: [GitHub Issues](https://github.com/home-ai-bot/homeocto/issues)
- 💬 **Chat Group**: <img src="./doc/images/group1.jpg" alt="HomeOcto Open Source Group 1" width="400">
- 🌐 **Project Homepage**: [GitHub](https://github.com/home-ai-union/homeocto)
- 🗺️ **RoadMap**: [ROADMAP.md](./doc/roadmap.md)

---

## 📜 Open Source License

This project is open-sourced under the [GPL-3.0 License](https://github.com/home-ai-union/homeocto/blob/main/LICENSE).

---

## 🙏 Acknowledgments

HomeOcto stands on the shoulders of many excellent open-source projects. Sincere thanks to:

- [**picoclaw**](https://github.com/sipeed/picoclaw) — Core AI Agent engine, providing multi-channel conversation and tool scheduling capabilities
- [**go2rtc**](https://github.com/AlexxIT/go2rtc) — High-performance real-time streaming media framework, supporting camera integration and audio/video forwarding
- [**FFmpeg**](https://ffmpeg.org) — Multimedia processing infrastructure, industry standard for audio/video encoding and stream processing

---

> **"Let every device focus on execution, let every interaction be full of wisdom."**  
> 🐾 **HomeOcto** —— The Smart Home Brain of the AI Era.
