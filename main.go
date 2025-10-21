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

const interactionsPerPair = 10 // 每对节点每轮交互次数

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
	log.Printf("每对节点每轮交互次数: %d\n\n", interactionsPerPair)

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

	for r := 0; r < rounds; r++ {
		roundStartTime := time.Now()
		proposer := nodes[vehicleIDs[r%len(vehicleIDs)]]
		proposer.Propose([]byte(fmt.Sprintf("Round %d positions", r+1)))

		// 记录本轮交互数量
		totalInteractions := 0
		for _, from := range vehicleIDs {
			for _, to := range vehicleIDs {
				if from == to {
					continue
				}
				raw := dataMap[from][r]
				baseTime := time.Now().Add(-time.Duration(raw.Time) * time.Second)
				for k := 0; k < interactionsPerPair; k++ {
					delay := time.Duration(rand.Intn(500)) * time.Millisecond
					ts := baseTime.Add(delay)
					inter := reputation.Interaction{
						From:         from,
						To:           to,
						PosEvents:    1,
						NegEvents:    0,
						Timestamp:    ts,
						TrajUser:     trajMap[from][:r+1],
						TrajProvider: trajMap[to][:r+1],
					}
					wg.Add(1)
					interChan <- inter
					totalInteractions++
				}
			}
		}
		wg.Wait()

		// 输出信誉到控制台和日志
		log.Printf("========================================\n")
		log.Printf("第 %d 轮信誉计算结果\n", r+1)
		log.Printf("----------------------------------------\n")
		log.Printf("提议者节点: %s\n", proposer.ID)
		log.Printf("本轮交互总数: %d\n", totalInteractions)
		log.Printf("----------------------------------------\n")

		fmt.Printf("=== 第 %d 轮信誉计算 ===\n", r+1)

		// 计算并记录每个节点的信誉值
		var minRepu, maxRepu, sumRepu float64 = 1.0, 0.0, 0.0
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

			// 输出到控制台
			fmt.Printf("节点 %s → 信誉值: %.4f\n", vid, repu)

			// 详细记录到日志
			if change != 0 {
				log.Printf("节点 %s: 信誉值=%.6f, 变化=%.6f (%.2f%%)\n",
					vid, repu, change, change*100)
			} else {
				log.Printf("节点 %s: 信誉值=%.6f (首次计算)\n", vid, repu)
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
	log.Printf("总节点数: %d\n", len(vehicleIDs))
	log.Printf("总交互次数: %d\n", rounds*len(vehicleIDs)*(len(vehicleIDs)-1)*interactionsPerPair)
	log.Printf("\n最终信誉值排名:\n")

	// 创建排序数组
	type NodeReputation struct {
		ID         string
		Reputation float64
	}
	var finalRanking []NodeReputation
	for _, vid := range vehicleIDs {
		repu := nodes[vid].Rm.ComputeReputation(vid, time.Now())
		finalRanking = append(finalRanking, NodeReputation{ID: vid, Reputation: repu})
	}
	sort.Slice(finalRanking, func(i, j int) bool {
		return finalRanking[i].Reputation > finalRanking[j].Reputation
	})

	for idx, nr := range finalRanking {
		log.Printf("  第 %d 名: 节点 %s = %.6f\n", idx+1, nr.ID, nr.Reputation)
	}

	log.Printf("\n结束时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("========================================\n")

	fmt.Println("\n信誉值已记录到 reputation_log.txt 文件中")
}
