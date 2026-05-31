# Runtime binaries layout

Windows (legacy):

- `bin/xray.exe`
- `bin/sing-box.exe`

Linux (new):

- `bin/linux-amd64/xray`
- `bin/linux-amd64/sing-box`
- `bin/linux-arm64/xray`
- `bin/linux-arm64/sing-box`

macOS (unsigned internal builds):

- `bin/darwin-amd64/xray`
- `bin/darwin-amd64/sing-box`
- `bin/darwin-arm64/xray`
- `bin/darwin-arm64/sing-box`

Runtime hashes are pinned in `publish/runtime-manifest.json`.
Pinned upstream archive sources are tracked in `publish/runtime-sources.json`.
Use `python3 tools/runtime/sync-runtime.py --target <target>` to refresh runtime files safely.
