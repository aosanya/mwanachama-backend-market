# Hub import drift

Before the hub deletes its in-tree copy of the market module (`projects/hub-steward/code/wakala-api/modules/market`) and imports this repository instead, the two are compared once more. Compared on 2026-11-19 with [`scripts/hub-copy-diff.sh`](../../scripts/hub-copy-diff.sh) (see [hub-copy-diff.md](hub-copy-diff.md)), which rewrites the hub's imports to this module's paths, drops comments and compares what is left. The hub folder was read, not changed.

The hub copy's last commit is 2057f14c (2026-11-17). This repository is now the union of both sides: everything the hub copy has exists here, and the hub copy lacks some of what exists here.

## Only in the hub

| File or behaviour | Disposition |
|---|---|
| `kinds_cover_test.go`: pins `CategoriseProduct` (anonymous and blank refusals, unknown product, every offer of the product moved, repeat answers a conflict, partial move) and `ListKinds` (negative limit, empty without categories, ranking by asked then offers then name, default limit of three, cap of ten, an ask counted through its offer) | Moved here unchanged apart from the import path. Hub commit 2057f14c. Green. |
| `validate_cover_test.go`: pins `Check`, `checkField`, `checkValue` and `isName` (unknown role, struct and pointer, blank required field, undeclared enum value, undeclared column, named pattern, unknown pattern is a module error) | Moved here unchanged apart from the import path. Hub commit 2057f14c. Green. |

No other file exists only in the hub. The hub copy holds no file that is not `*.go` or `*.json`, no `cmd/`, and no `documentation/`.

## Only in this repository

The hub copy predates these. They stay; the hub receives them when it imports this module.

| File or behaviour | Disposition |
|---|---|
| `GetAsk`, `GetNotice`, `GetQuote`, `GetDraft` on `MarketManager`, with the operations `get_ask`, `get_notice`, `get_quote`, `get_draft` (tools `market_get_ask`, `market_get_notice`, `market_get_quote`, `market_get_draft`; actions `market.ask.read`, `market.notice.read`, `market.quote.read`, `market.draft.read`) | Keep. Four more tools and four new permission actions that the hub's catalogue does not list yet; see the wiring table below. |
| `redactNotice`, shared by `ListNotices` and `GetNotice` | Keep. Same masking as the hub's list. |
| `WithdrawNotice` moves a notice to withdrawn only while it is open (`moveOpenNotice`) and answers a conflict otherwise | Keep. The hub's version overwrites the whole notice. |
| `SendQuote`, `DraftCounter`, `SendCounter`, `ReviseQuote` and `AcceptQuote` write inside a transaction that holds the notice open and guards the quote or draft by the state it read (`inTransaction`, `holdOpenNotice`, `claimNotice`, `holdWhileIn`, `updateWhileIn`) | Keep. The hub's versions are plain inserts and updates. |
| `SendQuote` answers a conflict when the unique index rejects a second quote from one poster | Keep. The hub's version also maps the unique violation; the difference is only that here it runs inside the transaction. |
| 40 test files at the top level and 15 in `routes/` the hub copy never carried (accept-quote atomicity and race, bidding, group race, quote writers guarded, withdraw, workflows, vocabulary, example domains, Postgres integration behind the `integration` tag, and the per-feature files) | Keep. |
| `spec/examples`: two example blueprints used by `examples_test.go` | Keep. |
| `cmd/ddl`, `scripts/`, `documentation/` | Keep. The hub copy has none of them. |

## Differing beyond the import path

Four files differ, and each difference is a hub copy behind this repository, not two behaviours that disagree about the same operation.

| File | Difference | Disposition |
|---|---|---|
| `asks.go` | This repository adds `GetAsk` and routes `WithdrawNotice` through `moveOpenNotice`; `ListNotices` uses `redactNotice` and `GetNotice` is added. | Keep this repository's. No conflict. |
| `manager.go` | This repository adds the four `Get*` methods to the interface and the helpers `updateWhileIn`, `holdWhileIn`, `inTransaction`. | Keep this repository's. No conflict. |
| `quotes.go` | The quote writers run in a transaction and `GetQuote` and `GetDraft` are added. | Keep this repository's. No conflict. |
| `market.operations.json` | Four operations added (`get_ask`, `get_notice`, `get_quote`, `get_draft`). Every operation the hub declares is declared here, unchanged. | Keep this repository's. No conflict. |

## Conflicts

None. No operation behaves differently on the two sides in a way that makes one side's behaviour a choice; the stricter transactional behaviour here is what the hub's own commit notes say the hub copy was waiting for.

## Pinned counts

Nothing in this repository pins a count that the merge changed: only two test files were added and no operation was added or removed. The hub pins counts that will change when it imports this module (see below).

## Hub wiring that is not part of the module

Not moved. The hub keeps all of it; the import step changes only the module's import path and the items marked as changing.

| Hub file | What it does | Changes at import |
|---|---|---|
| `internal/api/http/market_routes.go`, `market_mount_names.go`, `market_reserved_mount_keys.go`, `operation_routes.go` | Mounts the market routes per agency and instance, resolves a mount name to its key, reserves route segments | Import path only. |
| `internal/api/http/mcp_tools_market.go` | Builds the MCP tools from `marketroutes.BuildTools` and filters them per caller | Import path; the four new tools appear in the shape. |
| `internal/api/http/market_announced.go`, `market_announced_calledoff.go`, `market_names.go`, `market_ask_run.go`, `approvals_apply_market.go`, `approvals_apply.go`, `approvals_apply_groupgate.go`, `record_row_runs.go` | Wrappers that embed `market.MarketManager` and add announcements, names, ask runs and approvals | Import path only. Every wrapper embeds the interface, so the four added methods pass through. |
| `internal/testruns/stage.go`, `stage_ask.go`, `market_module.go`, `cmd/server/testruns.go` | Test-run staging over a market manager | Import path only. |
| `internal/agencies/registry.go`, `caches.go`, `test_copy.go`, `ask_run.go`, `ask_closed.go`, `internal/accounts/router.go` | Per-instance market manager registry and caches | Import path only. |
| `modules/agency/models/permissions_catalog.json` | Permission codes and the actions each carries (`market.browse`, `market.notice`, `market.ask`, `market.ask_answer`, `market.advise`, and others) | Lists none of `market.ask.read`, `market.notice.read`, `market.quote.read`, `market.draft.read`; each needs a place in a permission, or the four tools are unreachable to anyone. |
| `internal/api/http/testdata/mcp_roster/no-mounts.txt` and other tool rosters | A pinned list of tool names, 51 of them market tools | Gains the four `market_get_*` names, so the pinned market count moves from 51 to 55. |
| Hub tests under `internal/api/http` named `market_*_test.go`, `misuse_*_test.go`, `zz_fakes_cover_test.go`, and `internal/agencies/ask_*_test.go` | Exercise the wiring | Import path only; run them after the import. |
