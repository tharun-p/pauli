// Devnet-only tool: signs two conflicting Phase0-style attestations (double vote) for the same
// validator and POSTs them to the beacon node's attestation pool so the chain can include an
// attester slashing. Run only on private Kurtosis / local devnets.
//
// BLS secret key material must be obtained from your enclave (never commit keys). Typical flow:
// export the validator key for a sacrificial index from the ethereum-package files artifact or
// validator sidecar keystore, then pass the raw 32-byte secret here as hex.
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	blsu "github.com/protolambda/bls12-381-util"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/phase0"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/bitfields"
	"github.com/protolambda/ztyp/tree"
)

func main() {
	beacon := flag.String("beacon", "http://127.0.0.1:5052", "Beacon HTTP API base URL (no trailing slash)")
	validator := flag.Uint64("validator", 0, "Validator index to slash (must match the BLS secret)")
	secretHex := flag.String("secret-key-hex", "", "32-byte BLS secret key as hex (with or without 0x). Kurtosis-only; never commit.")
	littleEndian := flag.Bool("secret-little-endian", true, "If true, reverse key bytes before interpreting as big-endian BLS scalar (common for tooling exports)")
	flag.Parse()

	if *secretHex == "" {
		fmt.Fprintln(os.Stderr, "-secret-key-hex is required")
		flag.Usage()
		os.Exit(2)
	}

	spec := configs.Mainnet
	hFn := tree.GetHashFn()
	client := &http.Client{Timeout: 30 * time.Second}

	genesis, err := fetchGenesis(client, *beacon)
	if err != nil {
		exitErr("genesis: %v", err)
	}
	genesisRoot, err := decodeRoot(genesis.Data.GenesisValidatorsRoot)
	if err != nil {
		exitErr("genesis validators root: %v", err)
	}

	skBytes, err := parseHex32(*secretHex)
	if err != nil {
		exitErr("secret key: %v", err)
	}
	if *littleEndian {
		for i, j := 0, len(skBytes)-1; i < j; i, j = i+1, j-1 {
			skBytes[i], skBytes[j] = skBytes[j], skBytes[i]
		}
	}
	var sk blsu.SecretKey
	if err := sk.Deserialize((*[32]byte)(skBytes)); err != nil {
		exitErr("BLS secret deserialize (try -secret-little-endian=false): %v", err)
	}

	headSlot, err := fetchHeadSlot(client, *beacon)
	if err != nil {
		exitErr("head slot: %v", err)
	}
	slotsPerEpoch, err := fetchUint64Spec(client, *beacon, "SLOTS_PER_EPOCH")
	if err != nil {
		exitErr("spec SLOTS_PER_EPOCH: %v", err)
	}
	epoch := headSlot / slotsPerEpoch

	duty, err := fetchAttesterDuty(client, *beacon, epoch, *validator)
	if err != nil {
		exitErr("attester duty: %v", err)
	}
	// Prefer a duty at or after head; otherwise use the first returned duty.
	slot := duty.Slot
	if slot < headSlot {
		slot = headSlot
	}

	attDataJSON, err := fetchAttestationData(client, *beacon, slot, duty.CommitteeIndex)
	if err != nil {
		exitErr("attestation data: %v", err)
	}
	data1, err := parseAttestationData(attDataJSON)
	if err != nil {
		exitErr("parse attestation data: %v", err)
	}
	data2 := data1
	// Double vote: same target epoch, same slot/index, different beacon block root.
	if data2.BeaconBlockRoot == ([32]byte{}) {
		data2.BeaconBlockRoot[0] = 1
	} else {
		data2.BeaconBlockRoot[31] ^= 0xff
	}

	fork, err := fetchForkAtSlot(client, *beacon, uint64(data1.Slot))
	if err != nil {
		exitErr("fork: %v", err)
	}
	fv, err := forkVersionForSlot(fork, uint64(data1.Slot), slotsPerEpoch)
	if err != nil {
		exitErr("fork version: %v", err)
	}

	domain := common.ComputeDomain(common.DOMAIN_BEACON_ATTESTER, fv, genesisRoot)

	aggBits, err := buildAggregationBits(duty.CommitteeLength, duty.ValidatorCommitteeIndex, uint64(spec.MAX_VALIDATORS_PER_COMMITTEE))
	if err != nil {
		exitErr("aggregation bits: %v", err)
	}

	sig1, err := signAttestationData(&sk, hFn, domain, &data1)
	if err != nil {
		exitErr("sign attestation 1: %v", err)
	}
	sig2, err := signAttestationData(&sk, hFn, domain, &data2)
	if err != nil {
		exitErr("sign attestation 2: %v", err)
	}

	att1 := phase0.Attestation{AggregationBits: aggBits, Data: data1, Signature: sig1}
	att2 := phase0.Attestation{AggregationBits: aggBits, Data: data2, Signature: sig2}

	body1, err := json.Marshal([]any{attestationToJSON(&att1)})
	if err != nil {
		exitErr("json 1: %v", err)
	}
	body2, err := json.Marshal([]any{attestationToJSON(&att2)})
	if err != nil {
		exitErr("json 2: %v", err)
	}

	fmt.Printf("submitting double vote for validator %d slot %d committee_index %d\n", *validator, data1.Slot, data1.Index)
	if err := postAttestations(client, *beacon, body1); err != nil {
		exitErr("post attestation 1: %v", err)
	}
	if err := postAttestations(client, *beacon, body2); err != nil {
		exitErr("post attestation 2: %v", err)
	}
	fmt.Println("posted two conflicting attestations; watch beacon logs / Dora for attester slashing inclusion")
}

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

type genesisResp struct {
	Data struct {
		GenesisValidatorsRoot string `json:"genesis_validators_root"`
	} `json:"data"`
}

func fetchGenesis(c *http.Client, beacon string) (*genesisResp, error) {
	var g genesisResp
	if err := getJSON(c, beacon+"/eth/v1/beacon/genesis", &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func fetchHeadSlot(c *http.Client, beacon string) (uint64, error) {
	var out struct {
		Data struct {
			Header struct {
				Message struct {
					Slot string `json:"slot"`
				} `json:"message"`
			} `json:"header"`
		} `json:"data"`
	}
	if err := getJSON(c, beacon+"/eth/v1/beacon/headers/head", &out); err != nil {
		return 0, err
	}
	return strconv.ParseUint(out.Data.Header.Message.Slot, 10, 64)
}

func fetchUint64Spec(c *http.Client, beacon, key string) (uint64, error) {
	var out struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := getJSON(c, beacon+"/eth/v1/config/spec", &out); err != nil {
		return 0, err
	}
	raw, ok := out.Data[key]
	if !ok {
		return 0, fmt.Errorf("spec missing %q", key)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	return strconv.ParseUint(s, 10, 64)
}

type attesterDuty struct {
	Slot                    uint64 `json:"-"`
	CommitteeIndex          uint64 `json:"-"`
	ValidatorCommitteeIndex uint64 `json:"-"`
	CommitteeLength         uint64 `json:"-"`
}

func fetchAttesterDuty(c *http.Client, beacon string, epoch, valIdx uint64) (*attesterDuty, error) {
	idxStr := strconv.FormatUint(valIdx, 10)
	req, err := http.NewRequest(http.MethodPost, beacon+"/eth/v1/validator/duties/attester/"+strconv.FormatUint(epoch, 10), strings.NewReader("["+idxStr+"]"))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("duties http %d: %s", resp.StatusCode, truncate(string(b), 400))
	}
	var out struct {
		Data []struct {
			Slot                    string `json:"slot"`
			CommitteeIndex          string `json:"committee_index"`
			ValidatorCommitteeIndex string `json:"validator_committee_index"`
			CommitteeLength         string `json:"committee_length"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("no attester duty for validator %d epoch %d", valIdx, epoch)
	}
	raw := out.Data[0]
	d := &attesterDuty{}
	var err2 error
	d.Slot, err2 = strconv.ParseUint(raw.Slot, 10, 64)
	if err2 != nil {
		return nil, fmt.Errorf("slot: %w", err2)
	}
	d.CommitteeIndex, err2 = strconv.ParseUint(raw.CommitteeIndex, 10, 64)
	if err2 != nil {
		return nil, fmt.Errorf("committee_index: %w", err2)
	}
	d.ValidatorCommitteeIndex, err2 = strconv.ParseUint(raw.ValidatorCommitteeIndex, 10, 64)
	if err2 != nil {
		return nil, fmt.Errorf("validator_committee_index: %w", err2)
	}
	d.CommitteeLength, err2 = strconv.ParseUint(raw.CommitteeLength, 10, 64)
	if err2 != nil {
		return nil, fmt.Errorf("committee_length: %w", err2)
	}
	return d, nil
}

func fetchAttestationData(c *http.Client, beacon string, slot, committeeIndex uint64) (map[string]any, error) {
	u := fmt.Sprintf("%s/eth/v1/validator/attestation_data?slot=%d&committee_index=%d", beacon, slot, committeeIndex)
	var out struct {
		Data map[string]any `json:"data"`
	}
	if err := getJSON(c, u, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

type forkResp struct {
	Data struct {
		PreviousVersion string `json:"previous_version"`
		CurrentVersion  string `json:"current_version"`
		Epoch           string `json:"epoch"`
	} `json:"data"`
}

func fetchForkAtSlot(c *http.Client, beacon string, slot uint64) (*forkResp, error) {
	var f forkResp
	if err := getJSON(c, beacon+"/eth/v1/beacon/states/"+strconv.FormatUint(slot, 10)+"/fork", &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func forkVersionForSlot(f *forkResp, slot uint64, slotsPerEpoch uint64) (common.Version, error) {
	epoch := slot / slotsPerEpoch
	forkEpoch, err := strconv.ParseUint(f.Data.Epoch, 10, 64)
	if err != nil {
		return common.Version{}, err
	}
	// If the state's fork upgrade activates after this slot's epoch, use previous_version.
	if epoch < forkEpoch {
		return decodeVersion(f.Data.PreviousVersion)
	}
	return decodeVersion(f.Data.CurrentVersion)
}

func parseAttestationData(m map[string]any) (phase0.AttestationData, error) {
	var d phase0.AttestationData
	slot, err := numField(m, "slot")
	if err != nil {
		return d, err
	}
	idx, err := numField(m, "committee_index")
	if err != nil {
		return d, err
	}
	br, err := rootField(m, "beacon_block_root")
	if err != nil {
		return d, err
	}
	src, err := checkpointField(m, "source")
	if err != nil {
		return d, err
	}
	tgt, err := checkpointField(m, "target")
	if err != nil {
		return d, err
	}
	d.Slot = common.Slot(slot)
	d.Index = common.CommitteeIndex(idx)
	d.BeaconBlockRoot = br
	d.Source = src
	d.Target = tgt
	return d, nil
}

func numField(m map[string]any, k string) (uint64, error) {
	v, ok := m[k]
	if !ok {
		return 0, fmt.Errorf("missing %q", k)
	}
	switch t := v.(type) {
	case string:
		return strconv.ParseUint(t, 10, 64)
	case float64:
		return uint64(t), nil
	default:
		return 0, fmt.Errorf("bad type for %q", k)
	}
}

func rootField(m map[string]any, k string) (common.Root, error) {
	v, ok := m[k]
	if !ok {
		return common.Root{}, fmt.Errorf("missing %q", k)
	}
	s, ok := v.(string)
	if !ok {
		return common.Root{}, fmt.Errorf("bad type for %q", k)
	}
	return decodeRoot(s)
}

func checkpointField(m map[string]any, k string) (common.Checkpoint, error) {
	raw, ok := m[k]
	if !ok {
		return common.Checkpoint{}, fmt.Errorf("missing %q", k)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return common.Checkpoint{}, err
	}
	var c struct {
		Epoch string `json:"epoch"`
		Root  string `json:"root"`
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return common.Checkpoint{}, err
	}
	ep, err := strconv.ParseUint(c.Epoch, 10, 64)
	if err != nil {
		return common.Checkpoint{}, err
	}
	r, err := decodeRoot(c.Root)
	if err != nil {
		return common.Checkpoint{}, err
	}
	return common.Checkpoint{Epoch: common.Epoch(ep), Root: r}, nil
}

func decodeRoot(s string) (common.Root, error) {
	var out common.Root
	b, err := decodeHex32(s)
	if err != nil {
		return out, err
	}
	copy(out[:], b)
	return out, nil
}

func decodeVersion(s string) (common.Version, error) {
	var v common.Version
	b, err := decodeHex32(s)
	if err != nil {
		return v, err
	}
	if len(b) != 4 {
		return v, fmt.Errorf("version must be 4 bytes, got %d", len(b))
	}
	copy(v[:], b)
	return v, nil
}

func decodeHex32(s string) ([]byte, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

func parseHex32(s string) ([]byte, error) {
	b, err := decodeHex32(s)
	if err != nil {
		return nil, err
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("expected 32 bytes, got %d", len(b))
	}
	return b, nil
}

func buildAggregationBits(committeeLen, committeePos, maxValidatorsPerCommittee uint64) (phase0.AttestationBits, error) {
	if committeePos >= committeeLen {
		return nil, fmt.Errorf("validator_committee_index %d >= committee_length %d", committeePos, committeeLen)
	}
	numBits := committeeLen + 1
	nBytes := (numBits + 7) / 8
	buf := make([]byte, nBytes)
	bitfields.SetBit(buf, committeePos, true)
	bitfields.SetBit(buf, committeeLen, true)
	if err := bitfields.BitlistCheck(buf, maxValidatorsPerCommittee); err != nil {
		return nil, err
	}
	return phase0.AttestationBits(buf), nil
}

func signAttestationData(sk *blsu.SecretKey, hFn tree.HashFn, domain common.BLSDomain, data *phase0.AttestationData) (common.BLSSignature, error) {
	root := data.HashTreeRoot(hFn)
	sigRoot := common.ComputeSigningRoot(root, domain)
	sig := blsu.Sign(sk, sigRoot[:])
	var out common.BLSSignature
	sb := sig.Serialize()
	copy(out[:], sb[:])
	return out, nil
}

func attestationToJSON(a *phase0.Attestation) map[string]any {
	bits := "0x" + hex.EncodeToString([]byte(a.AggregationBits))
	sig := "0x" + hex.EncodeToString(a.Signature[:])
	return map[string]any{
		"aggregation_bits": bits,
		"data": map[string]any{
			"slot":              strconv.FormatUint(uint64(a.Data.Slot), 10),
			"index":             strconv.FormatUint(uint64(a.Data.Index), 10),
			"beacon_block_root": "0x" + hex.EncodeToString(a.Data.BeaconBlockRoot[:]),
			"source":            checkpointJSON(a.Data.Source),
			"target":            checkpointJSON(a.Data.Target),
		},
		"signature": sig,
	}
}

func checkpointJSON(c common.Checkpoint) map[string]any {
	return map[string]any{
		"epoch": strconv.FormatUint(uint64(c.Epoch), 10),
		"root":  "0x" + hex.EncodeToString(c.Root[:]),
	}
}

func postAttestations(c *http.Client, beacon string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, beacon+"/eth/v1/beacon/pool/attestations", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(b), 500))
	}
	return nil
}

func getJSON(c *http.Client, url string, out any) error {
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(b), 400))
	}
	return json.Unmarshal(b, out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
