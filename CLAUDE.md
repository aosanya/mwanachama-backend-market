# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-market

A domain-agnostic module for markets with an AI in the middle: posters post offers, people who want something pick among them, an AI asks one named poster only what is not already posted, and orders and group orders move through steps that wait for a named person's yes. Module path `github.com/aosanya/mwanachama-backend-market`.

It is in the same family as `mwanachama-backend-taskmanager`, `mwanachama-backend-riskmanager` and `mwanachama-backend-assetmanager`: a tool a host mounts per agency, not part of any product. It names no crop, no ride and no shop; the host's domain spec says what the objects are called.

## Objects are declared, not written

The tables come from a JSON blueprint, not from Go row structs, through `mwanachama-backend-shared/spec`, `specstore` and `dispatch`, as the other declared-domain modules do. Declare the objects in `market.blueprint.json` and the routes in `market.operations.json`.

## Rules the design binds

- A rule is a step that waits for a named person's yes. Nothing runs on a timer.
- The AI never says yes for a person.
- An order keeps the terms it started with, and the version of the workflow it started with.
- A supplier sees a group order's names and amounts and never a contact; a person sees her own line, the total, the minimum and the count, never another person's name or amount; a supplier's private stock list is his alone.

## Documentation

`documentation/` holds only `1. requirements`, `2. design`, `3. implementation` and `4. qa`. Anything that needs explaining goes there, never inline.

## Code

Write no comments of any kind in code. A name, a type or a smaller function carries the meaning.

## Verify

`make build`, `make test`, `make vet`. Tests run in memory; a test that needs Postgres is behind the `integration` tag.

## Status (2026-10-28, HS210)

Built: `market.blueprint.json` (eleven roles), `models/`, `MarketManager` over `specstore` with caller-bound visibility, `market.operations.json` and `routes/` over `dispatch`, `market.wakala.json` with `SpecFor`/`SpecForMount`, two example domains, `cmd/ddl`. Bidding (quotes, agent drafts, the `agent` actor) built 2026-10-29, see [documentation/2. design/bidding.md](documentation/2.%20design/bidding.md). Open: M4 (workflow gates) and mounting in wakala-api. See [documentation/2. design/declared-domain.md](documentation/2.%20design/declared-domain.md).
