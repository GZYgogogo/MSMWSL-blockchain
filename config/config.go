package config

import (
	"encoding/json"
	"os"
)

// Config 定义所有信誉计算参数，可从 JSON 文件加载
// ρ1,ρ2,ρ3: 三权重系数
// Eta, Epsilon: 时效性参数
// Tau1,Tau2: 轨迹相似性权重
// Mu: Pearl 增长曲线调整因子
// Gamma: 不确定性影响系数
// ρ1+ρ2+ρ3=1, Tau1+Tau2=1

type Config struct {
	Rho1    float64 `json:"rho1"`
	Rho2    float64 `json:"rho2"`
	Rho3    float64 `json:"rho3"`
	Eta     float64 `json:"eta"`
	Epsilon float64 `json:"epsilon"`
	Tau1    float64 `json:"tau1"`
	Tau2    float64 `json:"tau2"`
	Tau3    float64 `json:"tau3"`
	Mu      float64 `json:"mu"`
	Gamma   float64 `json:"gamma"`
}

// LoadConfig 从指定路径加载 JSON 配置
func LoadConfig(path string) (Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
