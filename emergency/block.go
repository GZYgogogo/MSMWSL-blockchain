package emergency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// EmergencyBlock 紧急区块结构
// 根据论文 3.4.1.2 紧急区块结构设计
type EmergencyBlock struct {
	// 区块头
	Index        int       // 区块高度
	Timestamp    time.Time // 时间戳
	PrevHash     string    // 父区块哈希
	Hash         string    // 当前区块哈希
	MerkleRoot   string    // 默克尔根
	Signature    string    // 数字签名
	ValidatorIDs []string  // 参与验证的验证器节点ID列表

	// 区块体
	Transactions []*EmergencyTransaction // k 笔按时间顺序排列的紧急交易
	TotalUrgency float64                 // 总紧急度 ED^total = ∑ED_i
}

// CalculateMerkleRoot 计算默克尔根
func (b *EmergencyBlock) CalculateMerkleRoot() string {
	if len(b.Transactions) == 0 {
		return ""
	}

	// 简化的默克尔树实现：将所有交易ID连接后哈希
	var txIDs string
	for _, tx := range b.Transactions {
		txIDs += tx.ID
	}

	hash := sha256.Sum256([]byte(txIDs))
	return hex.EncodeToString(hash[:])
}

// CalculateTotalUrgency 计算区块总紧急度
func (b *EmergencyBlock) CalculateTotalUrgency() float64 {
	var total float64
	for _, tx := range b.Transactions {
		total += tx.UrgencyDegree
	}
	return total
}

// CalculateHash 计算区块哈希
func (b *EmergencyBlock) CalculateHash() string {
	// 将区块头信息序列化
	blockData := struct {
		Index      int
		Timestamp  string
		PrevHash   string
		MerkleRoot string
	}{
		Index:      b.Index,
		Timestamp:  b.Timestamp.Format(time.RFC3339Nano),
		PrevHash:   b.PrevHash,
		MerkleRoot: b.MerkleRoot,
	}

	jsonData, _ := json.Marshal(blockData)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// NewEmergencyBlock 创建新的紧急区块
func NewEmergencyBlock(
	index int,
	prevHash string,
	transactions []*EmergencyTransaction,
	validatorIDs []string,
) *EmergencyBlock {
	block := &EmergencyBlock{
		Index:        index,
		Timestamp:    time.Now(),
		PrevHash:     prevHash,
		Transactions: transactions,
		ValidatorIDs: validatorIDs,
	}

	// 计算默克尔根
	block.MerkleRoot = block.CalculateMerkleRoot()

	// 计算总紧急度
	block.TotalUrgency = block.CalculateTotalUrgency()

	// 计算区块哈希
	block.Hash = block.CalculateHash()

	return block
}

// EmergencyBlockchain 紧急区块链
type EmergencyBlockchain struct {
	Chain       []*EmergencyBlock // 紧急区块链
	TxPool      *TransactionPool  // 交易池
	UrgencyCfg  UrgencyConfig     // 紧急度配置
	BlockSize   int               // 每个区块包含的交易数量 k
	BlockPeriod time.Duration     // 出块周期（例如 kms）
}

// NewEmergencyBlockchain 创建新的紧急区块链
func NewEmergencyBlockchain(urgencyCfg UrgencyConfig, blockSize int, blockPeriod time.Duration) *EmergencyBlockchain {
	// 创建创世区块
	genesisBlock := &EmergencyBlock{
		Index:        0,
		Timestamp:    time.Now(),
		PrevHash:     "0",
		Hash:         "genesis",
		MerkleRoot:   "",
		Transactions: make([]*EmergencyTransaction, 0),
		TotalUrgency: 0,
		ValidatorIDs: []string{},
	}

	return &EmergencyBlockchain{
		Chain:       []*EmergencyBlock{genesisBlock},
		TxPool:      NewTransactionPool(),
		UrgencyCfg:  urgencyCfg,
		BlockSize:   blockSize,
		BlockPeriod: blockPeriod,
	}
}

// AddTransaction 添加紧急交易到交易池
func (ebc *EmergencyBlockchain) AddTransaction(tx *EmergencyTransaction) {
	ebc.TxPool.AddTransaction(tx)
}

// GetLatestBlock 获取最新区块
func (ebc *EmergencyBlockchain) GetLatestBlock() *EmergencyBlock {
	if len(ebc.Chain) == 0 {
		return nil
	}
	return ebc.Chain[len(ebc.Chain)-1]
}

// AddBlock 添加新区块到链
func (ebc *EmergencyBlockchain) AddBlock(block *EmergencyBlock) {
	ebc.Chain = append(ebc.Chain, block)
}

// GetChainLength 获取区块链长度
func (ebc *EmergencyBlockchain) GetChainLength() int {
	return len(ebc.Chain)
}

// VerifyBlock 验证区块合法性
func (ebc *EmergencyBlockchain) VerifyBlock(block *EmergencyBlock) bool {
	// 1. 验证区块高度
	latestBlock := ebc.GetLatestBlock()
	if block.Index != latestBlock.Index+1 {
		return false
	}

	// 2. 验证前一个区块哈希
	if block.PrevHash != latestBlock.Hash {
		return false
	}

	// 3. 验证默克尔根
	expectedMerkleRoot := block.CalculateMerkleRoot()
	if block.MerkleRoot != expectedMerkleRoot {
		return false
	}

	// 4. 验证区块哈希
	expectedHash := block.CalculateHash()
	if block.Hash != expectedHash {
		return false
	}

	// 5. 验证总紧急度
	expectedTotalUrgency := block.CalculateTotalUrgency()
	if block.TotalUrgency != expectedTotalUrgency {
		return false
	}

	return true
}
