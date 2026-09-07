# Environments

The repository recognizes `development`, `test` and `production`. The tracked examples contain
only non-secret process settings. Credentials, session material, encryption keys and financial
data must come from an external secret store or the operator environment and must never be copied
into these files.

`make check` uses the test environment. Product runtime configuration is introduced with the
component that owns it.
