#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
信誉值变化曲线绘制脚本
"""

import matplotlib.pyplot as plt
import matplotlib
import numpy as np

# 设置中文字体支持
matplotlib.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'Arial Unicode MS']
matplotlib.rcParams['axes.unicode_minus'] = False

# 数据：从reputation_log.txt中提取
rounds = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

# 诚实节点平均信誉值
honest_reputation = [
    0.500,      # 第0轮（初始值）
    0.569150,   # 第1轮
    0.610065,   # 第2轮
    0.612010,   # 第3轮
    0.653257,   # 第4轮
    0.657985,   # 第5轮
    0.682819,   # 第6轮
    0.689881,   # 第7轮
    0.710751,   # 第8轮
    0.711235,   # 第9轮
    0.733346,   # 第10轮
]

# 恶意节点平均信誉值
malicious_reputation = [
    0.500,      # 第0轮（初始值）
    0.133333,   # 第1轮
    0.100000,   # 第2轮
    0.127174,   # 第3轮
    0.124735,   # 第4轮
    0.115910,   # 第5轮
    0.132481,   # 第6轮
    0.122351,   # 第7轮
    0.119674,   # 第8轮
    0.118675,   # 第9轮
    0.113548,   # 第10轮
]

# 创建图表
fig, ax = plt.subplots(figsize=(12, 7))

# 绘制诚实节点平均信誉曲线（绿色，圆点标记）
line1 = ax.plot(rounds, honest_reputation, 
                color='#2ecc71', 
                linewidth=2.5, 
                marker='o', 
                markersize=8,
                label='诚实节点平均信誉',
                zorder=3)

# 绘制恶意节点平均信誉曲线（红色，方块标记）
line2 = ax.plot(rounds, malicious_reputation, 
                color='#e74c3c', 
                linewidth=2.5, 
                marker='s', 
                markersize=8,
                label='恶意节点平均信誉',
                zorder=3)

# 绘制0.5信誉阈值虚线（灰色）
ax.axhline(y=0.5, color='gray', linestyle='--', linewidth=1.5, 
           label='信誉阈值（0.5）', alpha=0.7, zorder=1)

# 标注初始值（第0轮）
ax.annotate(f'{honest_reputation[0]:.3f}', 
            xy=(0, honest_reputation[0]), 
            xytext=(0.3, honest_reputation[0] + 0.02),
            fontsize=10,
            color='#2ecc71',
            bbox=dict(boxstyle='round,pad=0.3', facecolor='white', edgecolor='#2ecc71', alpha=0.8))

ax.annotate(f'{malicious_reputation[0]:.3f}', 
            xy=(0, malicious_reputation[0]), 
            xytext=(0.3, malicious_reputation[0] - 0.02),
            fontsize=10,
            color='#e74c3c',
            bbox=dict(boxstyle='round,pad=0.3', facecolor='white', edgecolor='#e74c3c', alpha=0.8))

# 标注第5轮（中间点）
mid_round = 5
ax.annotate(f'{honest_reputation[mid_round]:.3f}', 
            xy=(mid_round, honest_reputation[mid_round]), 
            xytext=(mid_round - 0.5, honest_reputation[mid_round] + 0.03),
            fontsize=10,
            color='#2ecc71',
            bbox=dict(boxstyle='round,pad=0.3', facecolor='white', edgecolor='#2ecc71', alpha=0.8))

ax.annotate(f'{malicious_reputation[mid_round]:.3f}', 
            xy=(mid_round, malicious_reputation[mid_round]), 
            xytext=(mid_round - 0.5, malicious_reputation[mid_round] + 0.03),
            fontsize=10,
            color='#e74c3c',
            bbox=dict(boxstyle='round,pad=0.3', facecolor='white', edgecolor='#e74c3c', alpha=0.8))

# 标注最终值（第10轮）
final_round = 10
ax.annotate(f'{honest_reputation[final_round]:.3f}', 
            xy=(final_round, honest_reputation[final_round]), 
            xytext=(final_round - 0.3, honest_reputation[final_round] + 0.02),
            fontsize=10,
            color='#2ecc71',
            bbox=dict(boxstyle='round,pad=0.3', facecolor='white', edgecolor='#2ecc71', alpha=0.8))

# 添加最终统计框
stats_text = (
    f'最终结果统计：\n'
    f'诚实节点：{honest_reputation[final_round]:.4f}（↑{(honest_reputation[final_round]-0.5)*100:.1f}%）\n'
    f'恶意节点：{malicious_reputation[final_round]:.4f}（↓{(0.5-malicious_reputation[final_round])*100:.1f}%）\n'
    f'信誉差距：{(honest_reputation[final_round]-malicious_reputation[final_round]):.4f}（{((honest_reputation[final_round]/malicious_reputation[final_round]-1)*100):.1f}%）'
)
ax.text(0.98, 0.02, stats_text,
        transform=ax.transAxes,
        fontsize=10,
        verticalalignment='bottom',
        horizontalalignment='right',
        bbox=dict(boxstyle='round,pad=0.8', facecolor='wheat', edgecolor='black', alpha=0.9))

# 设置图表标题和标签
ax.set_title('基于主观逻辑的信誉值变化曲线', fontsize=16, fontweight='bold', pad=20)
ax.set_xlabel('共识轮次', fontsize=14, fontweight='bold')
ax.set_ylabel('信誉值', fontsize=14, fontweight='bold')

# 设置坐标轴范围
ax.set_xlim(-0.5, 10.5)
ax.set_ylim(0.0, 0.85)

# 设置x轴刻度
ax.set_xticks(rounds)
ax.set_xticklabels(rounds)

# 添加网格
ax.grid(True, linestyle=':', alpha=0.3, zorder=0)

# 设置图例
ax.legend(loc='upper left', fontsize=12, framealpha=0.9, 
          edgecolor='black', fancybox=True, shadow=True)

# 调整布局
plt.tight_layout()

# 保存图表
output_file = 'reputation_chart.png'
plt.savefig(output_file, dpi=300, bbox_inches='tight')
print(f'图表已保存到: {output_file}')

# 显示图表
plt.show()

