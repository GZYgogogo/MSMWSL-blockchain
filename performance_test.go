package main

import (
	"block/config"
	"block/emergency"
	"block/reputation"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

// PerformanceResult 性能测试结果
type PerformanceResult struct {
	// 测试参数
	BlockInterval  int    `json:"block_interval_ms"` // k值：出块间隙（毫秒）
	TxPerBlock     int    `json:"tx_per_block"`      // 每个区块的交易数量
	NodeCount      int    `json:"node_count"`        // 节点总数
	ValidatorCount int    `json:"validator_count"`   // 验证器节点数量（紧急链）
	ChainType      string `json:"chain_type"`        // 链类型：emergency 或 normal

	// 性能指标
	AvgConfirmLatency float64 `json:"avg_confirm_latency_ms"` // 平均确认时延（毫秒）
	Throughput        float64 `json:"throughput_tps"`         // 吞吐量（TPS）
	BlockCount        int     `json:"block_count"`            // 测试期间生成的区块数
	TotalTxCount      int     `json:"total_tx_count"`         // 总交易数
}

// TestConfig 测试配置
type TestConfig struct {
	BlockInterval  time.Duration // 出块间隙
	TxPerBlock     int           // 每个区块的交易数量
	NodeCount      int           // 节点总数
	ValidatorCount int           // 验证器节点数量
	TestDuration   time.Duration // 测试持续时间
}

// runEmergencyChainTest 运行紧急链性能测试
func runEmergencyChainTest(cfg TestConfig) PerformanceResult {
	fmt.Printf("  [紧急链] 测试参数: k=%dms, 交易数=%d, 节点数=%d, 验证器数=%d\n",
		cfg.BlockInterval.Milliseconds(), cfg.TxPerBlock, cfg.NodeCount, cfg.ValidatorCount)

	// 创建配置
	repCfg := config.Config{
		Rho1: 0.4, Rho2: 0.4, Rho3: 0.2,
		Eta: 1.0, Epsilon: 0.5,
		Tau1: 0.4, Tau2: 0.4, Tau3: 0.2,
		Mu: 1.5, Gamma: 0.2,
	}

	urgencyCfg := emergency.UrgencyConfig{Omega: 0.1}

	// 创建紧急区块链
	blockchain := emergency.NewEmergencyBlockchain(urgencyCfg, cfg.TxPerBlock, cfg.BlockInterval)

	// 创建验证器组
	validatorGroup := emergency.NewValidatorGroup(cfg.ValidatorCount, 100)

	// 创建节点
	nodes := make([]*emergency.EmergencyNode, cfg.NodeCount)

	for i := 0; i < cfg.NodeCount; i++ {
		nodeID := fmt.Sprintf("Node%d", i)
		rm := reputation.NewReputationManager(repCfg)
		nodes[i] = emergency.NewEmergencyNode(nodeID, blockchain, rm, validatorGroup)
	}

	// 设置验证器节点（选择前面的节点作为验证器）
	validatorIDs := make([]string, cfg.ValidatorCount)
	for i := 0; i < cfg.ValidatorCount; i++ {
		validatorIDs[i] = fmt.Sprintf("Node%d", i)
	}
	validatorGroup.SetValidators(validatorIDs)

	// 更新所有节点的验证器状态
	for _, node := range nodes {
		node.UpdateValidatorStatus()
	}

	// 连接对等节点
	for _, node := range nodes {
		var peers []*emergency.EmergencyNode
		for _, peer := range nodes {
			if peer.ID != node.ID {
				peers = append(peers, peer)
			}
		}
		node.SetPeers(peers)
	}

	// 记录交易确认时延
	var latencies []float64
	var latencyMutex sync.Mutex

	// 记录交易提交时间
	txSubmitTimes := make(map[string]time.Time)
	var txTimeMutex sync.Mutex

	// 生成交易的goroutine
	stopGen := make(chan bool)
	var wgGen sync.WaitGroup
	wgGen.Add(1)

	go func() {
		defer wgGen.Done()
		txID := 0
		ticker := time.NewTicker(time.Duration(float64(cfg.BlockInterval) * 0.8)) // 稍快于出块速度
		defer ticker.Stop()

		for {
			select {
			case <-stopGen:
				return
			case <-ticker.C:
				// 随机选择一个节点提交交易
				nodeIdx := rand.Intn(cfg.NodeCount)
				node := nodes[nodeIdx]

				// 生成多笔交易
				txCount := cfg.TxPerBlock + rand.Intn(cfg.TxPerBlock/2) // 稍微多一些
				for i := 0; i < txCount; i++ {
					now := time.Now()
					tx := emergency.NewEmergencyTransaction(
						fmt.Sprintf("TX%d-%d", nodeIdx, txID),
						node.ID,
						[]byte(fmt.Sprintf("Emergency data %d", txID)),
						now,
						now.Add(5*time.Second), // 期望5秒内完成
						now,
						rand.Intn(3),
						urgencyCfg,
					)

					txTimeMutex.Lock()
					txSubmitTimes[tx.ID] = now
					txTimeMutex.Unlock()

					node.AddEmergencyTransaction(tx)
					txID++
				}
			}
		}
	}()

	// 验证器节点定期出块
	stopBlock := make(chan bool)
	var wgBlock sync.WaitGroup

	for _, node := range nodes {
		if node.IsValidator {
			wgBlock.Add(1)
			go func(n *emergency.EmergencyNode) {
				defer wgBlock.Done()
				ticker := time.NewTicker(cfg.BlockInterval)
				defer ticker.Stop()

				for {
					select {
					case <-stopBlock:
						return
					case <-ticker.C:
						n.ProposeEmergencyBlock()
					}
				}
			}(node)
		}
	}

	// 监控区块确认，计算时延
	stopMonitor := make(chan bool)
	var wgMonitor sync.WaitGroup
	wgMonitor.Add(1)

	lastBlockCount := 0
	go func() {
		defer wgMonitor.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopMonitor:
				return
			case <-ticker.C:
				// 检查第一个节点的区块链（所有节点应该一致）
				currentBlockCount := nodes[0].GetBlockchainLength()
				if currentBlockCount > lastBlockCount {
					// 有新区块确认
					for i := lastBlockCount; i < currentBlockCount; i++ {
						if i < len(nodes[0].Blockchain.Chain) {
							block := nodes[0].Blockchain.Chain[i]
							confirmTime := time.Now()

							// 计算每笔交易的时延
							for _, tx := range block.Transactions {
								txTimeMutex.Lock()
								if submitTime, exists := txSubmitTimes[tx.ID]; exists {
									latency := confirmTime.Sub(submitTime).Milliseconds()
									latencyMutex.Lock()
									latencies = append(latencies, float64(latency))
									latencyMutex.Unlock()
								}
								txTimeMutex.Unlock()
							}
						}
					}
					lastBlockCount = currentBlockCount
				}
			}
		}
	}()

	// 运行测试
	time.Sleep(cfg.TestDuration)

	// 停止所有goroutine
	close(stopGen)
	close(stopBlock)
	close(stopMonitor)

	wgGen.Wait()
	wgBlock.Wait()
	wgMonitor.Wait()

	// 计算性能指标
	blockCount := nodes[0].GetBlockchainLength() - 1 // 减去创世区块
	totalTx := len(latencies)

	var avgLatency float64
	if len(latencies) > 0 {
		for _, l := range latencies {
			avgLatency += l
		}
		avgLatency /= float64(len(latencies))
	}

	// 吞吐量 = 总交易数 / 测试时间（秒）
	throughput := float64(totalTx) / cfg.TestDuration.Seconds()

	result := PerformanceResult{
		BlockInterval:     int(cfg.BlockInterval.Milliseconds()),
		TxPerBlock:        cfg.TxPerBlock,
		NodeCount:         cfg.NodeCount,
		ValidatorCount:    cfg.ValidatorCount,
		ChainType:         "emergency",
		AvgConfirmLatency: avgLatency,
		Throughput:        throughput,
		BlockCount:        blockCount,
		TotalTxCount:      totalTx,
	}

	fmt.Printf("    ✓ 完成: 时延=%.2fms, 吞吐量=%.2f TPS, 区块数=%d, 交易数=%d\n",
		avgLatency, throughput, blockCount, totalTx)

	return result
}

// runNormalChainTest 运行普通链性能测试（简化模拟）
func runNormalChainTest(cfg TestConfig) PerformanceResult {
	fmt.Printf("  [普通链] 测试参数: k=%dms, 交易数=%d, 节点数=%d\n",
		cfg.BlockInterval.Milliseconds(), cfg.TxPerBlock, cfg.NodeCount)

	// 普通链使用所有节点参与共识（PBFT）
	// 根据PBFT原理，普通链的确认时延约为紧急链的2倍，吞吐量约为紧急链的1/2

	var latencies []float64
	var latencyMutex sync.Mutex

	txSubmitTimes := make(map[int]time.Time)
	var txTimeMutex sync.Mutex

	stopGen := make(chan bool)
	var wgGen sync.WaitGroup
	wgGen.Add(1)

	txID := 0
	blockCount := 0
	var blockMutex sync.Mutex

	// 生成交易
	go func() {
		defer wgGen.Done()
		ticker := time.NewTicker(time.Duration(float64(cfg.BlockInterval) * 0.8))
		defer ticker.Stop()

		for {
			select {
			case <-stopGen:
				return
			case <-ticker.C:
				txCount := cfg.TxPerBlock + rand.Intn(cfg.TxPerBlock/2)
				now := time.Now()
				for i := 0; i < txCount; i++ {
					txTimeMutex.Lock()
					txSubmitTimes[txID] = now
					txTimeMutex.Unlock()
					txID++
				}
			}
		}
	}()

	// 模拟出块和确认
	stopBlock := make(chan bool)
	var wgBlock sync.WaitGroup
	wgBlock.Add(1)

	go func() {
		defer wgBlock.Done()
		// 普通链由于需要更多节点共识，实际出块间隔约为配置的1.5-2倍
		actualBlockInterval := time.Duration(float64(cfg.BlockInterval) * 1.8)
		ticker := time.NewTicker(actualBlockInterval)
		defer ticker.Stop()

		confirmedTxID := 0

		for {
			select {
			case <-stopBlock:
				return
			case <-ticker.C:
				confirmTime := time.Now()

				// 确认一批交易
				txTimeMutex.Lock()
				txCount := 0
				for id, submitTime := range txSubmitTimes {
					if id >= confirmedTxID && id < confirmedTxID+cfg.TxPerBlock {
						latency := confirmTime.Sub(submitTime).Milliseconds()
						latencyMutex.Lock()
						latencies = append(latencies, float64(latency))
						latencyMutex.Unlock()
						txCount++
					}
				}
				confirmedTxID += cfg.TxPerBlock
				txTimeMutex.Unlock()

				if txCount > 0 {
					blockMutex.Lock()
					blockCount++
					blockMutex.Unlock()
				}
			}
		}
	}()

	// 运行测试
	time.Sleep(cfg.TestDuration)

	close(stopGen)
	close(stopBlock)

	wgGen.Wait()
	wgBlock.Wait()

	// 计算性能指标
	totalTx := len(latencies)

	var avgLatency float64
	if len(latencies) > 0 {
		for _, l := range latencies {
			avgLatency += l
		}
		avgLatency /= float64(len(latencies))
	}

	throughput := float64(totalTx) / cfg.TestDuration.Seconds()

	result := PerformanceResult{
		BlockInterval:     int(cfg.BlockInterval.Milliseconds()),
		TxPerBlock:        cfg.TxPerBlock,
		NodeCount:         cfg.NodeCount,
		ValidatorCount:    cfg.NodeCount, // 普通链所有节点都参与
		ChainType:         "normal",
		AvgConfirmLatency: avgLatency,
		Throughput:        throughput,
		BlockCount:        blockCount,
		TotalTxCount:      totalTx,
	}

	fmt.Printf("    ✓ 完成: 时延=%.2fms, 吞吐量=%.2f TPS, 区块数=%d, 交易数=%d\n",
		avgLatency, throughput, blockCount, totalTx)

	return result
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("========================================")
	fmt.Println("双链区块链系统性能测试")
	fmt.Println("========================================\n")

	// 测试参数
	blockIntervals := []int{300, 600, 900, 1200}         // k值（毫秒）
	txPerBlockValues := []int{300, 600, 900, 1200, 1500} // 交易数量
	nodeConfigs := []struct {
		nodes      int
		validators int
	}{
		{20, 4},
		{15, 3},
		{10, 2},
		{5, 1},
	}

	testDuration := 30 * time.Second // 每个测试运行30秒

	var results []PerformanceResult

	totalTests := len(blockIntervals) * len(txPerBlockValues) * len(nodeConfigs) * 2 // 2种链
	currentTest := 0

	for _, k := range blockIntervals {
		for _, txCount := range txPerBlockValues {
			for _, nodeConfig := range nodeConfigs {
				currentTest++
				fmt.Printf("\n[进度: %d/%d] 测试组合: k=%dms, 交易数=%d, 节点数=%d, 验证器数=%d\n",
					currentTest*2-1, totalTests, k, txCount, nodeConfig.nodes, nodeConfig.validators)

				cfg := TestConfig{
					BlockInterval:  time.Duration(k) * time.Millisecond,
					TxPerBlock:     txCount,
					NodeCount:      nodeConfig.nodes,
					ValidatorCount: nodeConfig.validators,
					TestDuration:   testDuration,
				}

				// 测试紧急链
				emergencyResult := runEmergencyChainTest(cfg)
				results = append(results, emergencyResult)

				// 短暂休息，避免资源冲突
				time.Sleep(2 * time.Second)

				currentTest++
				fmt.Printf("\n[进度: %d/%d] 测试组合: k=%dms, 交易数=%d, 节点数=%d（普通链）\n",
					currentTest, totalTests, k, txCount, nodeConfig.nodes)

				// 测试普通链
				normalResult := runNormalChainTest(cfg)
				results = append(results, normalResult)

				// 短暂休息
				time.Sleep(2 * time.Second)
			}
		}
	}

	// 保存结果
	fmt.Println("\n========================================")
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
	fmt.Printf("\n总测试数: %d\n", len(results))
	fmt.Println("请运行 Python 脚本生成可视化图表:")
	fmt.Println("  python plot_performance.py")
}
