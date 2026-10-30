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
| ask | One poster asked, the question, the answer (waiting, yes, no or withdrawn) | The AI, when the picker ticks a poster or picks an offer | The picker, the poster asked, the coordinator |
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

Read one request (HS312): `market_get_request` takes the request id and returns it. The picker who made it reads it, and so does a poster owner who was asked about it (he has an ask on it); anyone else gets not found, the same as an unknown id. Opening a request on a fresh page no longer depends on having just created it.

Product on an ask (HS313): an ask carries an optional `product` beside its question, set by `market_ask` and returned by `market_list_asks` and `market_answer_ask`. A poster matches the ask to a stock line by this field instead of guessing from the words of the question.

Offer or price on a yes (HS314): `market_answer_ask` takes an optional `offer_id` and `price` with a yes. The offer must be one of the answering poster's own (otherwise invalid), the price must not be negative, and a no carries neither. The yes keeps both, so `market_list_yeses` returns them and the buyer's Found for you row has a price and an offer to Buy now from. A yes without them is unchanged. In Go the old `AnswerAsk` stays and calls the new `AnswerAskWithOffer`; the tool is bound to the new one.

Group order trail and pages (HS316, decision 98): `market_group_order_trail` takes a group order and returns who did what to it, oldest first, each step with the actor's id and `actor_name` when the host supplies names. The group order's steps are written to the same trail records as an order's, keyed by the group order's id: started, joined, changed the quantity, asked for new terms, answered yes or no, promised a date. The coordinator, the poster's owner and a coordinator of the agency see every step; a picker with a line sees her own steps and the coordinator's and poster owner's, never another picker's; anyone else gets not found. `market_order_trail` also carries `actor_name` now. `market_list_group_orders` takes `limit` (absent or zero is 50, at most 200) and `after`, the id of the last group order of the page before; the list is ordered by id, so a page that holds fewer than `limit` is the last. Each group order is still shown as the caller may see it. The Go `ListGroupOrders` takes `limit` and `after` too.

Minutes on a yes (HS317): `market_answer_ask` takes an optional `minutes`, a whole number of minutes away, with a yes. It must not be negative and a no carries none. The yes keeps it, so `market_list_yeses` returns `minutes` to the asker and to the answering poster, and the ride thread no longer reads them out of the note. The tool is bound to `AnswerAskWithMinutes`; `AnswerAskWithOffer` and `AnswerAsk` stay and call it with zero. A poster read already carries `name`, `owner_name` and `approved_by_name` (HS270), so no further field was added for the Inbox and the cards: the display name of a poster is `name`, the display name of its owner is `owner_name`.

Names on yeses and requests (HS270, last part): `market_list_yeses` and `market_get_request` now carry `picker_name`, the same host-supplied name as on asks, orders and group order lines, empty when the host supplies none. What HS270 asked for was already in place before this change: `yes_id` and `promised_date` on an order set by `market_buy_now`, `picker_name` on asks, orders and lines, and `owner_name` and `approved_by_name` on posters. No `market_name_of` tool was added; the names ride on the reads.

Withdraw an ask (HS320): `market_withdraw_ask` takes an ask id and lets the picker who made it take it back while it is still waiting, which is the Cancel on a waiting ride in journey 17 step 2 and on a price ask. The ask's answer becomes `withdrawn`, and both the picker and the poster's owner see it so in `market_list_asks`. Once the poster has answered yes or no, or the ask is already withdrawn, the tool is refused as a conflict (409) and the ask is unchanged; anyone but the picker, the poster included, gets not found, the same as an unknown id. A withdrawn ask cannot be answered: `market_answer_ask` on it is the same conflict, and no yes is created. Asks keep no trail of their own, so nothing is written to the trail records; the `withdrawn` answer on the ask is the record. The Go method is `WithdrawAsk`.

Move an offer under another product (HS415): `market_regroup_offer` takes `offer_id` and `product` and changes the product the offer is grouped under, which is how the Products page moves an offer. Only `product` changes; the words, unit, prices and posted time stay. An unknown offer is not found (404), an empty or blank product is invalid (400), and a move onto the product the offer already has is refused as a conflict (409) that says `the offer is already under <product>`. The poster's ownership is not checked: like approving a poster, the host gates who may through the action `market.offer.regroup`, and a caller must be given. The Go method is `RegroupOffer`. No list-products tool was added: the product list is what `market_list_offers` already carries in `product`.

Categories and the kinds a member is offered (HS279, decision 81): the opening of Buy inputs offers a few prompts instead of the whole list, so the market now knows a kind of product above the product. An offer carries a free-text `category` beside the `product` the AI grouped it under; the AI sets both when it posts the offer, and an offer grouped under a product that already has a category takes that category when none is given. `market_categorise_product` takes `product` and `category` and puts every offer under that product into the category, which is how the coordinator corrects a wrong one the way `market_regroup_offer` corrects a wrong product; an empty product or category is invalid, a product no offer is under is not found, and a product already under that category is a conflict. Moving an offer under another product with `market_regroup_offer` gives it that product's category when the product has one, and keeps its own otherwise. `market_list_kinds` takes an optional `limit` (absent or zero is 3, at most 10) and returns the categories members ask for most, each with `category`, `asked`, `offers` and `product`, the product asked for most inside it. A product asked about counts once per ask that names it, or names one of its offers, and once per order placed on it; the categories are ranked by that count, then by how many offers they hold, then by name, so a market nobody has asked yet still offers its fullest categories and an offer with no category is never a kind. The app turns a kind into a prompt in words; the market returns no sentence. Who names the categories is not settled by the owner: this builds them as words the AI proposes and the coordinator corrects, and a fixed list kept by the coordinator would replace the free text without changing the tools. The host mounting the module needs `market_categorise_product` and `market_list_kinds` exposed as tools.
