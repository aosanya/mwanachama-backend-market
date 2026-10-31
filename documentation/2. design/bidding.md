# Bidding: a farmer's notice, quotes, and an agent who only drafts

A farmer says what she is selling. Buyers are told, each sends a price, and an AI agent helps her bargain. The agent reads the quotes and drafts a counter. She reads it and sends it, or does not. Only she accepts a quote.

## The journey

Wanjiru posts a notice: 40 bags of maize. Every approved poster except her is told. Otieno quotes 3000 and Kamau quotes 3200; she is told of each. Her helper, an agent seat in the agency, reads both quotes and drafts a counter of 3400 to Kamau with a note. She is told a draft is waiting. She reads it and sends it; Kamau is told, and no one else. Kamau answers with a new price of 3400. She accepts it. The notice is sold and Otieno's quote is declined.

## The objects

| Object | What it holds | Written by | Seen by |
|---|---|---|---|
| quote | The notice, the poster, the price, a note, the counter price once one is sent, and where it stands: open, countered, accepted or declined | The poster | The notice's picker and the agent see all; a poster sees only his own |
| draft | The quote, the price proposed, what to say, who wrote it, waiting or sent | The agent | The notice's picker and the agent who wrote it; no poster ever |

A notice is `open`, `withdrawn` or `sold`. A notice takes quotes only while it is open.

## The Bidding workflow

| Step | Who says yes | What it does |
|---|---|---|
| quoted | the poster | Sends a quote, or revises it after a counter. One quote stands per poster on a notice. |
| drafted | the agent | Writes a counter on an open quote. It goes nowhere. |
| countered | the picker | Sends a draft. The quote becomes `countered` and carries the counter price. |
| accepted | the picker | Accepts an open quote. The notice is sold and every other quote is declined. |

`agent` is a new kind of actor beside `picker`, `poster` and `coordinator`. The host says who is one through `UseAgents`; with none supplied nobody is, so nobody drafts. In `mwanachama-wakala-api` an agent is a member of the agent role who holds an agency role, the seat an AI role is mapped to.

## What the agent cannot do

It cannot send a counter, accept a quote or send a quote. Each step names someone else, and the permission `market.advise` lets an agent seat list quotes and write drafts and nothing more. This keeps the market rule that the AI never says yes for a person.

A counter waits for the buyer's new price. The picker accepts only an open quote, so a countered quote cannot be accepted until the poster revises it.

## Told

Four events, all in the market category, each told to its subject by default and changeable by the agency: `notice_posted` to every approved poster but the picker, `quote_sent` to the picker, `draft_waiting` to the picker, `counter_sent` to the quoting poster. A draft is never announced to a poster.

## Tested

`mwanachama-wakala-api` runs the whole journey over the real routes, MCP tools, AI gateway and notifications. A scripted model stands in for the provider and answers like an agent: it asks for the quotes, picks the highest, and drafts a counter 200 above it. A small loop in the test plays the host that carries the model's tool calls to the MCP endpoint as the agent seat. See `bidding_walk_test.go` and `market_bidding_agent_test.go` in `internal/api/http`.

## Not built

- A notice and its quotes do not keep the workflow version they started with.
- No way for a picker to discard a draft; she leaves it unsent.
- A counter closes nothing when the buyer stays silent; nothing runs on a timer.
- An accepted quote does not yet make an order.
- The loop that carries the model's tool calls to the endpoint lives only in the test. No product host runs an agent yet.
