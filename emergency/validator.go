package emergency

import (
	"block/reputation"
	"sort"
	"time"
)

// Validator 验证器节点
type Validator struct {
	ID         string  // 节点ID
	Reputation float64 // 信誉值
}

// ValidatorGroup 验证器节点组
// 根据论文 3.4.1.3 验证器节点组建
type ValidatorGroup struct {
	Validators   []*Validator // 验证器节点列表
	GroupSize    int          // 验证器组大小 N
	ActivePeriod int          // 验证器组活跃周期（区块周期数）
	CurrentRound int          // 当前区块周期
	CreatedAt    time.Time    // 验证器组创建时间
}

// NewValidatorGroup 创建新的验证器节点组
func NewValidatorGroup(groupSize int, activePeriod int) *ValidatorGroup {
	return &ValidatorGroup{
		Validators:   make([]*Validator, 0),
		GroupSize:    groupSize,
		ActivePeriod: activePeriod,
		CurrentRound: 0,
		CreatedAt:    time.Now(),
	}
}

// SelectValidators 根据信誉值选取验证器节点
// 选取信誉值最高的 groupSize 个节点作为验证器节点
func (vg *ValidatorGroup) SelectValidators(
	nodeIDs []string,
	reputationManagers map[string]*reputation.ReputationManager,
	now time.Time,
) {
	// 计算所有节点的信誉值
	nodeReputation := make([]*Validator, 0)
	for _, nodeID := range nodeIDs {
		rm := reputationManagers[nodeID]
		if rm != nil {
			repu := rm.ComputeReputation(nodeID, now)
			nodeReputation = append(nodeReputation, &Validator{
				ID:         nodeID,
				Reputation: repu,
			})
		}
	}

	// 按信誉值降序排序
	sort.Slice(nodeReputation, func(i, j int) bool {
		return nodeReputation[i].Reputation > nodeReputation[j].Reputation
	})

	// 选取前 groupSize 个节点
	if len(nodeReputation) < vg.GroupSize {
		vg.Validators = nodeReputation
	} else {
		vg.Validators = nodeReputation[:vg.GroupSize]
	}

	vg.CreatedAt = now
	vg.CurrentRound = 0
}

// IsActive 判断验证器组是否仍然活跃
func (vg *ValidatorGroup) IsActive() bool {
	return vg.CurrentRound < vg.ActivePeriod
}

// IncrementRound 增加当前轮数
func (vg *ValidatorGroup) IncrementRound() {
	vg.CurrentRound++
}

// NeedRefresh 判断是否需要重新选择验证器组
func (vg *ValidatorGroup) NeedRefresh() bool {
	// 如果验证器组已经工作了 ActivePeriod 个区块周期，需要刷新
	// 或者如果没有任何验证器节点，也需要刷新
	return !vg.IsActive() || len(vg.Validators) == 0
}

// GetValidatorIDs 获取所有验证器节点的ID列表
func (vg *ValidatorGroup) GetValidatorIDs() []string {
	ids := make([]string, len(vg.Validators))
	for i, v := range vg.Validators {
		ids[i] = v.ID
	}
	return ids
}

// IsValidator 判断节点是否是验证器节点
func (vg *ValidatorGroup) IsValidator(nodeID string) bool {
	for _, v := range vg.Validators {
		if v.ID == nodeID {
			return true
		}
	}
	return false
}

// GetValidator 获取指定ID的验证器节点
func (vg *ValidatorGroup) GetValidator(nodeID string) *Validator {
	for _, v := range vg.Validators {
		if v.ID == nodeID {
			return v
		}
	}
	return nil
}

// GetSize 获取验证器组大小
func (vg *ValidatorGroup) GetSize() int {
	return len(vg.Validators)
}

// SelectProposer 选择出块节点
// 根据信誉值和紧急度选择信誉值最高的节点作为出块者
func (vg *ValidatorGroup) SelectProposer() *Validator {
	if len(vg.Validators) == 0 {
		return nil
	}

	// 选择信誉值最高的验证器节点作为出块者
	proposer := vg.Validators[0]
	for _, v := range vg.Validators {
		if v.Reputation > proposer.Reputation {
			proposer = v
		}
	}

	return proposer
}

// PenalizeInactiveValidators 惩罚不活跃的验证器节点
// 如果验证器节点在 N 个区块周期内没有参与验证，将被移除
func (vg *ValidatorGroup) PenalizeInactiveValidators(
	inactiveValidators []string,
	reputationManagers map[string]*reputation.ReputationManager,
	newCandidates []string,
	now time.Time,
) {
	// 移除不活跃的验证器节点
	activeValidators := make([]*Validator, 0)
	for _, v := range vg.Validators {
		isInactive := false
		for _, inactive := range inactiveValidators {
			if v.ID == inactive {
				isInactive = true
				break
			}
		}
		if !isInactive {
			activeValidators = append(activeValidators, v)
		}
	}

	// 从候选节点中补充新的验证器节点
	needed := vg.GroupSize - len(activeValidators)
	if needed > 0 && len(newCandidates) > 0 {
		// 计算候选节点的信誉值
		candidateReputation := make([]*Validator, 0)
		for _, nodeID := range newCandidates {
			rm := reputationManagers[nodeID]
			if rm != nil {
				repu := rm.ComputeReputation(nodeID, now)
				candidateReputation = append(candidateReputation, &Validator{
					ID:         nodeID,
					Reputation: repu,
				})
			}
		}

		// 按信誉值降序排序
		sort.Slice(candidateReputation, func(i, j int) bool {
			return candidateReputation[i].Reputation > candidateReputation[j].Reputation
		})

		// 补充前 needed 个候选节点
		if len(candidateReputation) < needed {
			needed = len(candidateReputation)
		}
		activeValidators = append(activeValidators, candidateReputation[:needed]...)
	}

	vg.Validators = activeValidators
}
