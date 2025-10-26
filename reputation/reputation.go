package reputation

import (
	"block/config"
	"fmt"
	"math"
	"time"
)

// Vector 表示轨迹点（速度、方向、加速度）
type Vector struct {
	Speed        float64
	Direction    float64
	Acceleration float64
}

// TransactionType 交易类型
type TransactionType int

const (
	NormalTransaction    TransactionType = iota // 普通交易
	EmergencyTransaction                        // 紧急交易
)

// Interaction 表示一次交互事件
type Interaction struct {
	From          string          // 交互发起者
	To            string          // 交互接收者
	PosEvents     int             // 正面事件数量
	NegEvents     int             // 负面事件数量
	Timestamp     time.Time       // 事件发生时间
	TrajUser      []Vector        // 信任者轨迹
	TrajProvider  []Vector        // 被信任者轨迹
	TxType        TransactionType // 交易类型（普通/紧急）
	UrgencyDegree float64         // 紧急度（仅紧急交易有效）
}

// SubjectiveOpinion 主观意见三元组
type SubjectiveOpinion struct {
	T float64 // 信任度
	D float64 // 否定度
	I float64 // 不确定度
}

// DirectOpinion 直接意见及权重
type DirectOpinion struct {
	Opinion SubjectiveOpinion // 主观意见
	Weight  float64           // 直接权重 δ
}

// 初始信誉值常量
const InitialReputation = 0.5

// 信誉影响权重常量
const (
	// 普通交易的基础权重
	NormalTxWeight = 1.0

	// 紧急交易的基础权重（是普通交易的3倍，提升惩罚力度）
	EmergencyTxBaseWeight = 3.0

	// 紧急度影响系数（紧急度越高，影响越大，提升至0.8加强效果）
	UrgencyImpactFactor = 0.8

	// 最大权重倍数（提升至8.0，加强对恶意节点的惩罚）
	MaxWeightMultiplier = 8.0
)

// ReputationManager 管理信誉计算
type ReputationManager struct {
	cfg          config.Config
	interactions []Interaction
}

// NewReputationManager 创建管理器
func NewReputationManager(cfg config.Config) *ReputationManager {
	return &ReputationManager{cfg: cfg}
}

// AddInteraction 添加交互记录
func (rm *ReputationManager) AddInteraction(inter Interaction) {
	rm.interactions = append(rm.interactions, inter)
}

// CalculateTransactionWeight 计算交易类型对信誉的影响权重
// 公式设计：
// - 普通交易: W = 1.0
// - 紧急交易: W = EmergencyTxBaseWeight × (1 + UrgencyImpactFactor × UrgencyDegree)
// - 上限约束: W ≤ MaxWeightMultiplier
func CalculateTransactionWeight(txType TransactionType, urgencyDegree float64) float64 {
	var weight float64

	switch txType {
	case NormalTransaction:
		// 普通交易基础权重
		weight = NormalTxWeight

	case EmergencyTransaction:
		// 紧急交易权重 = 基础权重 × (1 + 紧急度影响)
		// 紧急度越高，权重越大
		weight = EmergencyTxBaseWeight * (1.0 + UrgencyImpactFactor*urgencyDegree)

		// 应用上限约束
		if weight > MaxWeightMultiplier {
			weight = MaxWeightMultiplier
		}

	default:
		weight = NormalTxWeight
	}

	return weight
}

// ComputeReputation 计算最终信誉值
func (rm *ReputationManager) ComputeReputation(target string, now time.Time) float64 {
	agg := rm.aggregateByPair()

	// 如果目标节点没有任何交互记录，返回初始信誉值
	if _, exists := agg[target]; !exists {
		return InitialReputation
	}

	direct := rm.computeDirectOpinions(agg, now)
	indirect := rm.computeIndirectOpinions(direct)
	final := rm.fuseOpinions(direct[target], indirect[target])
	return final.T + rm.cfg.Gamma*final.I
}

// aggregateByPair 聚合交互按 (To,From)
func (rm *ReputationManager) aggregateByPair() map[string]map[string]Interaction {
	agg := make(map[string]map[string]Interaction)
	for _, inter := range rm.interactions {
		if _, ok := agg[inter.To]; !ok {
			agg[inter.To] = make(map[string]Interaction)
		}
		exist, ok := agg[inter.To][inter.From]
		if !ok {
			agg[inter.To][inter.From] = inter
		} else {
			exist.PosEvents += inter.PosEvents
			exist.NegEvents += inter.NegEvents
			if inter.Timestamp.After(exist.Timestamp) {
				exist.Timestamp = inter.Timestamp
				exist.TrajUser = inter.TrajUser
				exist.TrajProvider = inter.TrajProvider
			}
			agg[inter.To][inter.From] = exist
		}
	}
	return agg
}

// computeDirectOpinions 计算每对节点的直接意见和权重，并输出调试信息
type directOpinionsMap map[string]map[string]DirectOpinion

func (rm *ReputationManager) computeDirectOpinions(
	agg map[string]map[string]Interaction,
	now time.Time,
) directOpinionsMap {
	direct := make(directOpinionsMap)
	for to, fromMap := range agg {
		// 计算平均事件数
		var sumCnt float64
		for _, inter := range fromMap {
			sumCnt += float64(inter.PosEvents + inter.NegEvents)
		}
		avgCnt := 1.0
		if len(fromMap) > 0 {
			avgCnt = sumCnt / float64(len(fromMap))
		}
		// 计算 θ
		var errNum, errDen float64
		tmp := make(map[string]DirectOpinion)
		for from, inter := range fromMap {
			Fi := float64(inter.PosEvents+inter.NegEvents) / avgCnt
			delta := now.Sub(inter.Timestamp).Seconds()
			fmt.Printf("DEBUG now=%s inter.Timestamp=%s \n", now.Format("2006-01-02 15:04:05"), inter.Timestamp.Format("2006-01-02 15:04:05"))
			var TIM float64
			if delta <= 0 {
				// TODO: 目前每轮所有节点都是delta < 0
				// TIM == 1
				TIM = rm.cfg.Eta
			} else {
				TIM = rm.cfg.Eta * math.Pow(delta, -rm.cfg.Epsilon)
			}
			sim := rm.computeTrajectorySimilarity(inter.TrajUser, inter.TrajProvider)

			// 原始权重计算
			baseWeight := rm.cfg.Rho1*Fi + rm.cfg.Rho2*TIM + rm.cfg.Rho3*sim

			// ⭐ 新增：计算交易类型影响权重
			txWeight := CalculateTransactionWeight(inter.TxType, inter.UrgencyDegree)

			// ⭐ 最终权重 = 原始权重 × 交易类型权重
			weight := baseWeight * txWeight

			// 修改：不确定度由交互次数决定，而不是轨迹相似度
			totalEvents := float64(inter.PosEvents + inter.NegEvents)
			Ii := 2.0 / (2.0 + totalEvents)

			// 调试输出（增加交易类型和权重信息）
			txTypeStr := "Normal"
			if inter.TxType == EmergencyTransaction {
				txTypeStr = "Emergency"
			}
			fmt.Printf("DEBUG Direct: to=%s from=%s delta=%.3f TIM=%.3f sim=%.3f baseWeight=%.3f txType=%s txWeight=%.3f finalWeight=%.3f totalEvents=%.0f Ii=%.3f\n",
				to, from, delta, TIM, sim, baseWeight, txTypeStr, txWeight, weight, totalEvents, Ii)

			tmp[from] = DirectOpinion{Opinion: SubjectiveOpinion{I: Ii}, Weight: weight}
			errNum += weight * float64(inter.NegEvents)
			errDen += weight
		}
		theta := 0.0
		if errDen != 0 {
			theta = rm.cfg.Mu / (1 + math.Exp(errNum/errDen))
		}
		// 填充 Opinion.T 和 Opinion.D，并调试
		direct[to] = make(map[string]DirectOpinion)
		for from, inter := range fromMap {
			d := tmp[from]
			alpha := (1 - theta) * float64(inter.PosEvents)
			beta := theta * float64(inter.NegEvents)
			sumEvt := alpha + beta
			if sumEvt > 0 {
				d.Opinion.T = (1 - d.Opinion.I) * alpha / sumEvt
				d.Opinion.D = (1 - d.Opinion.I) * beta / sumEvt
			}
			// fmt.Printf("DEBUG T/D: to=%s from=%s T=%.3f D=%.3f theta=%.3f\n", to, from, d.Opinion.T, d.Opinion.D, theta)
			direct[to][from] = d
		}
	}
	return direct
}

// computeIndirectOpinions 基于直接意见生成多跳间接意见
func (rm *ReputationManager) computeIndirectOpinions(
	direct directOpinionsMap,
) map[string]map[string]SubjectiveOpinion {
	// 最多允许 hopCount 条边（即 hopCount+1 个节点），可根据需要调整或从 cfg 中读取
	const hopCount = 2

	indirect := make(map[string]map[string]SubjectiveOpinion)
	// 辅助函数：判断 slice 中是否包含元素 s
	contains := func(slice []string, s string) bool {
		for _, v := range slice {
			if v == s {
				return true
			}
		}
		return false
	}

	for target, _ := range direct {
		indirect[target] = make(map[string]SubjectiveOpinion)
		// 对每个可能的 source 节点
		for source := range direct {
			if source == target {
				continue
			}
			// 收集所有从 source 到 target 的路径
			var paths [][]string
			var dfs func(path []string)
			dfs = func(path []string) {
				last := path[len(path)-1]
				// 如果超过 hopCount 条边，就返回
				if len(path)-1 > hopCount {
					return
				}
				// 找到一条以 target 结尾的路径，且非直接源（len(path)>1）
				if last == target && len(path) > 1 {
					p := make([]string, len(path))
					copy(p, path)
					paths = append(paths, p)
					return
				}
				// 否则继续沿 direct[last] 的邻居扩展
				for next := range direct[last] {
					if contains(path, next) {
						continue // 避免环路
					}
					dfs(append(path, next))
				}
			}
			dfs([]string{source})

			// 对每条路径做折扣运算并累加
			var sumW float64
			for _, path := range paths {
				// 路径示例: [source, m1, ..., target]
				// 初始化为路径起点
				T, D, I := 1.0, 0.0, 0.0
				w := 1.0
				// 遍历路径上的每一条边
				for i := 0; i < len(path)-1; i++ {
					from := path[i]
					toNode := path[i+1]
					// directOpinionsMap 是映射 direct[toNode][from]
					d := direct[toNode][from]
					// 折扣算子（discounting）：
					Tnew := T * d.Opinion.T
					Dnew := T * d.Opinion.D
					Inew := D + I + T*d.Opinion.I
					T, D, I = Tnew, Dnew, Inew
					w *= d.Weight
				}
				// 累加加权意见
				agg := indirect[target][source]
				agg.T += T * w
				agg.D += D * w
				agg.I += I * w
				indirect[target][source] = agg
				sumW += w
			}
			// 归一化
			if sumW > 0 {
				v := indirect[target][source]
				v.T /= sumW
				v.D /= sumW
				v.I /= sumW
				indirect[target][source] = v
			}
		}
	}
	return indirect
}

// fuseOpinions 融合直接与间接意见
func (rm *ReputationManager) fuseOpinions(
	dir map[string]DirectOpinion,
	ind map[string]SubjectiveOpinion,
) SubjectiveOpinion {
	// 直接聚合
	var sumW float64
	var sumTdir, sumDdir, sumIdir float64
	for _, d := range dir {
		sumW += d.Weight
		sumTdir += d.Opinion.T * d.Weight
		sumDdir += d.Opinion.D * d.Weight
		sumIdir += d.Opinion.I * d.Weight
	}
	Tdir, Ddir, Idir := 0.0, 0.0, 0.0
	if sumW > 0 {
		Tdir = sumTdir / sumW
		Ddir = sumDdir / sumW
		Idir = sumIdir / sumW
	}
	// 若无间接意见，直接返回
	if len(ind) == 0 {
		return SubjectiveOpinion{T: Tdir, D: Ddir, I: Idir}
	}
	// 间接聚合
	var sumTind, sumDind, sumIind float64
	for _, opin := range ind {
		sumTind += opin.T
		sumDind += opin.D
		sumIind += opin.I
	}
	Tind, Dind, Iind := 0.0, 0.0, 0.0
	if len(ind) > 0 {
		Tind = sumTind / float64(len(ind))
		Dind = sumDind / float64(len(ind))
		Iind = sumIind / float64(len(ind))
	}
	// 共识算子融合 - 按照论文公式(13)
	// k = I^dir_C * I^ind_C + T^ind_C * I^dir_C + D^ind_C * I^dir_C
	k := Idir*Iind + Tind*Idir + Dind*Idir
	return SubjectiveOpinion{
		T: (Tdir*Iind + Tind*Idir) / k,
		D: (Ddir*Iind + Dind*Idir) / k,
		I: (Idir * Iind) / k,
	}
}

// computeTrajectorySimilarity 计算轨迹相似度：速度、方向、加速度三分量
func (rm *ReputationManager) computeTrajectorySimilarity(user, prov []Vector) float64 {
	n := len(user)
	if len(prov) < n {
		n = len(prov)
	}
	var uspd, vspd, udir, vdir, uacc, vacc []float64
	for i := 0; i < n; i++ {
		uspd = append(uspd, user[i].Speed)
		vspd = append(vspd, prov[i].Speed)
		udir = append(udir, user[i].Direction)
		vdir = append(vdir, prov[i].Direction)
		uacc = append(uacc, user[i].Acceleration)
		vacc = append(vacc, prov[i].Acceleration)
	}
	sspd := cosineSimilarity(uspd, vspd)
	sdir := cosineSimilarity(udir, vdir)
	sacc := cosineSimilarity(uacc, vacc)
	// fmt.Println("DEBUG Trajectory: sspd=", sspd, "sdir=", sdir, "sacc=", sacc)
	// 三者加权融合，使用配置中的 Tau1、Tau2、Tau3
	return rm.cfg.Tau1*sspd + rm.cfg.Tau2*sdir + rm.cfg.Tau3*sacc
}

// cosineSimilarity 保持不变
func cosineSimilarity(a, b []float64) float64 {
	var num, sa, sb float64
	for i := range a {
		num += a[i] * b[i]
		sa += a[i] * a[i]
	}
	for _, v := range b {
		sb += v * v
	}
	if sa == 0 || sb == 0 {
		return 0
	}
	return num / (math.Sqrt(sa) * math.Sqrt(sb))
}
