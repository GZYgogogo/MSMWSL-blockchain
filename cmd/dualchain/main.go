package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"block/config"
	"block/emergency"
	"block/reputation"

	"github.com/xuri/excelize/v2"
)

// -------- 普通区块链（PBFT）部分 --------
type NormalBlock struct {
	Index     int
	Timestamp time.Time
	Data      []byte
	PrevHash  string
	Hash      string
}

type NormalMessageType int

const (
	NormalPrePrepare NormalMessageType = iota
	NormalPrepare
	NormalCommit
)

type NormalMessage struct {
	Type  NormalMessageType
	View  int
	Seq   int
	Block NormalBlock
	From  string
}

type NormalNode struct {
	ID     string
	Peers  []*NormalNode
	Rm     *reputation.ReputationManager
	ledger []NormalBlock
	mutex  sync.Mutex
	view   int
	seq    int
}

func NewNormalNode(id string, cfg config.Config) *NormalNode {
	return &NormalNode{ID: id, Rm: reputation.NewReputationManager(cfg)}
}

func (n *NormalNode) Broadcast(msg NormalMessage) {
	for _, peer := range n.Peers {
		go peer.Receive(msg)
	}
}

func (n *NormalNode) Receive(msg NormalMessage) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if msg.Type == NormalCommit {
		n.ledger = append(n.ledger, msg.Block)
	}
}

func (n *NormalNode) Propose(data []byte) {
	n.seq++
	block := NormalBlock{Index: len(n.ledger) + 1, Timestamp: time.Now(), Data: data, PrevHash: n.lastHash()}
	h := sha256.Sum256(append([]byte(block.PrevHash), data...))
	block.Hash = hex.EncodeToString(h[:])
	msg := NormalMessage{Type: NormalPrePrepare, View: n.view, Seq: n.seq, Block: block, From: n.ID}
	n.Broadcast(msg)
	msg.Type = NormalCommit
	n.Broadcast(msg)
}

func (n *NormalNode) lastHash() string {
	if len(n.ledger) == 0 {
		return ""
	}
	return n.ledger[len(n.ledger)-1].Hash
}

// RawData 从 Excel 导入的轨迹数据（包含时间戳）
type RawData struct {
	VehicleID    string
	Time         float64 // 单位：秒
	X            float64
	Y            float64
	Speed        float64
	Acceleration float64
}

// 恶意节点配置
var maliciousNodes = map[string]bool{
	"3": true,
}

func isMalicious(nodeID string) bool {
	return maliciousNodes[nodeID]
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// 创建日志文件
	logFile, err := os.OpenFile("dualchain_log.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("创建日志文件失败:", err)
		return
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(0)

	log.Printf("========================================\n")
	log.Printf("双链区块链系统启动时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("========================================\n\n")

	// 加载配置
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Printf("错误: 加载配置失败: %v\n", err)
		fmt.Println("加载配置失败:", err)
		return
	}
	log.Printf("配置加载成功\n\n")

	// 读取 Excel
	f, err := excelize.OpenFile("data.xlsx")
	if err != nil {
		log.Printf("错误: 打开 data.xlsx 失败: %v\n", err)
		fmt.Println("打开 data.xlsx 失败:", err)
		return
	}
	log.Printf("成功打开数据文件: data.xlsx\n")
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		log.Printf("错误: 读取表格失败或无数据\n")
		fmt.Println("读取表格失败或无数据")
		return
	}

	// 解析表头
	header := rows[0]
	var iVID, iTime, iLong, iSpd, iLane, iAcc int
	for idx, title := range header {
		switch title {
		case "vehicleID":
			iVID = idx
		case "time(s)":
			iTime = idx
		case "longitudinalDistance(m)":
			iLong = idx
		case "speed(m/s)":
			iSpd = idx
		case "laneID":
			iLane = idx
		case "acceleration(m/s^2)":
			iAcc = idx
		}
	}

	// 读取数据
	dataMap := make(map[string][]RawData)
	for _, row := range rows[1:] {
		vid := row[iVID]
		t, _ := strconv.ParseFloat(row[iTime], 64)
		lon, _ := strconv.ParseFloat(row[iLong], 64)
		x := lon
		laneIDInt, _ := strconv.Atoi(row[iLane])
		y := float64(laneIDInt-1) * 3.5
		spd, _ := strconv.ParseFloat(row[iSpd], 64)
		acc, _ := strconv.ParseFloat(row[iAcc], 64)

		dataMap[vid] = append(dataMap[vid], RawData{
			VehicleID:    vid,
			Time:         t,
			X:            x,
			Y:            y,
			Speed:        spd,
			Acceleration: acc,
		})
	}

	// 按时间排序
	for _, slice := range dataMap {
		sort.Slice(slice, func(i, j int) bool { return slice[i].Time < slice[j].Time })
	}

	// 获取车辆ID列表
	var vehicleIDs []string
	for vid := range dataMap {
		vehicleIDs = append(vehicleIDs, vid)
	}
	sort.Strings(vehicleIDs)

	log.Printf("\n节点初始化:\n")
	log.Printf("总节点数: %d\n", len(vehicleIDs))
	log.Printf("节点列表: %v\n\n", vehicleIDs)

	// ======== 初始化普通区块链（所有节点参与PBFT） ========
	normalNodes := make(map[string]*NormalNode)
	for _, vid := range vehicleIDs {
		normalNodes[vid] = NewNormalNode(vid, cfg)
	}
	for _, n := range normalNodes {
		for _, peer := range normalNodes {
			if peer.ID != n.ID {
				n.Peers = append(n.Peers, peer)
			}
		}
	}
	log.Printf("普通区块链初始化完成 (PBFT共识, 所有 %d 个节点参与)\n\n", len(vehicleIDs))

	// ======== 初始化紧急区块链（高信誉值节点组成验证器委员会） ========
	// 紧急度配置
	urgencyCfg := emergency.UrgencyConfig{
		Omega: 0.5, // 已申请紧急交易数量的影响权重
	}

	// 创建紧急区块链
	emergencyBlockchain := emergency.NewEmergencyBlockchain(
		urgencyCfg,
		5,             // 每个区块包含5笔交易
		3*time.Second, // 出块周期3秒
	)

	// 创建验证器节点组（选取前30%信誉值最高的节点）
	validatorGroupSize := int(math.Ceil(float64(len(vehicleIDs)) * 0.3))
	if validatorGroupSize < 4 {
		validatorGroupSize = 4 // 至少4个验证器节点以支持拜占庭容错
	}
	validatorGroup := emergency.NewValidatorGroup(validatorGroupSize, 10) // 10个区块周期后刷新

	// 创建紧急区块链节点
	emergencyNodes := make(map[string]*emergency.EmergencyNode)
	reputationManagers := make(map[string]*reputation.ReputationManager)

	for _, vid := range vehicleIDs {
		reputationManagers[vid] = normalNodes[vid].Rm
		emergencyNodes[vid] = emergency.NewEmergencyNode(
			vid,
			emergencyBlockchain,
			normalNodes[vid].Rm,
			validatorGroup,
		)
	}

	// 设置对等节点
	var emergencyNodeList []*emergency.EmergencyNode
	for _, node := range emergencyNodes {
		emergencyNodeList = append(emergencyNodeList, node)
	}
	for _, node := range emergencyNodes {
		node.SetPeers(emergencyNodeList)
	}

	log.Printf("紧急区块链初始化完成 (PoE共识)\n")
	log.Printf("验证器组大小: %d (占总节点的 %.0f%%)\n\n", validatorGroupSize, float64(validatorGroupSize)/float64(len(vehicleIDs))*100)

	// 构建轨迹向量
	trajMap := make(map[string][]reputation.Vector)
	for _, vid := range vehicleIDs {
		pts := dataMap[vid]
		var vecs []reputation.Vector
		for i := range pts {
			var dir float64
			if i > 0 {
				dx := pts[i].X - pts[i-1].X
				dy := pts[i].Y - pts[i-1].Y
				dir = math.Atan2(dy, dx)
			}
			vecs = append(vecs, reputation.Vector{
				Speed:        pts[i].Speed,
				Direction:    dir,
				Acceleration: pts[i].Acceleration,
			})
		}
		trajMap[vid] = vecs
	}

	// ======== 运行双链系统 ========
	rounds := len(trajMap[vehicleIDs[0]])
	if rounds > 20 { // 限制运行轮数用于演示
		rounds = 20
	}

	log.Printf("开始运行双链系统，共 %d 轮\n", rounds)
	log.Printf("========================================\n\n")

	interChan := make(chan reputation.Interaction, 1000)
	var wg sync.WaitGroup

	go func() {
		for inter := range interChan {
			normalNodes[inter.To].Rm.AddInteraction(inter)
			wg.Done()
		}
	}()

	// 紧急交易计数器（用于计算θ）
	emergencyTxCounter := make(map[string]int)

	for r := 0; r < rounds; r++ {
		roundStartTime := time.Now()

		fmt.Printf("\n========== 第 %d 轮 ==========\n", r+1)
		log.Printf("========== 第 %d 轮 ==========\n", r+1)

		// 1. 普通区块链：提议区块
		proposer := normalNodes[vehicleIDs[r%len(vehicleIDs)]]
		proposer.Propose([]byte(fmt.Sprintf("Normal Round %d", r+1)))
		log.Printf("普通区块链: 节点 %s 提议区块\n", proposer.ID)

		// 2. 信誉交互（与原代码类似，但简化）
		for _, sender := range vehicleIDs {
			// 随机选择几个接收者进行交互
			numInteractions := rand.Intn(3) // 0-2次交互
			for k := 0; k < numInteractions; k++ {
				receiver := vehicleIDs[rand.Intn(len(vehicleIDs))]
				if receiver == sender {
					continue
				}

				raw := dataMap[sender][r]
				baseTime := time.Now().Add(-time.Duration(raw.Time) * time.Second)
				delay := time.Duration(rand.Intn(500)) * time.Millisecond
				ts := baseTime.Add(delay)

				var posEvents, negEvents int
				if isMalicious(sender) {
					posEvents = 0
					negEvents = 1
				} else {
					posEvents = 1
					negEvents = 0
				}

				inter := reputation.Interaction{
					From:          receiver,
					To:            sender,
					PosEvents:     posEvents,
					NegEvents:     negEvents,
					Timestamp:     ts,
					TrajUser:      trajMap[receiver][:r+1],
					TrajProvider:  trajMap[sender][:r+1],
					TxType:        reputation.NormalTransaction, // ⭐ 标记为普通交易
					UrgencyDegree: 0.0,                          // 普通交易无紧急度
				}
				wg.Add(1)
				interChan <- inter
			}
		}
		wg.Wait()

		// 3. 更新验证器节点组（每轮或定期更新）
		if r == 0 || validatorGroup.NeedRefresh() {
			validatorGroup.SelectValidators(vehicleIDs, reputationManagers, time.Now())
			log.Printf("\n验证器节点组已更新:\n")
			for i, v := range validatorGroup.Validators {
				log.Printf("  验证器 %d: 节点 %s (信誉值=%.4f)\n", i+1, v.ID, v.Reputation)
			}
			log.Printf("\n")

			// 更新所有节点的验证器状态
			for _, node := range emergencyNodes {
				node.UpdateValidatorStatus()
			}

			fmt.Printf("验证器节点组已更新，共 %d 个验证器\n", len(validatorGroup.Validators))
		}

		// 4. 生成紧急交易（随机生成1-3笔）
		numEmergencyTx := 1 + rand.Intn(3)
		for i := 0; i < numEmergencyTx; i++ {
			// 随机选择一个节点发送紧急交易
			senderID := vehicleIDs[rand.Intn(len(vehicleIDs))]
			emergencyTxCounter[senderID]++

			// 生成紧急交易
			productTime := time.Now().Add(-time.Duration(rand.Intn(5)) * time.Second)
			deadlineTime := time.Now().Add(time.Duration(5+rand.Intn(10)) * time.Second)
			arrivalTime := time.Now()

			tx := emergency.NewEmergencyTransaction(
				fmt.Sprintf("ETx-%d-%s-%d", r, senderID, i),
				senderID,
				[]byte(fmt.Sprintf("Emergency data from %s", senderID)),
				productTime,
				deadlineTime,
				arrivalTime,
				emergencyTxCounter[senderID],
				urgencyCfg,
			)

			// 广播到所有节点的交易池
			for _, node := range emergencyNodes {
				node.AddEmergencyTransaction(tx)
			}

			fmt.Printf("紧急交易: %s (发送者=%s, 紧急度=%.4f)\n", tx.ID, senderID, tx.UrgencyDegree)
			log.Printf("紧急交易: %s (发送者=%s, 紧急度=%.4f)\n", tx.ID, senderID, tx.UrgencyDegree)
		}

		// 5. 紧急区块链：验证器节点提议紧急区块
		if validatorGroup.GetSize() > 0 {
			proposerValidator := validatorGroup.SelectProposer()
			if proposerValidator != nil {
				emergencyProposer := emergencyNodes[proposerValidator.ID]

				// 等待一小段时间让交易广播完成
				time.Sleep(100 * time.Millisecond)

				emergencyProposer.ProposeEmergencyBlock()

				// 等待共识完成
				time.Sleep(500 * time.Millisecond)
			}
		}

		// 增加验证器组轮数
		validatorGroup.IncrementRound()

		// 输出当前状态
		fmt.Printf("\n普通区块链长度: %d\n", len(proposer.ledger))
		fmt.Printf("紧急区块链长度: %d\n", emergencyBlockchain.GetChainLength())
		fmt.Printf("紧急交易池大小: %d\n", emergencyBlockchain.TxPool.Size())

		log.Printf("\n状态统计:\n")
		log.Printf("  普通区块链长度: %d\n", len(proposer.ledger))
		log.Printf("  紧急区块链长度: %d\n", emergencyBlockchain.GetChainLength())
		log.Printf("  紧急交易池大小: %d\n", emergencyBlockchain.TxPool.Size())
		log.Printf("  本轮耗时: %v\n", time.Since(roundStartTime))
		log.Printf("========================================\n\n")

		fmt.Printf("本轮耗时: %v\n", time.Since(roundStartTime))
	}

	close(interChan)

	// ======== 输出最终统计 ========
	fmt.Printf("\n\n╔════════════════════════════════════════╗\n")
	fmt.Printf("║         双链系统运行总结               ║\n")
	fmt.Printf("╚════════════════════════════════════════╝\n\n")

	log.Printf("\n\n╔════════════════════════════════════════╗\n")
	log.Printf("║         双链系统运行总结               ║\n")
	log.Printf("╚════════════════════════════════════════╝\n\n")

	// 输出普通区块链统计
	fmt.Printf("【普通区块链 - PBFT共识】\n")
	fmt.Printf("  所有节点参与: %d 个节点\n", len(vehicleIDs))
	fmt.Printf("  区块总数: %d\n", len(normalNodes[vehicleIDs[0]].ledger))

	log.Printf("【普通区块链 - PBFT共识】\n")
	log.Printf("  所有节点参与: %d 个节点\n", len(vehicleIDs))
	log.Printf("  区块总数: %d\n", len(normalNodes[vehicleIDs[0]].ledger))

	// 输出紧急区块链统计
	fmt.Printf("\n【紧急区块链 - PoE共识】\n")
	fmt.Printf("  验证器节点: %d 个 (%.0f%%)\n", validatorGroup.GetSize(),
		float64(validatorGroup.GetSize())/float64(len(vehicleIDs))*100)
	fmt.Printf("  区块总数: %d\n", emergencyBlockchain.GetChainLength()-1) // 减去创世区块

	log.Printf("\n【紧急区块链 - PoE共识】\n")
	log.Printf("  验证器节点: %d 个 (%.0f%%)\n", validatorGroup.GetSize(),
		float64(validatorGroup.GetSize())/float64(len(vehicleIDs))*100)
	log.Printf("  区块总数: %d\n", emergencyBlockchain.GetChainLength()-1)

	// 统计紧急区块中的交易
	totalEmergencyTx := 0
	var totalUrgency float64
	for i := 1; i < len(emergencyBlockchain.Chain); i++ {
		block := emergencyBlockchain.Chain[i]
		totalEmergencyTx += len(block.Transactions)
		totalUrgency += block.TotalUrgency
	}

	fmt.Printf("  紧急交易总数: %d\n", totalEmergencyTx)
	if totalEmergencyTx > 0 {
		fmt.Printf("  平均紧急度: %.4f\n", totalUrgency/float64(totalEmergencyTx))
	}

	log.Printf("  紧急交易总数: %d\n", totalEmergencyTx)
	if totalEmergencyTx > 0 {
		log.Printf("  平均紧急度: %.4f\n", totalUrgency/float64(totalEmergencyTx))
	}

	// 输出验证器节点信息
	fmt.Printf("\n【验证器节点信息】\n")
	log.Printf("\n【验证器节点信息】\n")

	for i, v := range validatorGroup.Validators {
		fmt.Printf("  第 %d 名: 节点 %s (信誉值=%.4f)\n", i+1, v.ID, v.Reputation)
		log.Printf("  第 %d 名: 节点 %s (信誉值=%.4f)\n", i+1, v.ID, v.Reputation)
	}

	// 输出所有节点的最终信誉值
	fmt.Printf("\n【所有节点最终信誉值】\n")
	log.Printf("\n【所有节点最终信誉值】\n")

	type NodeReputation struct {
		ID          string
		Reputation  float64
		IsValidator bool
	}
	var allNodeReputation []NodeReputation

	for _, vid := range vehicleIDs {
		repu := normalNodes[vid].Rm.ComputeReputation(vid, time.Now())
		isValidator := validatorGroup.IsValidator(vid)
		allNodeReputation = append(allNodeReputation, NodeReputation{
			ID:          vid,
			Reputation:  repu,
			IsValidator: isValidator,
		})
	}

	sort.Slice(allNodeReputation, func(i, j int) bool {
		return allNodeReputation[i].Reputation > allNodeReputation[j].Reputation
	})

	for i, nr := range allNodeReputation {
		nodeType := "普通节点"
		if nr.IsValidator {
			nodeType = "✅验证器"
		}
		if isMalicious(nr.ID) {
			nodeType += " ⚠️恶意"
		}

		fmt.Printf("  第 %d 名: 节点 %s [%s] = %.6f\n", i+1, nr.ID, nodeType, nr.Reputation)
		log.Printf("  第 %d 名: 节点 %s [%s] = %.6f\n", i+1, nr.ID, nodeType, nr.Reputation)
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("双链系统运行完成！\n")
	fmt.Printf("详细日志已保存到 dualchain_log.txt\n")
	fmt.Printf("========================================\n")

	log.Printf("\n========================================\n")
	log.Printf("结束时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("========================================\n")
}
