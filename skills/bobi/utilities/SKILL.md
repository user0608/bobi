---
name: utilities
description: Use Bobi tools, pointers, lists, and text matching helpers when the project already depends on those small utility packages. Trigger when using github.com/user0608/bobi/tools or asking for Bobi utility functions.
---

# Bobi Utilities

Inspect the package API before using a helper because these packages are intentionally small and independent:

- `tools/pointers`: pointer/value convenience helpers.
- `tools/list`: list operations.
- `tools/textmatch`: text matching helpers for confirmations and statuses.

Prefer the standard library when the operation is trivial or the utility would add indirection. Do not invent imports from a package that is not present in the selected Bobi version; verify with `go doc github.com/user0608/bobi/tools/...` or the local module cache.
