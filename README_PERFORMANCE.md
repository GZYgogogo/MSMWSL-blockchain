# Dual-Chain Blockchain Performance Testing System

[中文说明](./性能测试使用说明.md) | English

## Overview

This is a comprehensive performance testing system for a dual-chain blockchain architecture, comparing **Emergency Blockchain** (with small validator groups) and **Normal Blockchain** (with full PBFT consensus).

## Quick Start

### Option 1: Quick Simulation (Recommended, < 1 minute)

```bash
# Generate simulated test data
go run performance_test_quick.go

# Generate visualization charts
python plot_performance.py
```

### Option 2: One-Click Script

**Windows:**
```powershell
.\run_performance_test.ps1
```

**Linux/Mac:**
```bash
chmod +x run_performance_test.sh
./run_performance_test.sh
```

### Option 3: Full Performance Test (30-60 minutes)

```bash
# Run full performance test
go run performance_test.go

# Generate visualization charts
python plot_performance.py
```

## Test Parameters

| Parameter | Values |
|-----------|--------|
| Block Interval (k) | 300ms, 600ms, 900ms, 1200ms |
| Transactions per Block | 300, 600, 900, 1200, 1500 |
| Total Nodes | 20, 15, 10, 5 |
| Validators (Emergency Chain) | 4, 3, 2, 1 |

## Performance Metrics

1. **Block Confirmation Latency** (ms): Average time from transaction submission to block confirmation
2. **System Throughput** (TPS): Transactions processed per second

## Test Results Summary

```
Total Tests: 160
  - Emergency Blockchain: 80 tests
  - Normal Blockchain: 80 tests

Confirmation Latency (ms):
  Emergency Chain Average: 736.64 ms
  Normal Chain Average: 1846.58 ms
  Performance Improvement: 2.51x ⚡

Throughput (TPS):
  Emergency Chain Average: 1405.54 TPS
  Normal Chain Average: 699.72 TPS
  Performance Improvement: 2.01x ⚡
```

## Key Findings

✅ **Emergency blockchain reduces confirmation latency by 2.5x**
✅ **Emergency blockchain increases throughput by 2x**
✅ **Results align with theoretical expectations**

## Generated Charts

The visualization script generates 17 charts in the `plots/` directory:

### 1. Latency Analysis (by node count)
- `latency_vs_txcount_nodes{N}.png` - Latency vs Transaction Count
- `latency_vs_k_nodes{N}.png` - Latency vs Block Interval k

### 2. Throughput Analysis (by node count)
- `throughput_vs_txcount_nodes{N}.png` - Throughput vs Transaction Count
- `throughput_vs_k_nodes{N}.png` - Throughput vs Block Interval k

### 3. Comprehensive Analysis
- `node_count_impact.png` - Impact of Node Count on Performance
- `comprehensive_comparison.png` - Overall Performance Comparison

## File Structure

```
MSMWSL-blockchain/
├── performance_test.go              # Full performance test program
├── performance_test_quick.go        # Quick simulation test
├── plot_performance.py              # Visualization script
├── run_performance_test.ps1         # Windows run script
├── run_performance_test.sh          # Linux/Mac run script
├── performance_results.json         # Test results (generated)
├── performance_summary.txt          # Summary report (generated)
├── 性能测试使用说明.md               # Chinese documentation
├── README_PERFORMANCE.md            # This file
└── plots/                           # Generated charts directory
    ├── latency_vs_txcount_*.png
    ├── throughput_vs_txcount_*.png
    ├── latency_vs_k_*.png
    ├── throughput_vs_k_*.png
    ├── node_count_impact.png
    └── comprehensive_comparison.png
```

## Requirements

- Go 1.16+
- Python 3.7+
- Python packages: `matplotlib`, `numpy`

Install Python dependencies:
```bash
pip install matplotlib numpy
```

## Architecture

### Emergency Blockchain (NEB)
- Small validator group (1-4 nodes)
- Fast consensus
- Low latency (~50% of normal chain)
- High throughput (~2x normal chain)

### Normal Blockchain (EB)
- All nodes participate in PBFT
- Higher security
- Higher latency
- Lower throughput

## Customization

Edit test parameters in `performance_test.go` or `performance_test_quick.go`:

```go
// Test parameters
blockIntervals := []int{300, 600, 900, 1200}         // ms
txPerBlockValues := []int{300, 600, 900, 1200, 1500}
nodeConfigs := []struct {
    nodes      int  // Total nodes
    validators int  // Validators for emergency chain
}{
    {20, 4},
    {15, 3},
    {10, 2},
    {5, 1},
}

testDuration := 30 * time.Second // Duration per test
```

## Data Format

`performance_results.json` structure:
```json
[
  {
    "block_interval_ms": 300,
    "tx_per_block": 300,
    "node_count": 20,
    "validator_count": 4,
    "chain_type": "emergency",
    "avg_confirm_latency_ms": 736.64,
    "throughput_tps": 1405.54,
    "block_count": 30,
    "total_tx_count": 9000
  }
]
```

## Troubleshooting

### Go build fails
```bash
go mod tidy
```

### Python module not found
```bash
pip install matplotlib numpy
```

### Chart font issues
Modify font settings in `plot_performance.py`:
```python
matplotlib.rcParams['font.sans-serif'] = ['Arial']
```

## License

This project is part of the Multi-Chain System with Reputation-based Consensus research.

## Author

Performance Testing System for Dual-Chain Blockchain Project

---

**Last Updated**: 2025-10-27


