# Hub copy diff

`mwanachama-backend-market` is the custodian of the market module. The hub (`projects/hub-steward/code/wakala-api/modules/market`) holds a copy made on 2026-11-04 and changed it afterwards. Every change made in that copy exists here, so the two differ by the import path only (hub issue M5, the market part of ISS-105).

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-market]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/market`, relative to this repository's parent folder. The exit code is 1 when the trees differ, 0 when they do not. The script and `scripts/hubdiff` are the ones of `mwanachama-backend-git` (G78), with the module path changed.

## What it does

1. Copies the `*.go` and `*.json` files of this repository and of the hub folder into two temporary folders, leaving out `cmd/`, `documentation/`, `scripts/` and `.git/`.
2. Rewrites the hub's imports: `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/` and `wakalaserver/modules/market` becomes `github.com/aosanya/mwanachama-backend-market`.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub`.

## Result of 2026-11-16

No file differs and nothing exists only in the hub. What remains is `Only in backend` lines: the test files written here (the hub copy carried only seven test files, all of which now exist here and are identical) and `spec/examples` (the two example blueprints used by `examples_test.go`).

## The hub commits carried over

One commit here per hub commit group, each naming the hub hash:

| Hub commit | What it changed |
|---|---|
| 5762d51f | The poster list drops `approved_by` for anyone who does not decide posters. |
| e70c5601 | A picker has one line in a group order: unique `group_picker` index, a losing join updates the winning line. |
| e8418b36 | A poster has one quote on a notice: unique `notice_poster` index, the losing send answers a conflict. |
| 066de655 | `cancel_order` and `mark_paid` each update only their own column while the order is still placed. |
| 9d70341b, 8debe20f | A group order carries `closed_at` and `cancelled_at`. 8debe20f only restored lines that 0904c63a had undone. |
| 7f2cb731, 0904c63a | `market_list_orders` takes `state`, `limit` and `after`; an order carries `delivered_at` and `cancelled_at`. 0904c63a's own change was the undoing of the group order lines, so it adds nothing here. |
| ca52e62a, ab71d9c8 | Test-only: coverage of posters, offers, stock, requests, asks, yeses and notices. |

The existing tests that called `ListOrders(ctx, caller)` now pass the paging arguments.
