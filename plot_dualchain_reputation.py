#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
双链系统信誉值变化曲线绘制脚本
展示紧急交易权重改进的效果
"""

import matplotlib.pyplot as plt
import matplotlib
import numpy as np

# 设置中文字体支持
matplotlib.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'Arial Unicode MS']
matplotlib.rcParams['axes.unicode_minus'] = False

# 数据：模拟三类节点的信誉值变化
# 轮次
rounds = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 15, 18, 20]

# 节点A：只处理普通交易的诚实节点（权重1.0）
normal_only_reputation = [
    0.500,   # 初始值
    0.530,   # 第1轮
    0.555,   # 第2轮
    0.575,   # 第3轮
    0.595,   # 第4轮
    0.612,   # 第5轮
    0.628,   # 第6轮
    0.642,   # 第7轮
    0.655,   # 第8轮
    0.667,   # 第9轮
    0.678,   # 第10轮
    0.698,   # 第12轮
    0.725,   # 第15轮
    0.745,   # 第18轮
    0.758,   # 第20轮
]

# 节点B：积极处理紧急交易的诚实节点（权重2.0-5.0）
emergency_active_reputation = [
    0.500,   # 初始值
    0.565,   # 第1轮 - 紧急交易加速提升
    0.620,   # 第2轮
    0.665,   # 第3轮
    0.702,   # 第4轮
    0.735,   # 第5轮
    0.763,   # 第6轮
    0.787,   # 第7轮
    0.808,   # 第8轮
    0.826,   # 第9轮
    0.842,   # 第10轮
    0.868,   # 第12轮
    0.895,   # 第15轮
    0.912,   # 第18轮
    0.923,   # 第20轮
]

# 节点C：在紧急交易中作恶的恶意节点（权重5.0惩罚）
emergency_malicious_reputation = [
    0.500,   # 初始值
    0.385,   # 第1轮 - 紧急交易惩罚快速下降
    0.298,   # 第2轮
    0.235,   # 第3轮
    0.192,   # 第4轮
    0.162,   # 第5轮
    0.141,   # 第6轮
    0.126,   # 第7轮
    0.115,   # 第8轮
    0.107,   # 第9轮
    0.101,   # 第10轮
    0.092,   # 第12轮
    0.081,   # 第15轮
    0.074,   # 第18轮
    0.070,   # 第20轮
]

# 节点D：普通交易中的恶意节点（权重1.0惩罚）
normal_malicious_reputation = [
    0.500,   # 初始值
    0.450,   # 第1轮 - 普通惩罚较慢
    0.410,   # 第2轮
    0.378,   # 第3轮
    0.352,   # 第4轮
    0.330,   # 第5轮
    0.312,   # 第6轮
    0.297,   # 第7轮
    0.284,   # 第8轮
    0.273,   # 第9轮
    0.264,   # 第10轮
    0.248,   # 第12轮
    0.228,   # 第15轮
    0.213,   # 第18轮
    0.203,   # 第20轮
]

# 创建图表
fig, ax = plt.subplots(figsize=(14, 8))

# 绘制四条曲线
# 1. 积极处理紧急交易的节点（深绿色，圆点）
line1 = ax.plot(rounds, emergency_active_reputation, 
                color='#27ae60', 
                linewidth=3.0, 
                marker='o', 
                markersize=9,
                label='节点B：积极处理紧急交易（权重×2-5）',
                zorder=4)

# 2. 只处理普通交易的节点（浅绿色，圆点）
line2 = ax.plot(rounds, normal_only_reputation, 
                color='#2ecc71', 
                linewidth=2.5, 
                marker='o', 
                markersize=8,
                label='节点A：仅处理普通交易（权重×1）',
                zorder=3)

# 3. 紧急交易中作恶的节点（深红色，方块）
line3 = ax.plot(rounds, emergency_malicious_reputation, 
                color='#c0392b', 
                linewidth=3.0, 
                marker='s', 
                markersize=9,
                label='节点C：紧急交易作恶（惩罚×5）',
                zorder=4)

# 4. 普通交易中作恶的节点（浅红色，方块）
line4 = ax.plot(rounds, normal_malicious_reputation, 
                color='#e74c3c', 
                linewidth=2.5, 
                marker='s', 
                markersize=8,
                label='节点D：普通交易作恶（惩罚×1）',
                zorder=3)

# 绘制0.5信誉阈值虚线
ax.axhline(y=0.5, color='gray', linestyle='--', linewidth=2.0, 
           label='信誉阈值（0.5）', alpha=0.7, zorder=1)

# 绘制验证器选举阈值线（假设为0.7）
ax.axhline(y=0.7, color='orange', linestyle=':', linewidth=2.0, 
           label='验证器阈值（0.7）', alpha=0.6, zorder=1)

# 标注关键点
# 初始点
ax.annotate('初始值', 
            xy=(0, 0.5), 
            xytext=(0.5, 0.55),
            fontsize=11,
            color='black',
            arrowprops=dict(arrowstyle='->', color='gray', lw=1.5),
            bbox=dict(boxstyle='round,pad=0.4', facecolor='white', edgecolor='gray', alpha=0.9))

# 节点B突破验证器阈值
breakthrough_idx = 5
ax.annotate(f'突破验证器阈值\n{emergency_active_reputation[breakthrough_idx]:.3f}', 
            xy=(rounds[breakthrough_idx], emergency_active_reputation[breakthrough_idx]), 
            xytext=(rounds[breakthrough_idx] + 1.5, emergency_active_reputation[breakthrough_idx] + 0.08),
            fontsize=10,
            color='#27ae60',
            arrowprops=dict(arrowstyle='->', color='#27ae60', lw=2),
            bbox=dict(boxstyle='round,pad=0.5', facecolor='lightgreen', edgecolor='#27ae60', alpha=0.9))

# 最终值标注
final_idx = -1
ax.annotate(f'{emergency_active_reputation[final_idx]:.3f}', 
            xy=(rounds[final_idx], emergency_active_reputation[final_idx]), 
            xytext=(rounds[final_idx] - 1.5, emergency_active_reputation[final_idx] + 0.03),
            fontsize=11,
            color='#27ae60',
            bbox=dict(boxstyle='round,pad=0.4', facecolor='white', edgecolor='#27ae60', alpha=0.9))

ax.annotate(f'{normal_only_reputation[final_idx]:.3f}', 
            xy=(rounds[final_idx], normal_only_reputation[final_idx]), 
            xytext=(rounds[final_idx] - 1.5, normal_only_reputation[final_idx] + 0.03),
            fontsize=11,
            color='#2ecc71',
            bbox=dict(boxstyle='round,pad=0.4', facecolor='white', edgecolor='#2ecc71', alpha=0.9))

ax.annotate(f'{emergency_malicious_reputation[final_idx]:.3f}', 
            xy=(rounds[final_idx], emergency_malicious_reputation[final_idx]), 
            xytext=(rounds[final_idx] - 1.5, emergency_malicious_reputation[final_idx] + 0.03),
            fontsize=11,
            color='#c0392b',
            bbox=dict(boxstyle='round,pad=0.4', facecolor='white', edgecolor='#c0392b', alpha=0.9))

# 添加信誉增速对比箭头（第5轮到第10轮）
start_idx = 5
end_idx = 10

# 节点B的增速
b_growth = emergency_active_reputation[end_idx] - emergency_active_reputation[start_idx]
ax.annotate('', 
            xy=(rounds[end_idx] - 0.3, emergency_active_reputation[end_idx]), 
            xytext=(rounds[start_idx] + 0.3, emergency_active_reputation[start_idx]),
            arrowprops=dict(arrowstyle='<->', color='#27ae60', lw=2.5, linestyle='-'))
ax.text(rounds[start_idx] + 2.5, (emergency_active_reputation[start_idx] + emergency_active_reputation[end_idx]) / 2,
        f'+{b_growth:.3f}',
        fontsize=10, color='#27ae60', fontweight='bold',
        bbox=dict(boxstyle='round,pad=0.3', facecolor='lightgreen', alpha=0.8))

# 节点A的增速
a_growth = normal_only_reputation[end_idx] - normal_only_reputation[start_idx]
ax.annotate('', 
            xy=(rounds[end_idx] - 0.3, normal_only_reputation[end_idx]), 
            xytext=(rounds[start_idx] + 0.3, normal_only_reputation[start_idx]),
            arrowprops=dict(arrowstyle='<->', color='#2ecc71', lw=2, linestyle='--'))
ax.text(rounds[start_idx] + 2.5, (normal_only_reputation[start_idx] + normal_only_reputation[end_idx]) / 2 - 0.03,
        f'+{a_growth:.3f}',
        fontsize=10, color='#2ecc71',
        bbox=dict(boxstyle='round,pad=0.3', facecolor='white', edgecolor='#2ecc71', alpha=0.8))

# 添加统计信息框
stats_text = (
    f'📊 权重改进效果统计（第{rounds[final_idx]}轮）：\n'
    f'━━━━━━━━━━━━━━━━━━━━━━━━━\n'
    f'✅ 节点B（紧急交易）：{emergency_active_reputation[final_idx]:.4f}\n'
    f'   信誉增长：+{(emergency_active_reputation[final_idx]-0.5):.4f} ({(emergency_active_reputation[final_idx]-0.5)/0.5*100:.1f}%)\n'
    f'   增速：{b_growth/5:.4f}/轮\n'
    f'\n'
    f'✅ 节点A（普通交易）：{normal_only_reputation[final_idx]:.4f}\n'
    f'   信誉增长：+{(normal_only_reputation[final_idx]-0.5):.4f} ({(normal_only_reputation[final_idx]-0.5)/0.5*100:.1f}%)\n'
    f'   增速：{a_growth/5:.4f}/轮\n'
    f'\n'
    f'💡 紧急交易加速效果：\n'
    f'   信誉差距：{(emergency_active_reputation[final_idx]-normal_only_reputation[final_idx]):.4f}\n'
    f'   增速倍数：{(b_growth/a_growth):.2f}x\n'
    f'\n'
    f'⚠️ 恶意节点惩罚对比：\n'
    f'   紧急作恶：{emergency_malicious_reputation[final_idx]:.4f} (↓{(0.5-emergency_malicious_reputation[final_idx])/0.5*100:.1f}%)\n'
    f'   普通作恶：{normal_malicious_reputation[final_idx]:.4f} (↓{(0.5-normal_malicious_reputation[final_idx])/0.5*100:.1f}%)\n'
    f'   惩罚倍数：{((0.5-emergency_malicious_reputation[final_idx])/(0.5-normal_malicious_reputation[final_idx])):.2f}x'
)

ax.text(0.02, 0.98, stats_text,
        transform=ax.transAxes,
        fontsize=9.5,
        verticalalignment='top',
        horizontalalignment='left',
        family='monospace',
        bbox=dict(boxstyle='round,pad=1.0', facecolor='lightyellow', edgecolor='orange', alpha=0.95, linewidth=2))

# 设置图表标题和标签
ax.set_title('双链系统：紧急交易权重改进的信誉值变化对比', 
             fontsize=17, fontweight='bold', pad=20, color='#2c3e50')
ax.set_xlabel('共识轮次', fontsize=14, fontweight='bold')
ax.set_ylabel('信誉值', fontsize=14, fontweight='bold')

# 设置坐标轴范围
ax.set_xlim(-0.5, rounds[-1] + 0.5)
ax.set_ylim(0.0, 1.0)

# 设置x轴刻度
ax.set_xticks(rounds)
ax.set_xticklabels(rounds)

# 添加网格
ax.grid(True, linestyle=':', alpha=0.4, zorder=0)

# 设置图例（分两列）
ax.legend(loc='upper left', fontsize=11, framealpha=0.95, 
          edgecolor='black', fancybox=True, shadow=True, ncol=2)

# 添加水印
ax.text(0.98, 0.02, '基于主观逻辑的信誉系统 + 紧急交易权重改进',
        transform=ax.transAxes,
        fontsize=9,
        verticalalignment='bottom',
        horizontalalignment='right',
        alpha=0.5,
        style='italic')

# 调整布局
plt.tight_layout()

# 保存图表
output_file = 'dualchain_reputation_chart.png'
plt.savefig(output_file, dpi=300, bbox_inches='tight', facecolor='white')
print(f'✅ 图表已保存到: {output_file}')
print(f'📊 图表尺寸: 14×8 英寸')
print(f'🎨 分辨率: 300 DPI')

# 显示图表
plt.show()

