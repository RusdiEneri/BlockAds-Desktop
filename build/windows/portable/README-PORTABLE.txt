========================================================================
BLOCKADS DESKTOP — PORTABLE MODE DISTRIBUTION
========================================================================

Welcome to BlockAds Desktop (Portable Edition).

1. WHAT IS PORTABLE MODE?
-------------------------
In Portable Mode, BlockAds Desktop operates completely self-contained.
All persistent application state—including the SQLite database (blockads.db),
filter-list snapshots, query logs, and user configuration—is stored inside
a local "data" directory right next to BlockAds-Desktop.exe.

Portable Mode is activated automatically because "portable.marker" is
present in this directory.

2. RUNTIME PREREQUISITES
-------------------------
- Operating System: 64-bit Windows 10 or Windows 11 (amd64 / x86_64).
- WebView2 Runtime: Microsoft Edge WebView2 Runtime is required by Wails v2.
  (It is preinstalled on Windows 11 and up-to-date Windows 10 systems.
  If missing, download the Evergreen Bootstrapper from:
  https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
- Directory Permissions: Ensure this folder is placed in a writable location
  (e.g., C:\Tools\BlockAds, or a USB flash drive). Do not run from a
  read-only network share or CD/DVD.

3. HOW TO RUN
-------------
Double-click `BlockAds-Desktop.exe` to launch the application.
On the first run, it will automatically initialize `data/blockads.db`
and set up the local DNS filtering engine on loopback (127.0.0.1:5335).

4. BACKUP AND RESTORATION
-------------------------
- To backup your settings, custom rules, and filter subscriptions, simply copy
  the entire `data/` folder.
- To transfer BlockAds Desktop to another PC or USB drive, copy the entire
  portable directory.
- No files are written to %APPDATA% or %LOCALAPPDATA% when running in portable
  mode.

5. START WITH WINDOWS IN PORTABLE MODE
---------------------------------------
If you enable "Start with Windows" in Settings while in portable mode, the
application registers its current full executable path in your personal Windows
Run registry (HKCU). If you move the portable folder to a new location,
BlockAds Desktop will detect the moved path on next launch and prompt you to
re-synchronize or update the path.

6. SYSTEM PRIVACY & BOUNDARIES
------------------------------
BlockAds Desktop does NOT install any kernel drivers, does NOT install background
system services, and does NOT silently change your Windows network adapter DNS
settings. It binds a high-performance local DNS server on 127.0.0.1:5335 that
forwards filtered queries over encrypted DNS-over-HTTPS (DoH).

========================================================================
Project repository: https://github.com/RusdiEneri/BlockAds-Desktop
License: MIT License
========================================================================
