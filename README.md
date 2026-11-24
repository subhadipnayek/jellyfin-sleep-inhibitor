# Jellyfin Sleep Inhibitor

A lightweight, standalone Windows background service that monitors active Jellyfin playback sessions and prevents your Windows PC from sleeping while anyone is streaming. Once all users stop watching, it waits for a configurable idle grace period and releases the system sleep lock so Windows can follow its normal sleep and power plan.

---

## Key Highlights

- **Ultra-lightweight:** Idle memory footprint of only ~8 MB with virtually 0% CPU.
- **Single standalone executable:** Zero external dependencies (no .NET runtime, no DLL hell, no external interpreters).
- **Native Windows Power Management:** Uses Win32 `SetThreadExecutionState` with `ES_SYSTEM_REQUIRED` and `ES_AWAYMODE_REQUIRED` (keeps network streaming alive without waking displays).
- **Supports Multiple Concurrent Users:** Monitors DirectPlay, DirectStream, and Transcode sessions across web, mobile, TV, and desktop clients.
- **Smart Paused Playback Handling:** Distinguishes between active playback and long-paused sessions with a dedicated paused grace period.
- **Fail-Safe Protection:** Prevents unexpected sleep if Jellyfin temporarily restarts or has a network glitch.
- **Dual Execution Modes:** Run interactively in a console window for testing, or install as an auto-starting Windows Service with one command.

---

## Quick Start

### 1. Configure Jellyfin API Key

1. Open your Jellyfin Web Dashboard -> **Administration** -> **Dashboard**.
2. In the left sidebar, click **API Keys** (under *Advanced*).
3. Click the **+** button to create a new API Key (e.g. name it `SleepInhibitor`).
4. Copy `config.json.example` to `config.json` and paste your API key:

```json
{
  "jellyfin": {
    "server_url": "http://localhost:8096",
    "api_key": "PASTE_YOUR_JELLYFIN_API_KEY_HERE",
    "polling_interval_seconds": 15,
    "request_timeout_seconds": 10
  },
  "power": {
    "idle_grace_period_minutes": 5,
    "ignore_paused": false,
    "paused_grace_period_minutes": 15,
    "keep_awake_on_network_error": true,
    "require_away_mode": true
  },
  "logging": {
    "level": "info",
    "log_to_file": true,
    "log_file_path": "jellyfin_sleep_inhibitor.log"
  }
}
```

### 2. Test in Console Mode

Before installing as a background service, test it directly in PowerShell or Command Prompt:

```powershell
.\jellyfin-sleep-inhibitor.exe run
```

Start streaming media on any client (phone, TV, browser). You will see real-time log entries indicating who is watching, what media is playing, and that sleep inhibition is active.

### 3. Install as an Auto-Starting Windows Service

Open an **Elevated (Administrator) PowerShell / CMD**:

```powershell
# 1. Install service (sets Automatic startup with Windows)
.\jellyfin-sleep-inhibitor.exe install

# 2. Start the service
.\jellyfin-sleep-inhibitor.exe start

# 3. Verify status
.\jellyfin-sleep-inhibitor.exe status
```

To stop or uninstall:

```powershell
.\jellyfin-sleep-inhibitor.exe stop
.\jellyfin-sleep-inhibitor.exe uninstall
```

---

## How It Works

```
                     +---------------------------------------+
                     |    Jellyfin Server (/Sessions API)    |
                     +-------------------+-------------------+
                                         |
                       Poll every 15s    | Returns active sessions
                                         v
                     +---------------------------------------+
                     |       Jellyfin Sleep Inhibitor        |
                     +-------------------+-------------------+
                                         |
                  +----------------------+----------------------+
                  |                                             |
                  v                                             v
        [Active Streaming > 0]                          [0 Active Sessions]
                  |                                             |
                  v                                             v
       SetThreadExecutionState:                     Start Idle Grace Period (5m)
        ES_CONTINUOUS                                           |
      | ES_SYSTEM_REQUIRED                                      v
      | ES_AWAYMODE_REQUIRED                         Grace expired?
                  |                                             |
                  v                                             v
      PC Kept Awake (Away Mode)                     SetThreadExecutionState:
   (Visible in 'powercfg /requests')                 ES_CONTINUOUS (Sleep Allowed)
```

### Verifying with Windows Powercfg

While someone is streaming, you can verify Windows is holding the sleep lock by running:

```cmd
powercfg /requests
```

Under `[SYSTEM]`, you will see:
```text
[SYSTEM]
[SERVICE] JellyfinSleepInhibitor (Jellyfin Sleep Inhibitor Service)
```

---

## Configuration Reference

| Parameter | Default | Description |
|---|---|---|
| `jellyfin.server_url` | `http://localhost:8096` | Base URL of your Jellyfin server. |
| `jellyfin.api_key` | `""` | Jellyfin API token generated in Dashboard -> API Keys. |
| `jellyfin.polling_interval_seconds` | `15` | How frequently (in seconds) the service checks for active streams. |
| `power.idle_grace_period_minutes` | `5` | Minutes to keep the PC awake after all users finish streaming. |
| `power.ignore_paused` | `false` | If `true`, paused streams are ignored. If `false`, paused streams keep PC awake up to `paused_grace_period_minutes`. |
| `power.paused_grace_period_minutes` | `15` | Maximum duration (in minutes) a paused playback session keeps the system awake before being marked idle. |
| `power.keep_awake_on_network_error` | `true` | If Jellyfin is restarting or unreachable, temporarily keeps the PC awake as a fail-safe. |
| `power.require_away_mode` | `true` | Enables Windows Away Mode so screens turn off while background network streaming stays active. |
| `logging.log_to_file` | `true` | Writes persistent logs to `jellyfin_sleep_inhibitor.log`. |

---

## Building from Source

Requires Go (v1.24+):

```powershell
go build -o jellyfin-sleep-inhibitor.exe .
```
