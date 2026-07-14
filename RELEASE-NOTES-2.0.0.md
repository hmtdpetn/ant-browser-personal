# Ant Browser Personal 2.0.0

Ant Browser Personal 2.0.0 continues from the personal 1.3.0 mainline. It preserves the personal proxy connector rules and compatibility behavior instead of replacing them with upstream defaults.

## Highlights

- Per-subscription User-Agent selection, custom UA input, fallback rotation, and persisted UA reuse on refresh.
- Tree instance groups with descendant filtering, persistent sidebar collapse, and batch moves.
- Persistent tree proxy groups with child groups, rename/delete/promote operations, filtering, legacy `group_name` migration, and batch moves.
- Automatic database v13/v14 migrations, including subscription UA settings and proxy group data; backup merge remains compatible with old and new proxy fields.
- High-resolution multi-size instance icons, horizontal H1/H2 badges, modal layering fix, collapsible group panels, and persistent horizontal-only table column resizing.
- Fixed subscription refresh so it retains a proxy's tree group ID and cannot misplace same-named child groups at the root.

## Upgrade

Back up the current installation first. For an existing installation, retain its `data/` directory and personal configuration, then replace only the application/runtime files with the new version. First startup performs the v13/v14 migrations automatically.

Do not open the migrated database directly with an older version. To roll back, restore the complete pre-upgrade backup.

The portable ZIP contains no user database, browser profile, license, subscription address, or production configuration. See `UPGRADE-2.0.0-PERSONAL.zh-CN.md` for the complete Chinese upgrade guide.

## Verification

`go mod verify`, `go test ./... -count=1`, TypeScript type checking, Vite production build, Wails Windows/amd64 build, and `git diff --check` passed before release.
