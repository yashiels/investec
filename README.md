# investec — the Investec SA Private Bank API from your terminal

A read-only CLI over the Investec South Africa Private Banking Account Information API. Accounts, balances, transactions, beneficiaries, profiles, and statement/tax-certificate documents — human tables in a terminal, clean JSON in a pipe.

- **Headless-first** — `--json` emits the full native API envelope (`data`, `links`, `meta`); non-TTY output defaults to JSON so scripts and agents just work.
- **Safe by design** — read-only; no payments or transfers. Secrets never travel through flags, tokens cache with `0600` perms partitioned per environment + credential.
- **Real exit codes** — auth, not-found, rate-limit, and upstream failures each map to a distinct code so automation can branch on them.

## Install

```bash
brew install yashiels/tap/investec  # auto-taps yashiels/tap
```

Direct downloads from the [latest GitHub release](https://github.com/yashiels/investec/releases/latest).

Build from source:

```bash
git clone https://github.com/yashiels/investec.git
cd investec
make build
```

## Setup

Get an API key + client credentials from the [Investec Developer Portal](https://developer.investec.com). Provide them via environment variables:

```bash
export INVESTEC_CLIENT_ID=...
export INVESTEC_CLIENT_SECRET=...
export INVESTEC_API_KEY=...
```

...or a config file at `~/.config/investec/config.toml` (see `config.example.toml`), which can pull secrets from a manager via `credential_command`. Precedence: **environment > config file**. Use `--sandbox` to hit Investec's sandbox environment.

## Quick Start

```bash
investec accounts list
investec accounts balance <accountId>
investec accounts transactions <accountId> --from 2026-08-01 --to 2026-08-31
investec documents list <accountId> --from 2026-01-01 --to 2026-09-01 --json | jq
investec documents get <accountId> Statement 2026-08-31 -o statement.pdf
```

## Commands

| Command | Description |
|---------|-------------|
| `investec accounts list` | List accounts |
| `investec accounts balance <accountId>` | Account balance |
| `investec accounts transactions <accountId>` | Posted transactions (`--from --to --transaction-type --include-pending`) |
| `investec accounts pending <accountId>` | Pending transactions |
| `investec beneficiaries list` | List beneficiaries |
| `investec beneficiaries categories` | List beneficiary categories |
| `investec profiles list` | List profiles |
| `investec profiles accounts <profileId>` | Accounts for a profile |
| `investec profiles beneficiaries <profileId> <accountId>` | Beneficiaries for a profile account |
| `investec profiles authorisation-setup <profileId> <accountId>` | Payment authorisation setup |
| `investec documents list <accountId>` | Available documents (`--from --to` required) |
| `investec documents get <accountId> <documentType> <documentDate>` | Download a document (raw PDF, `-o`) |
| `investec auth token` | Mint and print a bearer token |
| `investec auth status` | Token endpoint, expiry, and reported scopes |
| `investec completion <shell>` | Generate a shell completion script |
| `investec --version` | Show version |

## Output & exit codes

Human tables render only on a TTY. Otherwise, or with `--json`, the full API envelope is emitted verbatim; `--plain` forces stable tab-separated lines. `--json` and `--plain` are mutually exclusive.

| Code | Meaning |
|------|---------|
| `0` | success (including empty results) |
| `1` | request rejected / unclassified (incl. HTTP 400) |
| `2` | CLI usage or local validation error |
| `3` | credential / token-mint / 401 / 403 |
| `4` | HTTP 404 |
| `5` | HTTP 429 (rate limited; `Retry-After` shown) |
| `6` | network / TLS / timeout / HTTP 5xx |
| `130` | interrupted (SIGINT) |

## Disclaimer

Not affiliated with Investec. Uses the official [Investec Developer](https://developer.investec.com) API; you are responsible for your own credentials and API-usage terms.

## Development

```bash
make build   # build ./investec with version stamped from version.env
make test    # go test ./...
make lint    # go vet + gofmt check
```

Releases are automated via GitHub Actions. Go to **Actions → Deploy: Ship**, pick `patch`, `minor`, or `major` — it bumps the version, cross-compiles standalone binaries, publishes a GitHub release, and updates the [Homebrew tap](https://github.com/yashiels/homebrew-tap).

## License

MIT — Yashiel Sookdeo
