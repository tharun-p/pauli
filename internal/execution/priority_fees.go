package execution

import (
	"fmt"
	"math/big"
)

// ReceiptFeeFields holds per-transaction fields needed to sum priority (tip) fees.
type ReceiptFeeFields struct {
	GasUsed           *big.Int
	EffectiveGasPrice *big.Int // post-London; use GasPrice when absent
	GasPrice          *big.Int // legacy fallback
}

// SumPriorityFeesWei returns the sum over receipts of gasUsed * max(0, effectivePrice - baseFeePerGas),
// where effectivePrice is EffectiveGasPrice if set, otherwise GasPrice. baseFeePerGas may be nil (treated as 0).
func SumPriorityFeesWei(baseFeePerGas *big.Int, receipts []ReceiptFeeFields) (*big.Int, error) {
	if baseFeePerGas == nil {
		baseFeePerGas = big.NewInt(0)
	}
	out := big.NewInt(0)
	for i := range receipts {
		r := &receipts[i]
		if r.GasUsed == nil || r.GasUsed.Sign() < 0 {
			return nil, fmt.Errorf("receipt %d: invalid gasUsed", i)
		}
		eff := r.EffectiveGasPrice
		if eff == nil {
			eff = r.GasPrice
		}
		if eff == nil {
			return nil, fmt.Errorf("receipt %d: missing effectiveGasPrice and gasPrice", i)
		}
		tip := new(big.Int).Sub(eff, baseFeePerGas)
		if tip.Sign() < 0 {
			tip = big.NewInt(0)
		}
		txTip := new(big.Int).Mul(tip, r.GasUsed)
		out.Add(out, txTip)
	}
	return out, nil
}
