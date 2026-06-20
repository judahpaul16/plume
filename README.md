# Plume

Run one command in any project and get a readable picture of how user information
flows through it: where personal data enters, where it's stored, where it's sent,
and where it leaks.

```sh
plume                 # scan the current directory and open the graphic
plume ./service ./infra ./other-repo   # scan several paths as one graph
```

Plume is a single static binary. No setup, no config, no annotations. It scans
code and infrastructure-as-code, builds a normalized flow graph, and opens a
self-contained interactive view in your browser.

## What you get

A graph from **User → your services → stores / logs / third parties**, where each
edge is tagged with the data categories it carries (email, name, card, SSN, …)
and colored by sensitivity. A flow that carries a sensitive category into a log
sink or a third party is exactly what a privacy review looks for.

The viewer is interactive (served on a loopback port so it works everywhere):

- **Focus mode** — click a node to highlight its full upstream and downstream
  lineage and fade the rest.
- **Filter** by sensitivity, **search** nodes, pan and zoom.
- A **Sankey** toggle for flow volume.

## What it detects

- **Sources** — the user, the origin of personal data.
- **Services** — your code files that handle it.
- **Stores** — database / ORM / cache / object-store / queue writes.
- **Sinks** — logger and stdout writes.
- **External** — HTTP calls to non-local hosts, known SDKs (Stripe, Twilio,
  Segment, Sentry, …), and email / messaging sends.
- **Categories** — a built-in dictionary recognizes personal data by identifier
  name and assigns a sensitivity (PII, financial, credential, health, special).

Infrastructure-as-code (Terraform/HCL, compose, Kubernetes, Serverless) is a
first-class input: declared resources refine the generic stores, so a code-level
"Database" becomes "PostgreSQL (Amazon RDS)".

## How it works

`collectors → normalized flow graph → renderer`. Files are detected and parsed
with an embedded, pure-Go tree-sitter runtime that covers 200+ languages, in
parallel across cores. Extraction is zero-config static heuristics plus a personal
-data dictionary: it surfaces candidate flows and filters obvious placeholder
data. Files that parse too slowly are skipped so the scan stays fast (a hundred-
file repo finishes in a few seconds). Extraction is best-effort by nature; widen
the dictionary and call patterns in `internal/scan` for your stack.

## Install

Download a binary from Releases, or build from source (Go 1.21+):

```sh
go install github.com/judahpaul16/plume@latest
# or
git clone https://github.com/judahpaul16/plume && cd plume && go build -o plume .
```

The release binaries are fully static (`CGO_ENABLED=0`), one per OS/arch.

## Flags

```
plume [flags] [path ...]
  --out FILE    output HTML file (default plume.html)
  --no-open     write the report but do not serve or open a browser
  --json        print the flow graph as JSON and exit
```

## License

MIT.
