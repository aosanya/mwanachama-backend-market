# Accepting a quote

Accepting a quote claims the notice first. `AcceptQuote` runs `UPDATE notice SET state = sold WHERE id = ? AND state = open` and requires one row affected. The call that gets no row answers `ErrConflict` and changes nothing else. Only the call that holds the claim declines the notice's other quotes and marks its own quote accepted.

Reading the notice and checking it is open, as the gate does, is not enough: two accepts of different quotes on one notice both read it open, both decline the other's quote and both mark their own accepted, which leaves two accepted quotes on a sold notice. The claim makes the database pick one winner.

`accept_quote_race_test.go` accepts eight quotes on one notice at once through two managers over one database, five rounds, and asserts one success, every other call answering `ErrConflict`, one accepted quote, the rest declined and the notice sold. It runs on SQLite by default and on Postgres when `MARKET_TEST_POSTGRES_DSN` is set.

The hub copy of the module (`projects/hub-steward/code/wakala-api/modules/market/quotes.go`) needs the same change until ISS-105 deletes it (hub issue MUN330).
