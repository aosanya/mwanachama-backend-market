package market

import _ "embed"

//go:embed market.operations.json
var operationsJSON []byte

func Operations() []byte { return operationsJSON }
