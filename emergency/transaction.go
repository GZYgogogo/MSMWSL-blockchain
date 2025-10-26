package emergency

import (
	"math"
	"time"
)

// EmergencyTransaction 紧急交易结构
type EmergencyTransaction struct {
	ID            string    // 交易ID
	VehicleID     string    // 车辆ID（发送者）
	Data          []byte    // 交易数据
	Timestamp     time.Time // 交易生成时间
	ProductTime   time.Time // 交易产生时间 tp
	DeadlineTime  time.Time // 交易期望完成时间 td
	ArrivalTime   time.Time // 交易到达RSU时间 ta
	Priority      int       // 车辆优先级
	UrgencyDegree float64   // 紧急度 ED
	Theta         int       // 车辆在此期间已申请的紧急交易数量
}

// UrgencyConfig 紧急度计算配置
type UrgencyConfig struct {
	Omega float64 // ω: 已申请紧急交易数量的影响权重
}

// CalculateUrgencyDegree 计算紧急交易的紧急度
// 根据公式 (3-13): ED = E × e^(ωθ)
// 其中 E 根据公式 (3-14): E = e^(-Tc/(Tr-Tu))
// Tc: 交易期望延迟 = td - ta
// Tu: 交易产生时间 tp
// Tr: 交易到达RSU时间 ta
func (tx *EmergencyTransaction) CalculateUrgencyDegree(cfg UrgencyConfig) {
	// 计算 Tc (期望延迟)
	Tc := tx.DeadlineTime.Sub(tx.ArrivalTime).Seconds()

	// 计算 Tr - Tu
	TrMinusTu := tx.ArrivalTime.Sub(tx.ProductTime).Seconds()

	// 计算 E = e^(-Tc/(Tr-Tu))
	var E float64
	if TrMinusTu > 0 {
		E = math.Exp(-Tc / TrMinusTu)
	} else {
		// 如果 Tr - Tu <= 0，说明时间参数异常，设置较低紧急度
		E = 0.1
	}

	// 计算 ED = E × e^(ωθ)
	theta := float64(tx.Theta)
	tx.UrgencyDegree = E * math.Exp(cfg.Omega*theta)
}

// NewEmergencyTransaction 创建新的紧急交易
func NewEmergencyTransaction(
	id string,
	vehicleID string,
	data []byte,
	productTime time.Time,
	deadlineTime time.Time,
	arrivalTime time.Time,
	theta int,
	cfg UrgencyConfig,
) *EmergencyTransaction {
	tx := &EmergencyTransaction{
		ID:           id,
		VehicleID:    vehicleID,
		Data:         data,
		Timestamp:    time.Now(),
		ProductTime:  productTime,
		DeadlineTime: deadlineTime,
		ArrivalTime:  arrivalTime,
		Theta:        theta,
	}

	// 计算紧急度
	tx.CalculateUrgencyDegree(cfg)

	return tx
}

// TransactionPool 交易池，用于存储待处理的紧急交易
type TransactionPool struct {
	transactions []*EmergencyTransaction
}

// NewTransactionPool 创建新的交易池
func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		transactions: make([]*EmergencyTransaction, 0),
	}
}

// AddTransaction 添加交易到交易池
func (pool *TransactionPool) AddTransaction(tx *EmergencyTransaction) {
	pool.transactions = append(pool.transactions, tx)
}

// GetTopKTransactions 获取紧急度最高的 k 笔交易
func (pool *TransactionPool) GetTopKTransactions(k int) []*EmergencyTransaction {
	if len(pool.transactions) == 0 {
		return nil
	}

	// 按紧急度降序排序
	sorted := make([]*EmergencyTransaction, len(pool.transactions))
	copy(sorted, pool.transactions)

	// 简单冒泡排序（实际应用中可使用更高效的排序算法）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].UrgencyDegree < sorted[j+1].UrgencyDegree {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	// 取前 k 笔
	if k > len(sorted) {
		k = len(sorted)
	}

	result := sorted[:k]

	// 从交易池中移除已选中的交易
	pool.RemoveTransactions(result)

	return result
}

// RemoveTransactions 从交易池中移除指定的交易
func (pool *TransactionPool) RemoveTransactions(txs []*EmergencyTransaction) {
	// 创建一个 map 用于快速查找
	toRemove := make(map[string]bool)
	for _, tx := range txs {
		toRemove[tx.ID] = true
	}

	// 保留未被移除的交易
	newTransactions := make([]*EmergencyTransaction, 0)
	for _, tx := range pool.transactions {
		if !toRemove[tx.ID] {
			newTransactions = append(newTransactions, tx)
		}
	}

	pool.transactions = newTransactions
}

// Size 返回交易池大小
func (pool *TransactionPool) Size() int {
	return len(pool.transactions)
}
