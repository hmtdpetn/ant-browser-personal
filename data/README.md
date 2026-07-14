# Runtime data directory

This directory is intentionally empty in the source tree and portable release package.

The application creates its runtime database and other local state here when it first runs. Do not copy another user's data into a release package. When upgrading an existing installation, keep your existing `data/` directory instead of replacing it with the empty directory from a new portable package.
