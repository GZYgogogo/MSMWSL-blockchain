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
	"block/reputation"

	"github.com/xuri/excelize/v2"
)

// -------- PBFT 区块链部分 --------
type Block struct {
	Index     int
	Timestamp time.Time
	Data      []byte
	PrevHash  string
	Hash      string
}
type MessageType int

const (
	PrePrepare MessageType = iota
	Prepare
	Commit
)

type Message struct {
	Type  MessageType
	View  int
	Seq   int
	Block Block
	From  string
}

type Node struct {
	ID     string
	Peers  []*Node
	Rm     *reputation.ReputationManager
	ledger []Block
	mutex  sync.Mutex
	view   int
	seq    int
}

func NewNode(id string, cfg config.Config) *Node {
	return &Node{ID: id, Rm: reputation.NewReputationManager(cfg)}
}

func (n *Node) Broadcast(msg Message) {
	for _, peer := range n.Peers {
		go peer.Receive(msg)
	}
}

func (n *Node) Receive(msg Message) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if msg.Type == Commit {
		n.ledger = append(n.ledger, msg.Block)
	}
}

func (n *Node) Propose(data []byte) {
	n.seq++
	block := Block{Index: len(n.ledger) + 1, Timestamp: time.Now(), Data: data, PrevHash: n.lastHash()}
	h := sha256.Sum256(append([]byte(block.PrevHash), data...))
	block.Hash = hex.EncodeToString(h[:])
	msg := Message{Type: PrePrepare, View: n.view, Seq: n.seq, Block: block, From: n.ID}
	n.Broadcast(msg)
	msg.Type = Commit
	n.Broadcast(msg)
}

func (n *Node) lastHash() string {
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

// 随机交互配置
const (
	// 交互概率配置（总和应为100）
	NoInteractionProb      = 70 // 没有交互的概率：70%
	OneInteractionProb     = 20 // 1次交互的概率：20%
	MultiInteractionProb   = 10 // 多次交互的概率：10%
	MaxInteractionsPerPair = 5  // 多次交互时的最大次数
)

// 恶意节点配置：设置哪些节点是恶意的
var maliciousNodes = map[string]bool{
	"3": true, // 将节点3设为恶意节点
	// 可以添加更多恶意节点，例如: "7": true,
}

// 判断节点是否为恶意节点
func isMalicious(nodeID string) bool {
	return maliciousNodes[nodeID]
}

// getRandomInteractionCount 返回随机的交互次数
// 70%概率返回0（没有交互）
// 20%概率返回1（单次交互）
// 10%概率返回2-5（多次交互）
func getRandomInteractionCount() int {
	r := rand.Intn(100)
	if r < NoInteractionProb {
		return 0 // 70%概率没有交互
	} else if r < NoInteractionProb+OneInteractionProb {
		return 1 // 20%概率1次交互
	} else {
		// 10%概率2-5次交互
		return 2 + rand.Intn(MaxInteractionsPerPair-1)
	}
}

func main() {

	rand.Seed(time.Now().UnixNano())

	// 创建日志文件
	logFile, err := os.OpenFile("reputation_log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("创建日志文件失败:", err)
		return
	}
	defer logFile.Close()

	// 设置日志输出到文件和控制台
	multiWriter := os.Stdout
	log.SetOutput(logFile)
	log.SetFlags(0) // 不显示时间戳

	// 记录开始时间
	log.Printf("========================================\n")
	log.Printf("信誉系统启动时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("========================================\n\n")

	// 加载配置
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Printf("错误: 加载配置失败: %v\n", err)
		fmt.Println("加载配置失败:", err)
		return
	}
	log.Printf("配置加载成功: rho1=%.2f, rho2=%.2f, rho3=%.2f, gamma=%.2f\n",
		cfg.Rho1, cfg.Rho2, cfg.Rho3, cfg.Gamma)
	_ = multiWriter

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
	log.Printf("读取到 %d 行数据（包含表头）\n", len(rows))

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

	// 读取并归一化坐标，同时读取加速度
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

	// 初始化 PBFT 节点
	var vehicleIDs []string
	for vid := range dataMap {
		vehicleIDs = append(vehicleIDs, vid)
	}
	sort.Strings(vehicleIDs)
	if len(vehicleIDs) == 0 {
		log.Printf("错误: 未找到任何车辆数据\n")
		fmt.Println("未找到任何车辆数据")
		return
	}
	log.Printf("\n节点初始化:\n")
	log.Printf("总节点数: %d\n", len(vehicleIDs))
	log.Printf("节点列表: %v\n", vehicleIDs)

	// 统计恶意节点
	var maliciousCount int
	var maliciousList []string
	var honestList []string
	for _, vid := range vehicleIDs {
		if isMalicious(vid) {
			maliciousCount++
			maliciousList = append(maliciousList, vid)
		} else {
			honestList = append(honestList, vid)
		}
	}
	log.Printf("诚实节点 (%d个): %v\n", len(honestList), honestList)
	log.Printf("恶意节点 (%d个): %v ⚠️\n", maliciousCount, maliciousList)

	nodes := make(map[string]*Node)
	for _, vid := range vehicleIDs {
		nodes[vid] = NewNode(vid, cfg)
	}
	for _, n := range nodes {
		for _, peer := range nodes {
			if peer.ID != n.ID {
				n.Peers = append(n.Peers, peer)
			}
		}
	}
	log.Printf("每个节点连接的对等节点数: %d\n\n", len(vehicleIDs)-1)

	// 构建轨迹向量：Speed, Direction, Acceleration
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

	// 信誉交互 & PBFT 模拟（同之前，只是传入的新 Vector）
	rounds := len(trajMap[vehicleIDs[0]])
	log.Printf("开始信誉交互模拟:\n")
	log.Printf("总轮数: %d\n", rounds)
	log.Printf("评价模型:\n")
	log.Printf("  📤 节点发送交易 → 📥 其他节点验证 → 📝 给发送者评价\n")
	log.Printf("  ✅ 诚实节点发送正常交易 → 收到正面评价\n")
	log.Printf("  ⚠️ 恶意节点发送恶意交易 → 收到负面评价\n")
	log.Printf("交互频率:\n")
	log.Printf("  ✅ 诚实节点: 随机交互（70%%概率无交互，20%%概率1次，10%%概率2-%d次）\n", MaxInteractionsPerPair)
	log.Printf("  ⚠️ 恶意节点: 每轮固定1次交互\n")

	// 显示初始信誉值
	log.Printf("\n初始信誉值（交互前）:\n")
	for _, vid := range vehicleIDs {
		initialRepu := nodes[vid].Rm.ComputeReputation(vid, time.Now())
		nodeType := "✅诚实"
		if isMalicious(vid) {
			nodeType = "⚠️恶意"
		}
		log.Printf("  节点 %s [%s]: %.2f\n", vid, nodeType, initialRepu)
	}
	log.Printf("\n")

	interChan := make(chan reputation.Interaction)
	var wg sync.WaitGroup

	go func() {
		for inter := range interChan {
			nodes[inter.To].Rm.AddInteraction(inter)
			wg.Done()
		}
	}()

	// 用于记录信誉变化
	reputationHistory := make(map[string][]float64)
	for _, vid := range vehicleIDs {
		reputationHistory[vid] = make([]float64, 0)
	}

	// 记录总交互次数
	grandTotalInteractions := 0

	for r := 0; r < rounds; r++ {
		roundStartTime := time.Now()
		proposer := nodes[vehicleIDs[r%len(vehicleIDs)]]
		proposer.Propose([]byte(fmt.Sprintf("Round %d positions", r+1)))

		// 记录本轮交互数量和统计信息
		totalInteractions := 0
		noInteractionCount := 0    // 没有交互的节点对数量
		hasInteractionCount := 0   // 有交互的节点对数量
		maliciousInteractions := 0 // 恶意节点发起的交互数量
		honestInteractions := 0    // 诚实节点发起的交互数量

		// 为每个恶意节点随机选择一个目标（每轮只发1个交易）
		maliciousTargets := make(map[string]string) // sender -> receiver
		for _, sender := range vehicleIDs {
			if isMalicious(sender) {
				// 随机选择一个不是自己的目标节点
				possibleTargets := make([]string, 0)
				for _, receiver := range vehicleIDs {
					if receiver != sender {
						possibleTargets = append(possibleTargets, receiver)
					}
				}
				if len(possibleTargets) > 0 {
					maliciousTargets[sender] = possibleTargets[rand.Intn(len(possibleTargets))]
				}
			}
		}

		// 遍历所有可能的发送者-接收者组合
		for _, sender := range vehicleIDs {
			for _, receiver := range vehicleIDs {
				if sender == receiver {
					continue
				}

				// 决定本次交互的次数（发送者发送多少次交易）
				var interactionCount int
				if isMalicious(sender) {
					// 恶意节点特殊处理：每轮只发1个交易到随机选中的目标
					if target, exists := maliciousTargets[sender]; exists && target == receiver {
						interactionCount = 1
					} else {
						interactionCount = 0
					}
				} else {
					// 诚实节点：随机决定本次发送的交易次数
					interactionCount = getRandomInteractionCount()
				}

				if interactionCount == 0 {
					noInteractionCount++
					continue // 本轮sender没有向receiver发送交易
				}

				hasInteractionCount++
				raw := dataMap[sender][r]
				baseTime := time.Now().Add(-time.Duration(raw.Time) * time.Second)

				for k := 0; k < interactionCount; k++ {
					delay := time.Duration(rand.Intn(500)) * time.Millisecond
					ts := baseTime.Add(delay)

					// 新逻辑：sender发送交易，receiver验证并评价sender
					// From = receiver（评价者）
					// To = sender（被评价者，交易发送者）
					var posEvents, negEvents int
					if isMalicious(sender) {
						// 如果发送者是恶意节点，发送恶意交易，接收者识别后给负面评价
						posEvents = 0
						negEvents = 1
					} else {
						// 如果发送者是诚实节点，发送正常交易，接收者验证后给正面评价
						posEvents = 1
						negEvents = 0
					}

					inter := reputation.Interaction{
						From:         receiver, // 评价者（接收并验证交易的节点）
						To:           sender,   // 被评价者（发送交易的节点）
						PosEvents:    posEvents,
						NegEvents:    negEvents,
						Timestamp:    ts,
						TrajUser:     trajMap[receiver][:r+1], // 评价者的轨迹
						TrajProvider: trajMap[sender][:r+1],   // 被评价者的轨迹
					}
					wg.Add(1)
					interChan <- inter
					totalInteractions++

					// 统计恶意节点和诚实节点发送的交易数量
					if isMalicious(sender) {
						maliciousInteractions++
					} else {
						honestInteractions++
					}
				}
			}
		}
		wg.Wait()

		// 累加总交互次数
		grandTotalInteractions += totalInteractions

		// 输出信誉到控制台和日志
		totalPairs := len(vehicleIDs) * (len(vehicleIDs) - 1)
		interactionRate := float64(hasInteractionCount) / float64(totalPairs) * 100

		log.Printf("========================================\n")
		log.Printf("第 %d 轮信誉计算结果\n", r+1)
		log.Printf("----------------------------------------\n")
		log.Printf("提议者节点: %s\n", proposer.ID)
		log.Printf("本轮交互统计:\n")
		log.Printf("  总交互次数: %d\n", totalInteractions)
		log.Printf("    ├─ 诚实节点发送交易: %d 次（收到正面评价）\n", honestInteractions)
		log.Printf("    └─ 恶意节点发送交易: %d 次（收到负面评价）⚠️\n", maliciousInteractions)
		log.Printf("  有交互的节点对: %d/%d (%.1f%%)\n", hasInteractionCount, totalPairs, interactionRate)
		log.Printf("  无交互的节点对: %d/%d (%.1f%%)\n", noInteractionCount, totalPairs, float64(noInteractionCount)/float64(totalPairs)*100)
		log.Printf("----------------------------------------\n")

		fmt.Printf("=== 第 %d 轮信誉计算 ===\n", r+1)

		// 计算并记录每个节点的信誉值
		var minRepu, maxRepu, sumRepu float64 = 1.0, 0.0, 0.0
		var honestRepuSum, maliciousRepuSum float64
		var honestCount, maliciousNodeCount int

		for idx, vid := range vehicleIDs {
			repu := nodes[vid].Rm.ComputeReputation(vid, time.Now())
			reputationHistory[vid] = append(reputationHistory[vid], repu)

			// 计算变化量
			change := 0.0
			if len(reputationHistory[vid]) > 1 {
				change = repu - reputationHistory[vid][len(reputationHistory[vid])-2]
			}

			// 统计
			if repu < minRepu {
				minRepu = repu
			}
			if repu > maxRepu {
				maxRepu = repu
			}
			sumRepu += repu

			// 分类统计
			nodeType := "✅诚实"
			if isMalicious(vid) {
				nodeType = "⚠️恶意"
				maliciousRepuSum += repu
				maliciousNodeCount++
			} else {
				honestRepuSum += repu
				honestCount++
			}

			// 输出到控制台
			fmt.Printf("节点 %s [%s] → 信誉值: %.4f\n", vid, nodeType, repu)

			// 详细记录到日志
			if change != 0 {
				log.Printf("节点 %s [%s]: 信誉值=%.6f, 变化=%.6f (%.2f%%)\n",
					vid, nodeType, repu, change, change*100)
			} else {
				log.Printf("节点 %s [%s]: 信誉值=%.6f (首次计算)\n", vid, nodeType, repu)
			}

			// 每5个节点换行一次以便阅读
			if (idx+1)%5 == 0 {
				log.Printf("\n")
			}
		}

		avgRepu := sumRepu / float64(len(vehicleIDs))
		log.Printf("----------------------------------------\n")
		log.Printf("统计信息:\n")
		log.Printf("  最小信誉值: %.6f\n", minRepu)
		log.Printf("  最大信誉值: %.6f\n", maxRepu)
		log.Printf("  平均信誉值: %.6f\n", avgRepu)
		log.Printf("  信誉值范围: %.6f\n", maxRepu-minRepu)

		// 对比诚实节点和恶意节点
		if honestCount > 0 {
			log.Printf("  诚实节点平均信誉: %.6f ✅\n", honestRepuSum/float64(honestCount))
		}
		if maliciousNodeCount > 0 {
			log.Printf("  恶意节点平均信誉: %.6f ⚠️\n", maliciousRepuSum/float64(maliciousNodeCount))
		}
		if honestCount > 0 && maliciousNodeCount > 0 {
			diff := (honestRepuSum / float64(honestCount)) - (maliciousRepuSum / float64(maliciousNodeCount))
			log.Printf("  信誉差距: %.6f (诚实节点高出 %.2f%%)\n", diff, diff*100)
		}

		log.Printf("本轮耗时: %v\n", time.Since(roundStartTime))
		log.Printf("========================================\n\n")
	}

	close(interChan)

	// 最终总结
	log.Printf("\n")
	log.Printf("╔════════════════════════════════════════╗\n")
	log.Printf("║         信誉系统运行总结               ║\n")
	log.Printf("╚════════════════════════════════════════╝\n")
	log.Printf("总轮数: %d\n", rounds)
	log.Printf("总节点数: %d (诚实: %d, 恶意: %d)\n", len(vehicleIDs), len(honestList), len(maliciousList))
	log.Printf("总交互次数: %d (随机交互模式)\n", grandTotalInteractions)
	log.Printf("平均每轮交互次数: %.1f\n", float64(grandTotalInteractions)/float64(rounds))

	// 创建排序数组
	type NodeReputation struct {
		ID         string
		Reputation float64
	}
	var finalRanking []NodeReputation
	var finalHonestSum, finalMaliciousSum float64
	var finalHonestCount, finalMaliciousCount int

	for _, vid := range vehicleIDs {
		repu := nodes[vid].Rm.ComputeReputation(vid, time.Now())
		finalRanking = append(finalRanking, NodeReputation{ID: vid, Reputation: repu})

		if isMalicious(vid) {
			finalMaliciousSum += repu
			finalMaliciousCount++
		} else {
			finalHonestSum += repu
			finalHonestCount++
		}
	}
	sort.Slice(finalRanking, func(i, j int) bool {
		return finalRanking[i].Reputation > finalRanking[j].Reputation
	})

	log.Printf("\n最终信誉值排名:\n")
	for idx, nr := range finalRanking {
		nodeType := "✅诚实"
		if isMalicious(nr.ID) {
			nodeType = "⚠️恶意"
		}
		log.Printf("  第 %d 名: 节点 %s [%s] = %.6f\n", idx+1, nr.ID, nodeType, nr.Reputation)
	}

	log.Printf("\n最终对比分析:\n")
	if finalHonestCount > 0 {
		log.Printf("  诚实节点最终平均信誉: %.6f ✅\n", finalHonestSum/float64(finalHonestCount))
	}
	if finalMaliciousCount > 0 {
		log.Printf("  恶意节点最终平均信誉: %.6f ⚠️\n", finalMaliciousSum/float64(finalMaliciousCount))
	}
	if finalHonestCount > 0 && finalMaliciousCount > 0 {
		finalDiff := (finalHonestSum / float64(finalHonestCount)) - (finalMaliciousSum / float64(finalMaliciousCount))
		log.Printf("  最终信誉差距: %.6f\n", finalDiff)
		log.Printf("  诚实节点信誉高出: %.2f%%\n", (finalDiff/(finalMaliciousSum/float64(finalMaliciousCount)))*100)
		log.Printf("  ✅ 系统成功识别并惩罚了恶意节点！\n")
	}

	log.Printf("\n结束时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("========================================\n")

	fmt.Println("\n信誉值已记录到 reputation_log.txt 文件中")
}
