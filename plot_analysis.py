import json
import matplotlib.pyplot as plt
import numpy as np
from collections import defaultdict
import os

# 设置中文字体
plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'Arial Unicode MS']
plt.rcParams['axes.unicode_minus'] = False

# 确保plots目录存在
os.makedirs('plots', exist_ok=True)

# 读取数据
with open('performance_results.json', 'r', encoding='utf-8') as f:
    data = json.load(f)

# 组织数据：按 tx_per_block 分组
data_by_tx = defaultdict(list)
for item in data:
    data_by_tx[item['tx_per_block']].append(item)

# 获取所有唯一值
tx_per_block_values = sorted(data_by_tx.keys())
block_intervals = sorted(list(set(item['block_interval_ms'] for item in data)))
node_counts = sorted(list(set(item['node_count'] for item in data)))

# 颜色映射（不同出块间隙）
colors = {
    300: '#1f77b4',   # 蓝色
    600: '#ff7f0e',   # 橙色
    900: '#2ca02c',   # 绿色
    1200: '#d62728'   # 红色
}

# 为每个 tx_per_block 创建两张图
for tx_count in tx_per_block_values:
    current_data = data_by_tx[tx_count]

    # ==================== 第一类图：吞吐量（均值柱状+误差棒） ====================
    fig, ax = plt.subplots(figsize=(12, 7))

    x_base = np.arange(len(node_counts))  # 基础x索引
    n_intervals = len(block_intervals)
    group_width = 0.8
    interval_slot_width = group_width / n_intervals
    bar_width = interval_slot_width / 2.0

    legend_handles = []
    legend_labels = []

    for idx_interval, interval in enumerate(block_intervals):
        means_emergency, stds_emergency = [], []
        means_normal, stds_normal = [], []

        for node in node_counts:
            vals_e = [d['throughput_tps'] for d in current_data
                      if d['block_interval_ms'] == interval
                      and d['chain_type'] == 'emergency'
                      and d['node_count'] == node]
            vals_n = [d['throughput_tps'] for d in current_data
                      if d['block_interval_ms'] == interval
                      and d['chain_type'] == 'normal'
                      and d['node_count'] == node]

            means_emergency.append(np.mean(vals_e) if vals_e else np.nan)
            stds_emergency.append(np.std(vals_e) if vals_e else np.nan)
            means_normal.append(np.mean(vals_n) if vals_n else np.nan)
            stds_normal.append(np.std(vals_n) if vals_n else np.nan)

        interval_center_offset = (idx_interval - (n_intervals - 1) / 2.0) * interval_slot_width
        x_pos_emergency = x_base + interval_center_offset - bar_width / 2.0
        x_pos_normal = x_base + interval_center_offset + bar_width / 2.0

        bars_e = ax.bar(
            x_pos_emergency, means_emergency, yerr=stds_emergency,
            capsize=4, width=bar_width,
            color=colors[interval], alpha=0.9,
            edgecolor='black', linewidth=0.8,
            label=f'紧急链 - {interval}ms'
        )
        bars_n = ax.bar(
            x_pos_normal, means_normal, yerr=stds_normal,
            capsize=4, width=bar_width,
            color=colors[interval], alpha=0.4,
            edgecolor='black', linewidth=0.8,
            hatch='//', label=f'普通链 - {interval}ms'
        )

        for h, lab in [(bars_e, f'紧急链 - {interval}ms'), (bars_n, f'普通链 - {interval}ms')]:
            if lab not in legend_labels:
                legend_handles.append(h)
                legend_labels.append(lab)

    # 横坐标节点数量 ×5
    scaled_nodes = [n * 5 for n in node_counts]
    ax.set_xticks(x_base)
    ax.set_xticklabels([str(n) for n in scaled_nodes], fontsize=12)

    ax.set_xlabel('节点数量', fontsize=14, fontweight='bold')
    ax.set_ylabel('吞吐量 (TPS)', fontsize=14, fontweight='bold')
    ax.set_title(f'吞吐量对比 (交易数量: {tx_count})', fontsize=16, fontweight='bold')

    ax.grid(True, axis='y', alpha=0.3, linestyle='--')
    ax.tick_params(labelsize=12)

    y_min, y_max = ax.get_ylim()
    ax.set_ylim(y_min, y_max * 1.1)

    ax.legend(
        legend_handles, legend_labels,
        loc='upper center', bbox_to_anchor=(0.5, 1.25),
        bbox_transform=ax.transAxes, fontsize=10,
        ncol=4, frameon=True, framealpha=0.9, borderpad=0.4
    )

    plt.tight_layout(rect=[0, 0, 1, 0.8])
    plt.savefig(f'plots/throughput_comparison_tx{tx_count}.png', dpi=300, bbox_inches='tight')
    print(f'已保存: plots/throughput_comparison_tx{tx_count}.png')
    plt.close()

    # ==================== 第二类图：延迟（保持原样，仅横坐标×5） ====================
    fig, ax = plt.subplots(figsize=(12, 7))

    for interval in block_intervals:
        emergency_data = [d for d in current_data
                          if d['block_interval_ms'] == interval and d['chain_type'] == 'emergency']
        normal_data = [d for d in current_data
                       if d['block_interval_ms'] == interval and d['chain_type'] == 'normal']

        emergency_data.sort(key=lambda x: x['node_count'])
        normal_data.sort(key=lambda x: x['node_count'])

        if emergency_data:
            nodes_e = [d['node_count'] * 5 for d in emergency_data]
            latency_e = [d['avg_confirm_latency_ms'] for d in emergency_data]
            ax.plot(nodes_e, latency_e, color=colors[interval],
                    linestyle='-', marker='o', linewidth=2.5,
                    markersize=8, label=f'紧急链 - {interval}ms')

        if normal_data:
            nodes_n = [d['node_count'] * 5 for d in normal_data]
            latency_n = [d['avg_confirm_latency_ms'] for d in normal_data]
            ax.plot(nodes_n, latency_n, color=colors[interval],
                    linestyle='--', marker='s', linewidth=2.5,
                    markersize=8, label=f'普通链 - {interval}ms')

    ax.set_xlabel('节点数量', fontsize=14, fontweight='bold')
    ax.set_ylabel('平均确认延迟 (ms)', fontsize=14, fontweight='bold')
    ax.set_title(f'延迟对比 (交易数量: {tx_count})', fontsize=16, fontweight='bold')
    ax.legend(loc='best', fontsize=10, ncol=2)
    ax.grid(True, alpha=0.3, linestyle='--')
    ax.tick_params(labelsize=12)

    ax.set_xticks([n * 5 for n in node_counts])
    plt.tight_layout()
    plt.savefig(f'plots/latency_comparison_tx{tx_count}.png', dpi=300, bbox_inches='tight')
    print(f'已保存: plots/latency_comparison_tx{tx_count}.png')
    plt.close()

print(f'\n总共生成了 {len(tx_per_block_values) * 2} 张图表')
print(f'交易数量值: {tx_per_block_values}')
