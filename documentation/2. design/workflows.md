# Workflow steps (HS211)

A rule is a step that waits for a named person's yes (decision 72). Nothing runs on a timer.

## What the agency backend carries

Investigated read-only in `mwanachama-backend-agency`. A `Workflow` has a name, description, ordinality and an opaque `EventFlows` JSON blob; a `WorkItem` has a title, description and prompt. Neither names a role or a person who says yes, and a workflow has no version of its own (only a publication has a tag and version). RACI is assigned to Objectives, not to steps. So an agency's step actors cannot be read from it today, and the agency repo is left untouched. A host reads its agency's workflows and answers the market through the seam below.

## What is built

- `DefaultWorkflows()` holds the defaults as data: Buy now (asked: picker, yes: poster, paid: poster), Cancel (cancelled: picker, closes the poster's open yes), Group order (start and paid: coordinator), Approve a poster (approved: coordinator). Who says yes is `picker`, `poster`, `coordinator` or `none`; `none` refuses everybody.
- `UseWorkflows(WorkflowSource)` lets a host supply an agency's workflow by name and version. A source returns `(workflow, found, err)`; not found falls back to the default; a workflow that is malformed, names the wrong workflow or answers a different version than asked is `ErrInvalid`.
- `UseCoordinators(func)` tells the manager who the agency's coordinators are. With none supplied nobody is a coordinator, so approving a poster and starting a group order are refused.
- A new order and a new group order are stamped with the version the source returns for Buy now and Group order. Later steps on an order ask the source for that stamped version, so an order keeps the workflow it started with when the source moves on. An agency versions its workflows as a set, so one stamp serves the order's Buy now and Cancel steps.
- Every state-changing act checks the step's named actor against the caller and answers `ErrForbidden`: approve, decline or remove a poster, start a group order, the poster's yes, mark paid, cancel. A group order's coordinator is the one who started it.
- Cancel defaults to at once by the picker and closes the picker's open yeses at that poster. A source that replaces Cancel names its own actor and decides whether the yes closes.

## Not built

The group order's open, send and per-line paid steps have no manager methods yet, so they are not gated. The group order default lists only start and paid.
