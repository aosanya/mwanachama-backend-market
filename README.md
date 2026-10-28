# mwanachama-backend-market

A Mwanachama module for markets with an AI in the middle. People who supply something post a price, people who want it pick among the posts, and an AI asks only what is not already posted. It names no domain: the same objects serve a farmers' group buying inputs and selling a harvest, riders posting a fare, and any agency that matches people who post with people who pick.

The module declares eleven objects: `poster` (a supplier, a buyer or a rider), `offer`, `request`, `ask`, `notice`, `yes`, `order`, `group_order`, `line`, `stock_list` and `trail`. [documentation/2. design/market.md](documentation/2.%20design/market.md) says what each holds, who writes it and who sees it.

Its rules are steps that wait for a named person's yes, written as an agency's workflows. Nothing runs on a timer, the AI never says yes for a person, and each person sees only what the design allows.

Status: the repository is created and holds the design. Nothing is built yet.

## Verify

```
make build
make test
make vet
```

Licence: Apache-2.0.
