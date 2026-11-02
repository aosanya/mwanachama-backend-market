# Workflow steps (HS211)

A rule is a step that waits for a named person's yes (decision 72). Nothing runs on a timer.

## What the agency backend carries

Investigated read-only in `mwanachama-backend-agency`. A `Workflow` has a name, description, ordinality and an opaque `EventFlows` JSON blob; a `WorkItem` has a title, description and prompt. Neither names a role or a person who says yes, and a workflow has no version of its own (only a publication has a tag and version). RACI is assigned to Objectives, not to steps. So an agency's step actors cannot be read from it today, and the agency repo is left untouched. A host reads its agency's workflows and answers the market through the seam below.

## What is built

- `DefaultWorkflows()` holds the defaults as data: Buy now (asked: picker, yes: poster, paid: poster), Cancel (cancelled: picker, closes the poster's open yes), Group order (start, sent, line_paid, decided, reopened and called_off: coordinator), Approve a poster (approved: coordinator). Who says yes is `picker`, `poster`, `coordinator` or `none`; `none` refuses everybody.
- `UseWorkflows(WorkflowSource)` lets a host supply an agency's workflow by name and version. A source returns `(workflow, found, err)`; not found falls back to the default; a workflow that is malformed, names the wrong workflow or answers a different version than asked is `ErrInvalid`.
- `UseCoordinators(func)` tells the manager who the agency's coordinators are. With none supplied nobody is a coordinator, so approving a poster and starting a group order are refused.
- A new order and a new group order are stamped with the version the source returns for Buy now and Group order. Later steps on an order ask the source for that stamped version, so an order keeps the workflow it started with when the source moves on. An agency versions its workflows as a set, so one stamp serves the order's Buy now and Cancel steps.
- Every state-changing act checks the step's named actor against the caller and answers `ErrForbidden`: approve, decline or remove a poster, start, send or call off a group order, decide for a silent line, mark a group order line paid, the poster's yes, mark paid, cancel. A group order's coordinator is the one who started it.
- Cancel defaults to at once by the picker and closes the picker's open yeses at that poster. A source that replaces Cancel names its own actor and decides whether the yes closes.

## Bidding

A fifth default, Bidding (quoted: poster, drafted: agent, countered: picker, accepted: picker), gates the notice-and-quote journey. See [bidding.md](bidding.md).

## Group order steps (HS268)

`SendGroupOrder` moves an open group order to `sent` (step `sent`; a revising, sent, closed or cancelled order answers `ErrConflict`). After it nobody joins, revises or answers. `MarkLinePaid` marks one line of a sent group order paid (step `line_paid`; a line of another order is `ErrNotFound`, an order not yet sent is `ErrConflict`). Both default to the coordinator who started the order, ask the order's stamped workflow version, and add a trail entry. A picker still sees only her own line (with its paid mark) plus the total, minimum and count.

## Silent lines and calling off (HS268)

`DecideSilentLine` (step `decided`) settles one line whose picker has not answered the current terms, the coordinator's job in decision 70. `keep` makes the line a yes at the current terms version, `take_off` makes it a no; both set `decided_by_coordinator`, so a line the picker answered herself differs from one the coordinator settled. A taken-off line leaves the total and the count, a kept one stays in them. A line that already answered, a line of another order and an order that is sent, closed or called off are refused (`ErrConflict`, `ErrNotFound`, `ErrConflict`); any other decision word is `ErrInvalid`. The tool is `market_decide_silent_line` with `groupOrderID`, `lineID` and `decision`; the route is `POST /group-orders/{groupOrderID}/lines/{lineID}/decision`.

`CallOffGroupOrder` (step `called_off`) moves an open, revising or sent group order to `cancelled` (decision 92: the coordinator calls the order off at any step; decision 69 and the market page: at any step). A closed or already cancelled order answers `ErrConflict`; after it nobody joins, answers or pays. The tool is `market_call_off` with `groupOrderID`; the route is `POST /group-orders/{groupOrderID}/call-off`. Open yeses at the poster are not touched; the owner has not said whether a called-off group order closes them.

Both steps are new in the default Group order workflow. A replacement workflow from `UseWorkflows` that omits them answers `ErrInvalid` on these two acts, as it does for `sent` and `line_paid`.

## Taking a revised order back to open (HS706)

`ReopenGroupOrder` (step `reopened`) moves a revising group order back to `open` so it can be sent (decisions 144 and 145: an exception to a workflow is a step of that workflow, said by a named person; nothing runs on a timer, so the farmers' yeses and the total alone do not reopen it). The coordinator who started the order says it by default. An order in any other state answers `ErrConflict`, a caller the step does not name answers `ErrForbidden`, and the terms version, the lines' answers and the total are left as they are. It adds a trail entry, "took the revised order back to open". The tool is `market_reopen` with `groupOrderID`; the route is `POST /group-orders/{groupOrderID}/reopen`. A replacement workflow from `UseWorkflows` that omits the step answers `ErrInvalid` on this act, as it does for `sent`.

The default Group order workflow has no Paid step (decisions 89 and 90): who has paid is a row in the agency's Payments record. Buy now keeps its own `paid` step.

## Not built

- The agency-side who-says-yes in a workflow step: an agency's step actors cannot be read from the agency backend yet, and the step's role would have to map onto `picker`, `poster`, `coordinator`, `agent` or `none` (decision 94 lets a step name any role or group of the agency). Open with the owner (Q72, who holds a role).
- The per-line `Paid` mark and the `line_paid` step are still here; decision 90 retires the tick.
