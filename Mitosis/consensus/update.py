# -*- coding: utf-8 -*-

def generate_topology(pshard_num, nodes, r_pershard):
    pshard_ids = list(range(1001, pshard_num + 1001))

    pshard_count = pshard_num * nodes
    rshard_count = pshard_num // r_pershard

    pshard_nodes = []
    for i in range(1001, 1001 + pshard_count):
        pshard_nodes.append(i)

    pshard_ids_map = {}
    for i in range(1, rshard_count + 1):
        start = (i - 1) * r_pershard
        end = i * r_pershard
        pshard_ids_map[i] = pshard_ids[start:end]


    extra_pshard_ids = pshard_ids[rshard_count * r_pershard:]
    if extra_pshard_ids:
        pshard_ids_map[rshard_count + 1] = extra_pshard_ids
        rshard_count = rshard_count + 1

    return pshard_ids_map, rshard_count, pshard_count + rshard_count * nodes

# 输入参数
pshard_num = 4
nodes = 4
r_pershard = 2

# 生成拓扑
pshard_ids_map, rshard_count, total_nodes = generate_topology(pshard_num, nodes, r_pershard)

# 输出结果
print("PshardIds:")
print("{")
for rshard_id, pshard_ids in pshard_ids_map.items():
    if pshard_ids:
        print(f"\t{rshard_id}: {{{', '.join(str(pshard_id) for pshard_id in pshard_ids)}}},")
print("}")

print("RshardIds:")
print("{")
for rshard_id in range(1, rshard_count + 1):
    print(f"{rshard_id},",end="")
print("}")

print("Rshard节点数量:", rshard_count * nodes)
print("总节点数量:", total_nodes)