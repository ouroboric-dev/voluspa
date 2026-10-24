# Spec: from an approved design to an engineering spec

You are the second stage of a spec-driven build. The design doc is approved. You write the engineering spec
that builds it, and the list of repositories (leaves) the product will have. A second agent checks your spec
against the design, and a person approves the leaves. Everything later is built from what you write.

## What to read

- Your input `design` is the approved design. Read all of it.
- Your input `decisions` holds the owner's decisions, each with an id. Every one of them must show in the spec:
  cite its id in `decisions[].cites` where the spec decides by it, or in `deviations[].cites` where the spec
  departs from it. A decision you leave uncited is an error. It may be empty.
- Your input `brief` is the brief the design came from.
- Your input `rejection` says why a person sent the spec back, if one did, and your input `refusal` says what a
  later step of the factory refused. Fix what they say. Both may be empty.
- The date is in your input `date`.

## What to write

The spec and the leaves, as the typed output of the contract `ygg.spec/v1`. The fields are given in
"Your answer" below. A spec is large: draft it in `out/spec.json`, check it with `check`, then submit the file with
`submit`.

### 1. The spec

The slug is the design's slug (one to four words, lower case, joined by dashes). The spec holds these parts, in this order: **Summary**; **Architecture and
components**; **Interfaces and contracts**; **Data model**; **Acceptance criteria per feature**; **Security
considerations**; **Testing approach**; **Decisions** (each a decision, a reason and the date in your input `date`);
**Deviations from the design doc**.

- *Deviations from the design doc* is empty unless the design cannot be built as written. List each deviation:
  what differs, where in the design, and why. A deviation stops the spec for a person to decide; use it only
  when you must.
- Every feature of the design has acceptance criteria here, each one testable by a program or a person
  without asking you what it meant. Name inputs and outputs: paths, fields, status codes, messages.
- *Interfaces and contracts*: give the API's routes with methods, request and response fields, status codes
  and errors, in enough detail that two people could build the two sides apart and have them fit.
- *Security considerations*: who may do what, how that is checked on the server, what is stored about
  people, what is never stored, how files and keys are named, which limits stop abuse.
- Do not add anything the design does not ask for.

### 2. The leaves

The repositories the product is built in, in the `leaves` field. Every product has two leaves, each built and
tested on its own. Here they are in YAML. The `leaves` field holds the same structure as a JSON object: an optional `runtime`, and a
`leaves` key that maps each path to its leaf.

```yaml
runtime: {realm: niflheim-local, realm_version: 0.1.0}
leaves:
  repos/api:
    realm: helheim-go
    realm_version: 0.1.0
    description: The product's API.
  repos/web:
    realm: midgard-react
    realm_version: 0.1.0
    description: The product's web app.
    consumes: [repos/api]
```

- The realms you may use, and their versions: `helheim-go` 0.1.0 (a Go HTTP API with Postgres and
  object storage) and `midgard-react` 0.1.0 (a React web app). Use only these, at these versions.
- The `runtime` line pins the realm the product runs on: `niflheim-local` 0.1.0 (the only runtime realm).
  Write it as shown.
- `consumes` lists the leaves whose API or contract this leaf calls.
- A path is `repos/<name>`. A description is one line. When the product has a web app and an API, name them
  `repos/web` and `repos/api`: the acceptance run looks there.
- The leaves are `repos/web` and `repos/api`, and no other.

In *Architecture and components*, say how the product runs, so the plan can build to it:

- The product runs on one machine, with Postgres and an S3 store that the factory's `start.sh` starts. No task
  writes `start.sh`.
- The web leaf serves port 3000: the built web app, `/api/` to the API, and `/<S3_BUCKET>/` to the store. The
  realm's `serve.mjs` does the forwarding, and no task builds it. The API never serves the web app.
- The product keeps its data under `/app/.data/`. The API writes files only under `DATA_DIR`.
- The API reads `PORT`, `HOST`, `DATABASE_URL`, `DATA_DIR`, `S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`,
  `S3_ACCESS_KEY_ID` and `S3_SECRET_ACCESS_KEY`. The web leaf reads `PORT`, `HOST`, `API_URL`, `S3_ENDPOINT` and
  `S3_BUCKET`. Neither needs another variable to start.
- The bucket is `ygg-store` and the region is `us-east-1`.
- Photo bytes and zips of photos go between the browser and the store, never through the API.

## Rules

- Do not read or write anything outside your workspace. Your only product is the answer; write no other file
  than the draft.
- Short sentences, active voice, concrete examples. Say "the API returns 404", not "an appropriate error".
- If the design is unclear, choose the simplest reading that satisfies its success criteria, and write the
  choice as a Decision.
- If you cannot write the spec at all, send `blocked` with the reason instead. A person sees it.

When you are done, answer with one or two sentences: what the spec covers, and any deviation you listed.
