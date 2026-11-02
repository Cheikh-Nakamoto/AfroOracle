package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const amoyChainID = 80002

type chainClient interface {
	bind.ContractBackend
	bind.DeployBackend
	ChainID(context.Context) (*big.Int, error)
}

type deployment struct {
	ChainID                int64  `json:"chain_id"`
	Publisher              string `json:"publisher"`
	AgriculturalData       string `json:"agricultural_data_address"`
	AgriculturalDataTx     string `json:"agricultural_data_tx"`
	ParametricInsurance    string `json:"parametric_insurance_address"`
	ParametricInsuranceTx  string `json:"parametric_insurance_tx"`
	InsuranceThresholdX100 string `json:"insurance_threshold_x100"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "deploy-amoy:", err)
		os.Exit(1)
	}
}

func run() error {
	agriBinPath := flag.String("agri-bin", "", "compiled AgriculturalData creation bytecode")
	insuranceBinPath := flag.String("insurance-bin", "", "compiled ParametricInsurance creation bytecode")
	abiPath := flag.String("abi", "../../contracts/abi.json", "canonical contract ABI JSON")
	flag.Parse()
	if *agriBinPath == "" || *insuranceBinPath == "" {
		return errors.New("-agri-bin and -insurance-bin are required; use scripts/deploy-amoy.sh")
	}

	rpc := strings.TrimSpace(os.Getenv("POLYGON_RPC"))
	privateKey := strings.TrimSpace(strings.TrimPrefix(os.Getenv("PRIVATE_KEY"), "0x"))
	if rpc == "" || privateKey == "" {
		return errors.New("POLYGON_RPC and PRIVATE_KEY must be set in the environment")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return errors.New("PRIVATE_KEY is not a valid hex-encoded ECDSA private key")
	}
	threshold, err := insuranceThreshold(os.Getenv("INSURANCE_THRESHOLD_X100"))
	if err != nil {
		return err
	}
	agriCode, err := readBytecode(*agriBinPath)
	if err != nil {
		return fmt.Errorf("AgriculturalData bytecode: %w", err)
	}
	insuranceCode, err := readBytecode(*insuranceBinPath)
	if err != nil {
		return fmt.Errorf("ParametricInsurance bytecode: %w", err)
	}
	insuranceABI, err := contractABI(*abiPath, "ParametricInsurance")
	if err != nil {
		return err
	}
	address, txHash, err := deployContract(ctx, client, key, chainID, agriCode)
	if err != nil {
		return fmt.Errorf("deploy AgriculturalData: %w", err)
	}
	fmt.Fprintf(os.Stderr, "AgriculturalData deployed: %s (tx %s)\n", address.Hex(), txHash.Hex())

	constructorArgs, err := insuranceABI.Pack("", address, threshold)
	if err != nil {
		return fmt.Errorf("encode insurance constructor: %w", err)
	}
	insuranceCode = append(insuranceCode, constructorArgs...)
	insuranceAddress, insuranceTx, err := deployContract(ctx, client, key, chainID, insuranceCode)
	if err != nil {
		return fmt.Errorf("deploy ParametricInsurance: %w", err)
	}

	result := deployment{
		ChainID:                chainID.Int64(),
		Publisher:              crypto.PubkeyToAddress(key.PublicKey).Hex(),
		AgriculturalData:       address.Hex(),
		AgriculturalDataTx:     txHash.Hex(),
		ParametricInsurance:    insuranceAddress.Hex(),
		ParametricInsuranceTx:  insuranceTx.Hex(),
		InsuranceThresholdX100: threshold.String(),
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func requireAmoy(chainID *big.Int) error {
	if chainID == nil || chainID.Cmp(big.NewInt(amoyChainID)) != 0 {
		return fmt.Errorf("refusing deployment: expected Polygon Amoy chain ID %d, connected to %v", amoyChainID, chainID)
	}
	return nil
}

func insuranceThreshold(raw string) (*big.Int, error) {
	if strings.TrimSpace(raw) == "" {
		return big.NewInt(20000), nil
	}
	threshold, ok := new(big.Int).SetString(strings.TrimSpace(raw), 10)
	if !ok || threshold.Sign() <= 0 {
		return nil, errors.New("INSURANCE_THRESHOLD_X100 must be a positive base-10 integer")
	}
	return threshold, nil
}

func readBytecode(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	code, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(string(raw)), "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	if len(code) == 0 {
		return nil, errors.New("bytecode is empty")
	}
	return code, nil
}

func contractABI(path, contractName string) (abi.ABI, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("read contract ABI: %w", err)
	}
	var contracts map[string]json.RawMessage
	if err := json.Unmarshal(raw, &contracts); err != nil {
		return abi.ABI{}, fmt.Errorf("parse contract ABI file: %w", err)
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

func deployContract(ctx context.Context, client chainClient, key *ecdsa.PrivateKey, chainID *big.Int, code []byte) (common.Address, common.Hash, error) {
	opts := bind.NewKeyedTransactor(key, chainID)
	opts.Context = ctx
	contract := bind.NewBoundContract(common.Address{}, abi.ABI{}, client, client, client)
	tx, err := contract.RawCreationTransact(opts, code)
	if err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("submit creation transaction: %w", err)
	}
	receipt, err := bind.WaitMined(ctx, client, tx.Hash())
	if err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("wait for deployment receipt %s: %w", tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, common.Hash{}, fmt.Errorf("deployment transaction %s reverted", tx.Hash())
	}
	if receipt.ContractAddress == (common.Address{}) {
		return common.Address{}, common.Hash{}, fmt.Errorf("deployment transaction %s has no contract address", tx.Hash())
	}
	return receipt.ContractAddress, tx.Hash(), nil
}
