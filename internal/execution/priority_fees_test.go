package execution

import (
	"math/big"
	"testing"
)

func TestSumPriorityFeesWei_London(t *testing.T) {
	base := big.NewInt(100) // per gas
	receipts := []ReceiptFeeFields{
		{GasUsed: big.NewInt(21_000), EffectiveGasPrice: big.NewInt(150), GasPrice: nil},  // tip 50 * 21k
		{GasUsed: big.NewInt(100_000), EffectiveGasPrice: big.NewInt(100), GasPrice: nil}, // tip 0
	}
	sum, err := SumPriorityFeesWei(base, receipts)
	if err != nil {
		t.Fatal(err)
	}
	// 50*21000 + 0 = 1_050_000
	want := big.NewInt(50*21_000 + 0)
	if sum.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", sum.String(), want.String())
	}
}

func TestSumPriorityFeesWei_LegacyNoBaseFee(t *testing.T) {
	receipts := []ReceiptFeeFields{
		{GasUsed: big.NewInt(21_000), EffectiveGasPrice: nil, GasPrice: big.NewInt(50)},
	}
	sum, err := SumPriorityFeesWei(nil, receipts)
	if err != nil {
		t.Fatal(err)
	}
	want := big.NewInt(50 * 21_000)
	if sum.Cmp(want) != 0 {
		t.Fatalf("got %s want %s", sum.String(), want.String())
	}
}

func TestSumPriorityFeesWei_NegativeTipClamped(t *testing.T) {
	base := big.NewInt(200)
	receipts := []ReceiptFeeFields{
		{GasUsed: big.NewInt(1000), EffectiveGasPrice: big.NewInt(100), GasPrice: nil},
	}
	sum, err := SumPriorityFeesWei(base, receipts)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Sign() != 0 {
		t.Fatalf("expected 0 got %s", sum.String())
	}
}

func TestHexToBigInt(t *testing.T) {
	n, err := hexToBigInt("0xa")
	if err != nil {
		t.Fatal(err)
	}
	if n.Int64() != 10 {
		t.Fatalf("got %v", n.Int64())
	}
}
