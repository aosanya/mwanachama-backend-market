# The market as a declared domain (HS210)

The records in [market.md](market.md) are eleven roles in `market.blueprint.json`: poster, offer, request, ask, notice, yes, order, group_order, line, stock and trail. There are no row structs and no `AutoMigrate`; `market.wakala.json` is the default domain a host mounts through `SpecFor(instance)` and `SpecForMount(instance, mount)`, and `spec/examples/` holds `inputs` and `clinic`, two domains far enough apart that a domain word in the module would show.

## Who is acting

Every manager method that acts takes the caller as its first argument after the context. Over HTTP and MCP the caller is bound with `from: caller` and a body that names an owner, picker or actor is overwritten. A poster is owned by whoever applied; an order's picker, a line's picker and a group order's coordinator are the caller who made them.

## Visibility, one rule per test

| Rule | Where | Test |
|---|---|---|
| A stranger cannot read an order; only its picker and poster can | `visibleOrder` | `TestOnlyThePosterAnswersHisAskAndMarksPaid` |
| Only the poster marks an order paid | `MarkPaid` | same |
| A stock list is read and written by its poster only | `GetStock`, `SetStock` | `TestStockIsPrivateToItsPoster` |
| A picker sees her own line plus total, minimum and count; the poster and the coordinator see every line; no contact is stored here | `view` | `TestAGroupOrderNeedsAMinimum...` |
| A group order needs a minimum | `StartGroupOrder` | same |
| Only the poster asks for a revision | `AskRevision` | `TestARevisionKeepsThePreviousTerms...` |
| An ask is made once per poster per request | `CreateAsk` | `TestAnAskIsMadeOncePerPosterPerRequest` |

Each was shown to fail by breaking its rule in turn.

## Things that are kept

An order keeps the unit price it was tapped at and the workflow version it started with (`TestAnOrderKeepsThePriceItStartedWith...`). A revision pushes the previous price and minimum onto `terms_history` and raises `terms_version`; each line keeps the version it last answered (`answered_version`). A yes is a record and a reply never opens an order. Nothing runs on a timer: silence stays waiting. Every order action writes a trail row.

## Routes

`market.operations.json` declares 42 addresses with actions `market.<object>.<verb>` and the tool names the host's agents call. `routes.AnonymousActions` is empty.
