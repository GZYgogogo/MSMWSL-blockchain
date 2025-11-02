package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// PerformanceResult 性能测试结果
type PerformanceResult struct {
	BlockInterval     int     `json:"block_interval_ms"`
	TxPerBlock        int     `json:"tx_per_block"`
	NodeCount         int     `json:"node_count"`
	ValidatorCount    int     `json:"validator_count"`
	ChainType         string  `json:"chain_type"`
	AvgConfirmLatency float64 `json:"avg_confirm_latency_ms"`
	Throughput        float64 `json:"throughput_tps"`
	BlockCount        int     `json:"block_count"`
	TotalTxCount      int     `json:"total_tx_count"`
}

// 快速模拟版本 - 基于理论值生成测试数据
func generateSimulatedResults() []PerformanceResult {
	rand.Seed(time.Now().UnixNano())

	blockIntervals := []int{300, 600, 900, 1200}
	txPerBlockValues := []int{300, 600, 900, 1200, 1500}
	nodeConfigs := []struct {
		nodes      int
		validators int
	}{
		{20, 4},
		{15, 3},
		{10, 2},
		{5, 1},
	}

	var results []PerformanceResult

	for _, k := range blockIntervals {
		for _, txCount := range txPerBlockValues {
			for _, nodeConfig := range nodeConfigs {
				// 紧急链结果
				// 基础时延与k成正比，与节点数成正比，与交易数轻微正相关
				baseLatencyEmergency := float64(k) * 0.8 * (1.0 + float64(nodeConfig.validators)*0.05) * (1.0 + float64(txCount)*0.0001)
				latencyEmergency := baseLatencyEmergency + rand.Float64()*50 - 25

				// 吞吐量与k成反比，与交易数成正比
				baseThroughputEmergency := float64(txCount) / (float64(k) / 1000.0) * 0.9
				throughputEmergency := baseThroughputEmergency + rand.Float64()*20 - 10

				emergencyResult := PerformanceResult{
					BlockInterval:     k,
					TxPerBlock:        txCount,
					NodeCount:         nodeConfig.nodes,
					ValidatorCount:    nodeConfig.validators,
					ChainType:         "emergency",
					AvgConfirmLatency: latencyEmergency,
					Throughput:        throughputEmergency,
					BlockCount:        30,
					TotalTxCount:      txCount * 30,
				}
				results = append(results, emergencyResult)

				// 普通链结果（时延约为紧急链的2倍，吞吐量约为紧急链的1/2）
				latencyNormal := latencyEmergency * (1.8 + rand.Float64()*0.4) * (1.0 + float64(nodeConfig.nodes)*0.02)
				throughputNormal := throughputEmergency / (1.8 + rand.Float64()*0.4)

				normalResult := PerformanceResult{
					BlockInterval:     k,
					TxPerBlock:        txCount,
					NodeCount:         nodeConfig.nodes,
					ValidatorCount:    nodeConfig.nodes,
					ChainType:         "normal",
					AvgConfirmLatency: latencyNormal,
					Throughput:        throughputNormal,
					BlockCount:        30,
					TotalTxCount:      txCount * 30,
				}
				results = append(results, normalResult)
			}
		}
	}

	return results
}

func main() {
	fmt.Println("========================================")
	fmt.Println("双链区块链系统性能测试（快速模拟版）")
	fmt.Println("========================================\n")

	fmt.Println("生成模拟测试数据...")
	fmt.Println("注意：这是基于理论模型的快速模拟版本")
	fmt.Println("完整实际测试请运行：go run performance_test.go\n")

	results := generateSimulatedResults()

	// 统计
	emergencyCount := 0
	normalCount := 0
	for _, r := range results {
		if r.ChainType == "emergency" {
			emergencyCount++
		} else {
			normalCount++
		}
	}

	fmt.Printf("生成测试结果:\n")
	fmt.Printf("  - 紧急区块链测试: %d 组\n", emergencyCount)
	fmt.Printf("  - 普通区块链测试: %d 组\n", normalCount)
	fmt.Printf("  - 总计: %d 组\n\n", len(results))

	// 保存结果
	fmt.Println("保存测试结果...")

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("序列化结果失败: %v\n", err)
		return
	}

	err = os.WriteFile("performance_results.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("保存结果失败: %v\n", err)
		return
	}

	fmt.Println("✓ 结果已保存到 performance_results.json")
	fmt.Println("========================================")
	fmt.Println("\n请运行 Python 脚本生成可视化图表:")
	fmt.Println("  python plot_performance.py")
	fmt.Println("\n或运行完整脚本:")
	fmt.Println("  .\\run_performance_test.ps1 (Windows)")
	fmt.Println("  ./run_performance_test.sh (Linux/Mac)")
}

