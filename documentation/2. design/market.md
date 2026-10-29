# The market

A market puts the people who post a price in touch with the people who pick one, with an AI between them that finds out only what is not already posted.

## The pieces

| | Inputs | Harvest | Rides |
|---|---|---|---|
| Poster | A supplier | A buyer | A rider |
| Picker | A farmer | A farmer | A customer |
| Offer | The shop's price for a product, with bulk prices | The price a buyer pays | A fare |
| Ask | "Can you fill 2 bags by Friday?" | "Will you take 40 bags?" | "Can you take me from the market to Ndori?" |
| Group order | Many pickers buying from one poster | Many pickers selling to one buyer | Not needed yet |
| Coordinator | Starts and sends group orders, approves posters | The same | The same |

## The objects

| Object | What it holds | Written by | Seen by |
|---|---|---|---|
| poster | Name, area, what he posts, who approved him | The AI drafts, the coordinator approves | Everyone in the agency |
| offer | The poster's words for the product and unit, the price, bulk prices (a minimum and a price each), the product the AI grouped it under | The poster | Everyone; the latest is live |
| request | What the picker wants in her words, the amount, needed by | The picker, with the AI | The picker, the coordinator |
| ask | One poster asked, the question, the answer (waiting, yes or no) | The AI, when the picker ticks a poster or picks an offer | The picker, the poster asked, the coordinator |
| notice | What is wanted, how much, needed by | The AI, for the picker | Every poster; stays until the picker takes it down |
| yes | What, how many, for whom, the date and a note | The poster | The poster, the picker, the coordinator; open until delivered or called off |
| order | Picker, poster, product, amount, the price as it was when she tapped, the workflow version, the state, the paid mark | The AI, from her tap and his yes | The picker, the poster, the coordinator |
| group_order | Poster, product, terms (price and minimum) and every revised version, the workflow version, the state | The coordinator starts it; the AI keeps it | The coordinator sees all; the poster sees names and amounts and no contact; a picker sees her own line, the total, the minimum and the count |
| line | One picker's amount in a group order, her answer to the current terms, the coordinator's decision for a silent one, her paid mark | The picker, the coordinator | As the group order |
| stock_list | A poster's own count for each product | The poster | The poster only |
| trail | Who did what and when, on every order | The module | The picker for her own, the coordinator for all |

An order and a group order keep their own copy of the terms they started with. A poster who wants different terms asks the pickers to revise, and each answer is kept against the version it was given to. A yes is kept as a record, and the module never works out a poster's stock.

## The steps

Buy now: the picker taps and the AI asks the poster she picked, once; he says yes with a date, or no; on a no she picks another; with no answer nothing happens; he marks it paid. Cancel is at once until an agency adds a cancel workflow.

Group order: the coordinator starts it from a posted bulk price and sets the minimum, which is required; pickers join; the poster sees the order and may ask the pickers to revise, each answers yes or no, and the coordinator decides each one who stays silent; the coordinator says it is ready, locked below the minimum unless the poster agrees to a smaller amount; the AI asks the poster to fill it and he says yes with a date, or no; each picker is told her share, price and date; the coordinator marks each picker paid and can call the order off at any step.

Find it: the picker says what she wants and chooses to ask posters or put up a notice; only the posters she ticks get one message each; each reply is a row found for her.

Approve a poster: he applies, the AI drafts the request, the coordinator says yes or no.

## The AI

It asks, tells, drafts and records. It never says yes for a person, sends nothing when a posted price answers, asks only the poster a picker picked or ticked and only once, and does not nudge or fall back when someone is silent.

## Yes on the order, and names on reads (HS270)

An order carries `yes_id` and `promised_date`: `market_buy_now` takes an optional `yes_id`, which must be the caller's own open yes at the same poster, and the order keeps that id and the yes's date. The module stores no names and knows no people. A host that wants a picker shown by name calls `UseNames` with a `NameOf` function; asks, orders and group order lines then come back with a derived `picker_name`, and posters with `owner_name` and `approved_by_name` (never columns), all empty when no function is supplied or the id is unknown.

## A manager for one test run (HS280)

`ForRun(runID)` returns a manager that shares the parent's hooks (coordinators, agents, names, workflow source and clock) and reaches every table through a store scoped to the run: it reads and writes the `_test` copy of each table, stamps `run_id` on what it inserts, and reads only rows with that run id, so two runs using the same key never meet and no row is written to a live table. The parent is not changed. An empty run id is refused. The hooks are copied when `ForRun` is called, so a host sets them first. The host removes a run's rows with the shared engine's `RemoveRun`.
## What the buy and supply inputs needed (HS311 to HS314)

Group order promise (HS311): a group order carries `promised_date` and `note`, both empty until the shop speaks. `market_promise_group_order` takes `date` (required) and an optional `note`, is the poster owner's alone (anyone else gets forbidden), and saying it again replaces the first. Every reader of the group order (`market_get_group_order`, `market_list_group_orders`) sees both fields beside the terms; a picker's own order is still found through her line and the group order, not by a guess.
