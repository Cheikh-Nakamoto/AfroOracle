package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const amoyChainID = 80002

type checkResult struct {
	ChainID           int64  `json:"chain_id"`
	InsuranceContract string `json:"insurance_contract"`
	Zone              string `json:"zone"`
	Product           string `json:"product"`
	Date              string `json:"date"`
	ThresholdX100     string `json:"threshold_x100"`
	ThresholdFCFA     string `json:"threshold_fcfa"`
	PayoutX100        string `json:"payout_x100"`
	PayoutFCFA        string `json:"payout_fcfa"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "check-insurance:", err)
		os.Exit(1)
	}
}

func run() error {
	rpc := strings.TrimSpace(os.Getenv("POLYGON_RPC"))
	addressText := strings.TrimSpace(os.Getenv("INSURANCE_CONTRACT_ADDRESS"))
	zone := envOr("ORACLE_ZONE", "Kaolack")
	date := envOr("ORACLE_DATE", time.Now().UTC().Format("2006-01-02"))
	if rpc == "" || addressText == "" {
		return errors.New("POLYGON_RPC and INSURANCE_CONTRACT_ADDRESS are required")
	}
	if err := validateDate(date); err != nil {
		return err
	}
	if !common.IsHexAddress(addressText) {
		return errors.New("INSURANCE_CONTRACT_ADDRESS is not a valid EVM address")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpc)
	if err != nil {
		return fmt.Errorf("connect to Polygon RPC: %w", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("read RPC chain ID: %w", err)
	}
	if err := requireAmoy(chainID); err != nil {
		return err
	}
	address := common.HexToAddress(addressText)
	code, err := client.CodeAt(ctx, address, nil)
	if err != nil {
		return fmt.Errorf("read insurance contract code: %w", err)
	}
	if len(code) == 0 {
		return fmt.Errorf("no contract code at %s on Polygon Amoy", address.Hex())
	}

	contractABI, err := loadContractABI("../contracts/abi.json", "ParametricInsurance")
	if err != nil {
		return err
	}
	thresholdValues, err := callView(ctx, client, address, contractABI, "thresholdX100")
	if err != nil {
		return fmt.Errorf("read insurance threshold: %w", err)
	}
	threshold, err := asBigInt(thresholdValues)
	if err != nil {
		return fmt.Errorf("decode insurance threshold: %w", err)
	}
	payoutValues, err := callView(ctx, client, address, contractABI, "check", zone, date)
	if err != nil {
		return fmt.Errorf("insurance check for %s on %s failed: %w", zone, date, err)
	}
	payout, err := asBigInt(payoutValues)
	if err != nil {
		return fmt.Errorf("decode insurance payout: %w", err)
	}

	result := checkResult{
		ChainID:           chainID.Int64(),
		InsuranceContract: address.Hex(),
		Zone:              zone,
		Product:           "millet",
		Date:              date,
		ThresholdX100:     threshold.String(),
		ThresholdFCFA:     formatX100(threshold),
		PayoutX100:        payout.String(),
		PayoutFCFA:        formatX100(payout),
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func requireAmoy(chainID *big.Int) error {
	if chainID == nil || chainID.Cmp(big.NewInt(amoyChainID)) != 0 {
		return fmt.Errorf("refusing insurance read: expected Polygon Amoy chain ID %d, connected to %v", amoyChainID, chainID)
	}
	return nil
}

func validateDate(date string) error {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil || parsed.Format("2006-01-02") != date {
		return fmt.Errorf("ORACLE_DATE must use YYYY-MM-DD format, got %q", date)
	}
	return nil
}

func loadContractABI(path, contractName string) (abi.ABI, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("read ABI %s: %w (run from backend/)", path, err)
	}
	var contracts map[string]json.RawMessage
	if err := json.Unmarshal(raw, &contracts); err != nil {
		return abi.ABI{}, fmt.Errorf("parse ABI file: %w", err)
	}
	contract, ok := contracts[contractName]
	if !ok {
		return abi.ABI{}, fmt.Errorf("ABI %q not found in %s", contractName, path)
	}
	parsed, err := abi.JSON(strings.NewReader(string(contract)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("parse %s ABI: %w", contractName, err)
	}
	return parsed, nil
}

func callView(ctx context.Context, client *ethclient.Client, address common.Address, contractABI abi.ABI, method string, args ...any) ([]any, error) {
	data, err := contractABI.Pack(method, args...)
	if err != nil {
		return nil, fmt.Errorf("encode %s call: %w", method, err)
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &address, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	values, err := contractABI.Unpack(method, out)
	if err != nil {
		return nil, fmt.Errorf("decode %s response: %w", method, err)
	}
	return values, nil
}

func asBigInt(values []any) (*big.Int, error) {
	if len(values) != 1 {
		return nil, fmt.Errorf("expected one output, received %d", len(values))
	}
	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("expected uint256 output, got %T", values[0])
	}
	return value, nil
}

func formatX100(value *big.Int) string {
	whole, cents := new(big.Int), new(big.Int)
	whole.QuoRem(value, big.NewInt(100), cents)
	return fmt.Sprintf("%s.%02d", whole.String(), cents.Int64())
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
