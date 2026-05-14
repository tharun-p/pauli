package execution

import (
	"fmt"
	"math/big"
	"strings"
)

func hexToBigInt(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0x" {
		return big.NewInt(0), nil
	}
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return nil, fmt.Errorf("expected 0x-prefixed hex, got %q", s)
	}
	n := new(big.Int)
	_, ok := n.SetString(strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X"), 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex integer: %q", s)
	}
	return n, nil
}
