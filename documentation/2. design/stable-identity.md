# A thing's stable identity

`MarketManager` embeds the shared `identity.Owner` (shared S56), so a host that resolves a manager per instance and mount already holds the owner for it.

## The key

Nothing in the market is re-minted: a poster, offer, request, ask, notice, quote, draft, yes, order, group order, line, stock row or trail row keeps its row id for as long as it exists, and an edit, a regroup or a revision does not touch it. So the stable key is the row id. The raw is

`market/<instance>/<mount>/<object>/<id>`

for example `market/agy1f2e3d4c/inputs/poster/3f0c...`. The instance is the spec's instance. The mount and the object ride in the identity because a uuid does not say which of the thirteen tables it is in, and because an agency can run more than one market (HS272, decision 80): a raw minted by Inputs is `identity.ErrNotOwned` in Harvest, not a confirmed miss that would mark a live thing stale. A host that renames a mount renames every raw with it, which is the same break as renaming the mount's tables.

`StableIdentity(ctx, object, id)` answers the raw for the thirteen roles and `ErrNoStableIdentity` for anything else. `Resolve(ctx, raw)` answers the `(object, id)` back:

- `identity.ErrInvalidRaw` for a string that does not parse;
- `identity.ErrNotOwned` for another module's raw, another instance's or another mount's;
- `identity.ErrNotFound` for an identity that is not `<mount>/<object>/<id>`, an object the market lacks, or a row that is gone.

Neither method takes a caller, so the market's visibility rules do not apply to them. They only confirm that a row exists under an id the caller already named, and read nothing from it; a private order or a stock list is no more exposed than by a uuid guess.

## What the operations mark

Every path id the file binds one at a time carries `refers`: `posterID` and the query filter `poster_id` (`market.poster`), `offerID` and `offer_id` (`market.offer`), `requestID`, `askID`, `noticeID`, `quoteID`, `draftID`, `orderID`, `groupOrderID` and `lineID`, plus the body field `yes_id` on `buy_now` and `offer_id` on `answer_ask`. The object after the dot is the name `StableIdentity` takes. Arguments that arrive inside a whole body or a whole query (a group order's `poster_id`, an ask's `request_id`, a quote's `notice_id`) cannot be marked until the engine can name a field of a whole request (S57 refuses a mark on one). `stock` and `trail` are never named by an argument, so no operation marks them.

Tests: `operations_refers_test.go`.

## The conversation

This module has no MCP server of its own: `routes.BuildTools` returns `dispatch.Tool`s and the host registers them. The conversation id reaches a manager only if the host calls `Tool.Handle(ctx, dispatch.Request{Arguments, Meta})` with the SDK's `req.Params.Meta`; calling `Invoke` drops it. Nothing in this repository changes for that.

Tests for identity: `identity_owner_test.go`. Tests for the two-mount separation: `hs273_test.go`.
