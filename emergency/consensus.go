package emergency

import (
	"block/reputation"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// MessageType PBFT消息类型
type MessageType int

const (
	PrePrepare MessageType = iota
	Prepare
	Commit
)

// ConsensusMessage PBFT共识消息
type ConsensusMessage struct {
	Type      MessageType     // 消息类型
	BlockHash string          // 区块哈希
	Block     *EmergencyBlock // 紧急区块
	From      string          // 发送者ID
	Timestamp time.Time       // 时间戳
}

// EmergencyNode 紧急区块链节点
type EmergencyNode struct {
	ID                string                        // 节点ID
	IsValidator       bool                          // 是否是验证器节点
	Blockchain        *EmergencyBlockchain          // 紧急区块链
	ReputationManager *reputation.ReputationManager // 信誉管理器
	ValidatorGroup    *ValidatorGroup               // 验证器节点组
	Peers             []*EmergencyNode              // 对等节点
	mutex             sync.Mutex                    // 互斥锁

	// PBFT共识相关
	prePrepareReceived map[string]*ConsensusMessage // PrePrepare消息缓存
	prepareVotes       map[string]map[string]bool   // Prepare投票记录 [blockHash][voterID]
	commitVotes        map[string]map[string]bool   // Commit投票记录 [blockHash][voterID]
}

// NewEmergencyNode 创建新的紧急区块链节点
func NewEmergencyNode(
	id string,
	blockchain *EmergencyBlockchain,
	reputationManager *reputation.ReputationManager,
	validatorGroup *ValidatorGroup,
) *EmergencyNode {
	return &EmergencyNode{
		ID:                 id,
		Blockchain:         blockchain,
		ReputationManager:  reputationManager,
		ValidatorGroup:     validatorGroup,
		Peers:              make([]*EmergencyNode, 0),
		prePrepareReceived: make(map[string]*ConsensusMessage),
		prepareVotes:       make(map[string]map[string]bool),
		commitVotes:        make(map[string]map[string]bool),
	}
}

// SetPeers 设置对等节点
func (en *EmergencyNode) SetPeers(peers []*EmergencyNode) {
	en.Peers = peers
}

// UpdateValidatorStatus 更新节点的验证器状态
func (en *EmergencyNode) UpdateValidatorStatus() {
	en.IsValidator = en.ValidatorGroup.IsValidator(en.ID)
}

// Broadcast 广播消息给所有节点
func (en *EmergencyNode) Broadcast(msg ConsensusMessage) {
	for _, peer := range en.Peers {
		if peer.ID != en.ID {
			go peer.ReceiveMessage(msg)
		}
	}
}

// BroadcastToValidators 广播消息给验证器节点
func (en *EmergencyNode) BroadcastToValidators(msg ConsensusMessage) {
	for _, peer := range en.Peers {
		if peer.ID != en.ID && peer.IsValidator {
			go peer.ReceiveMessage(msg)
		}
	}
}

// ReceiveMessage 接收共识消息
func (en *EmergencyNode) ReceiveMessage(msg ConsensusMessage) {
	en.mutex.Lock()
	defer en.mutex.Unlock()

	switch msg.Type {
	case PrePrepare:
		en.handlePrePrepare(msg)
	case Prepare:
		en.handlePrepare(msg)
	case Commit:
		en.handleCommit(msg)
	}
}

// handlePrePrepare 处理PrePrepare消息
func (en *EmergencyNode) handlePrePrepare(msg ConsensusMessage) {
	// 验证器节点接收PrePrepare消息
	if !en.IsValidator {
		return
	}

	// 验证区块合法性
	if !en.Blockchain.VerifyBlock(msg.Block) {
		fmt.Printf("节点 %s: 验证区块 %s 失败\n", en.ID, msg.BlockHash)
		return
	}

	// 缓存PrePrepare消息
	en.prePrepareReceived[msg.BlockHash] = &msg

	// 发送Prepare消息
	prepareMsg := ConsensusMessage{
		Type:      Prepare,
		BlockHash: msg.BlockHash,
		Block:     msg.Block,
		From:      en.ID,
		Timestamp: time.Now(),
	}
	en.BroadcastToValidators(prepareMsg)
}

// handlePrepare 处理Prepare消息
func (en *EmergencyNode) handlePrepare(msg ConsensusMessage) {
	// 验证器节点接收Prepare消息
	if !en.IsValidator {
		return
	}

	// 记录Prepare投票
	if _, exists := en.prepareVotes[msg.BlockHash]; !exists {
		en.prepareVotes[msg.BlockHash] = make(map[string]bool)
	}
	en.prepareVotes[msg.BlockHash][msg.From] = true

	// 检查是否收到足够的Prepare消息（超过 f+1 个）
	// 在拜占庭容错中，f = (N-1)/3，N是验证器总数
	N := en.ValidatorGroup.GetSize()
	f := (N - 1) / 3
	requiredVotes := f + 1

	if len(en.prepareVotes[msg.BlockHash]) >= requiredVotes {
		// 发送Commit消息
		commitMsg := ConsensusMessage{
			Type:      Commit,
			BlockHash: msg.BlockHash,
			Block:     msg.Block,
			From:      en.ID,
			Timestamp: time.Now(),
		}
		en.BroadcastToValidators(commitMsg)
	}
}

// handleCommit 处理Commit消息
func (en *EmergencyNode) handleCommit(msg ConsensusMessage) {
	// 记录Commit投票
	if _, exists := en.commitVotes[msg.BlockHash]; !exists {
		en.commitVotes[msg.BlockHash] = make(map[string]bool)
	}
	en.commitVotes[msg.BlockHash][msg.From] = true

	// 检查是否收到足够的Commit消息（超过 2f+1 个）
	N := en.ValidatorGroup.GetSize()
	f := (N - 1) / 3
	requiredVotes := 2*f + 1

	if len(en.commitVotes[msg.BlockHash]) >= requiredVotes {
		// 将区块添加到区块链
		en.Blockchain.AddBlock(msg.Block)
		fmt.Printf("节点 %s: 区块 %d 已确认并添加到紧急区块链\n", en.ID, msg.Block.Index)

		// ⭐ 新增：记录紧急交易的信誉交互
		en.recordEmergencyInteractions(msg.Block)

		// 清理投票记录
		delete(en.prePrepareReceived, msg.BlockHash)
		delete(en.prepareVotes, msg.BlockHash)
		delete(en.commitVotes, msg.BlockHash)
	}
}

// recordEmergencyInteractions 记录紧急区块中交易的信誉交互
// 验证器节点验证紧急交易后，给交易发送者评价
func (en *EmergencyNode) recordEmergencyInteractions(block *EmergencyBlock) {
	// 只有验证器节点才记录信誉交互
	if !en.IsValidator {
		return
	}

	// 为区块中的每笔紧急交易创建信誉交互
	for _, tx := range block.Transactions {
		// 验证器（当前节点）作为评价者，交易发送者作为被评价者
		// 假设紧急交易都是合法的（已经通过验证），给予正面评价
		// 如果发现恶意交易，可以给负面评价

		// 随机模拟验证结果（实际中应该是真实的验证逻辑）
		// 90%概率是诚实交易，10%概率是恶意交易
		var posEvents, negEvents int
		if rand.Float64() < 0.9 {
			posEvents = 1
			negEvents = 0
		} else {
			posEvents = 0
			negEvents = 1
		}

		// 创建紧急交易类型的信誉交互
		inter := reputation.Interaction{
			From:          en.ID,        // 验证器节点（评价者）
			To:            tx.VehicleID, // 交易发送者（被评价者）
			PosEvents:     posEvents,
			NegEvents:     negEvents,
			Timestamp:     time.Now(),
			TrajUser:      []reputation.Vector{}, // 可以从节点轨迹数据中获取
			TrajProvider:  []reputation.Vector{},
			TxType:        reputation.EmergencyTransaction, // ⭐ 标记为紧急交易
			UrgencyDegree: tx.UrgencyDegree,                // ⭐ 记录紧急度
		}

		// 添加到信誉管理器
		en.ReputationManager.AddInteraction(inter)

		fmt.Printf("  验证器 %s 对紧急交易 %s 的发送者 %s 进行评价 (紧急度=%.2f, 正面=%d, 负面=%d)\n",
			en.ID, tx.ID, tx.VehicleID, tx.UrgencyDegree, posEvents, negEvents)
	}
}

// ProposeEmergencyBlock 提议新的紧急区块（仅验证器节点）
// 根据论文 3.4.1.4 紧急区块生成
func (en *EmergencyNode) ProposeEmergencyBlock() {
	en.mutex.Lock()
	defer en.mutex.Unlock()

	// 只有验证器节点才能提议区块
	if !en.IsValidator {
		return
	}

	// 检查交易池中是否有足够的交易
	if en.Blockchain.TxPool.Size() == 0 {
		return
	}

	// 从交易池中获取紧急度最高的 k 笔交易
	transactions := en.Blockchain.TxPool.GetTopKTransactions(en.Blockchain.BlockSize)
	if len(transactions) == 0 {
		return
	}

	// 创建新区块
	latestBlock := en.Blockchain.GetLatestBlock()
	newBlock := NewEmergencyBlock(
		latestBlock.Index+1,
		latestBlock.Hash,
		transactions,
		en.ValidatorGroup.GetValidatorIDs(),
	)

	fmt.Printf("验证器节点 %s: 提议紧急区块 %d (包含 %d 笔交易, 总紧急度=%.2f)\n",
		en.ID, newBlock.Index, len(newBlock.Transactions), newBlock.TotalUrgency)

	// 发送PrePrepare消息给所有验证器节点
	prePrepareMsg := ConsensusMessage{
		Type:      PrePrepare,
		BlockHash: newBlock.Hash,
		Block:     newBlock,
		From:      en.ID,
		Timestamp: time.Now(),
	}
	en.BroadcastToValidators(prePrepareMsg)

	// 自己也处理这个消息
	en.handlePrePrepare(prePrepareMsg)
}

// AddEmergencyTransaction 添加紧急交易（所有节点）
func (en *EmergencyNode) AddEmergencyTransaction(tx *EmergencyTransaction) {
	en.mutex.Lock()
	defer en.mutex.Unlock()

	en.Blockchain.AddTransaction(tx)

	// 广播交易到所有节点
	fmt.Printf("节点 %s: 收到紧急交易 %s (紧急度=%.4f)\n", en.ID, tx.ID, tx.UrgencyDegree)
}

// GetReputation 获取节点信誉值
func (en *EmergencyNode) GetReputation() float64 {
	return en.ReputationManager.ComputeReputation(en.ID, time.Now())
}

// GetBlockchainLength 获取紧急区块链长度
func (en *EmergencyNode) GetBlockchainLength() int {
	return en.Blockchain.GetChainLength()
}

// PrintBlockchain 打印紧急区块链信息
func (en *EmergencyNode) PrintBlockchain() {
	en.mutex.Lock()
	defer en.mutex.Unlock()

	fmt.Printf("\n=== 节点 %s 的紧急区块链 ===\n", en.ID)
	for _, block := range en.Blockchain.Chain {
		fmt.Printf("区块 %d: Hash=%s, TxCount=%d, TotalUrgency=%.2f\n",
			block.Index, block.Hash[:8], len(block.Transactions), block.TotalUrgency)
	}
	fmt.Printf("===========================\n\n")
}
