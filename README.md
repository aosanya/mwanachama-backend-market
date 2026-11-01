# mwanachama-backend-market

A Mwanachama module for markets with an AI in the middle. People who supply something post a price, people who want it pick among the posts, and an AI asks only what is not already posted. It names no domain: the same objects serve a farmers' group buying inputs and selling a harvest, riders posting a fare, and any agency that matches people who post with people who pick.

The module declares thirteen objects: `poster` (a supplier, a buyer or a rider), `offer`, `request`, `ask`, `notice`, `yes`, `order`, `group_order`, `line`, `stock`, `trail`, `quote` and `draft`. [documentation/2. design/market.md](documentation/2.%20design/market.md) says what each holds, who writes it and who sees it; [bidding.md](documentation/2.%20design/bidding.md) covers `quote` and `draft`.

Its rules are steps that wait for a named person's yes, written as an agency's workflows. Nothing runs on a timer, the AI never says yes for a person, and each person sees only what the design allows.

Status: built and mounted per agency in `mwanachama-wakala-api`. The blueprint, the manager over `specstore`, the declared route table (50 routes, each also a tool) and the default workflows are in place; what remains is on [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).

## Verify

```
make build
make test
make vet
```

Licence: Apache-2.0.
