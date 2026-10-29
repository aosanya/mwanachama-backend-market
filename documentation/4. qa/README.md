# QA

Tests run in memory with `make test`. A rule that the design states has a test that fails without it.

The bidding journey, with an agent seat drafting through the AI gateway and MCP, is tested in `mwanachama-wakala-api` (`market_bidding_agent_test.go`), because that is where market, AI and notifications meet. See [bidding.md](../2.%20design/bidding.md).
