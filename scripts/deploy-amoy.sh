#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
TASK_PID=$$
ARTIFACT_DIR="${TMPDIR:-/tmp}/afrioracle-solc-${TASK_PID}"

if [ -z "${POLYGON_RPC:-}" ]; then
	printf '%s\n' 'POLYGON_RPC is required (use a Polygon Amoy RPC endpoint).' >&2
	exit 1
fi
if [ -z "${PRIVATE_KEY:-}" ]; then
	printf '%s\n' 'PRIVATE_KEY is required; use the dedicated Amoy test wallet key.' >&2
	exit 1
fi

mkdir -p "$ARTIFACT_DIR"
npx --yes solc@0.8.20 --bin --base-path "$ROOT_DIR" -o "$ARTIFACT_DIR" \
	"$ROOT_DIR/contracts/AgriculturalData.sol" \
	"$ROOT_DIR/contracts/ParametricInsurance.sol"

cd "$ROOT_DIR/backend"
go run ./cmd/deploy-amoy \
	-agri-bin "$ARTIFACT_DIR/contracts_AgriculturalData_sol_AgriculturalData.bin" \
	-insurance-bin "$ARTIFACT_DIR/contracts_ParametricInsurance_sol_ParametricInsurance.bin" \
	-abi "$ROOT_DIR/contracts/abi.json"
