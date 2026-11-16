# Accepting a quote

Accepting a quote claims the notice first. `AcceptQuote` runs `UPDATE notice SET state = sold WHERE id = ? AND state = open` and requires one row affected. The call that gets no row answers `ErrConflict` and changes nothing else. Only the call that holds the claim declines the notice's other quotes and marks its own quote accepted.

Reading the notice and checking it is open, as the gate does, is not enough: two accepts of different quotes on one notice both read it open, both decline the other's quote and both mark their own accepted, which leaves two accepted quotes on a sold notice. The claim makes the database pick one winner.

`accept_quote_race_test.go` accepts eight quotes on one notice at once through two managers over one database, five rounds, and asserts one success, every other call answering `ErrConflict`, one accepted quote, the rest declined and the notice sold. It runs on SQLite by default and on Postgres when `MARKET_TEST_POSTGRES_DSN` is set.

## One transaction

The claim, the decline of the notice's other quotes and the winner's accepted write run in one transaction, `marketManager.inTransaction`, which is `specstore.Store.InTransaction` over a copy of the manager. If any of the three fails, all three roll back: the notice stays open and every quote keeps its state, so the accept can be retried. The winner's write is `UPDATE quote ... WHERE id = ? AND state = open` and requires one row; a quote that is no longer open at that write (countered or declined since it was read) answers `ErrConflict` and the claim rolls back.

`accept_quote_atomic_test.go` fails the decline, then the winner's accepted write, with a gorm update callback and asserts the notice is open and every quote open; a third test flips the winning quote to countered between the read and the claim and asserts `ErrConflict` with the notice still open.

## Withdrawing a notice

`WithdrawNotice` runs `UPDATE notice SET state = withdrawn WHERE id = ? AND state = open` through `moveOpenNotice`, the same statement shape as the claim, and requires one row affected. A sold or already withdrawn notice answers `ErrConflict` and stays as it was, so a sold notice cannot be withdrawn and an accept and a withdraw at once leave one winner. A caller who is not the notice's picker still answers `ErrNotFound`.

`withdraw_notice_test.go` withdraws a sold notice, withdraws a notice twice, and runs an accept and a withdraw at once through two managers for ten rounds, asserting one success and a notice state that matches the winner.

The hub copy of the module (`projects/hub-steward/code/wakala-api/modules/market/quotes.go`) needs the same change until ISS-105 deletes it (hub issues MUN330, M7 and M8).
