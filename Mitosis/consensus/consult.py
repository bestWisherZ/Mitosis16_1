import re

# 日志文件路径
log_file_path = './log3.txt'

def extract_block_entries(log_content, shard_id1, shard_id2):
    """
    提取 receive/gossip block 信息
    """
    pattern = re.compile(rf'time="[^"]*" level=info msg="\[Node-{shard_id1}-[^]]*\] (receive|gossip) [^"]* block-{shard_id2}-[^"]*" process=blockchain')
    matches = pattern.findall(log_content)
    return matches

# 查找local_peer
def extract_local_peer(log_content, shard_id, node_id):
    """
    提取 localPeer 的值，返回找到的第一个值
    """
    # 定义正则表达式模式
    pattern = re.compile(rf'time="[^"]*" level=info msg="\[Node-{shard_id}-{node_id}\] starting gossip" localPeer=([^ ]*) process=p2p')
    
    # 查找所有符合模式的行
    match = pattern.search(log_content)
    
    if match:
        return match.group(1)  # 返回第一个匹配的 localPeer 值
    else:
        return None

def extract_connections_entries(log_content, shard_id, peer_id):
    """
    提取连接到对等节点的信息
    """
    pattern = re.compile(rf'time="[^"]*" level=info msg="\[Node-{shard_id}-\d+\] connected to peer: {peer_id} with dht" process=p2p')
    matches = pattern.findall(log_content)
    return matches

def read_log_file(file_path):
    """
    从文件中读取日志内容
    """
    try:
        with open(file_path, 'r') as file:
            return file.read()
    except FileNotFoundError:
        print(f"Error: The file {file_path} does not exist.")
        return None
    except IOError as e:
        print(f"Error reading file {file_path}: {e}")
        return None

def main():
    # 设置shard_id
    shard_id1 = "1003"
    shard_id2 = "1019"

    shard_id = "1039"
    
    # 从文件中读取日志内容
    log_content = read_log_file(log_file_path)
    if log_content is None:
        return

    # 提取日志条目
    # block_entries = extract_block_entries(log_content, shard_id1, shard_id2)

    # 查找[3-12]的local_peer
    local_peer = extract_local_peer(log_content, 3, 12)
    # local_peer = "12D3KooWDDPSWyg2CUwVmPQvcf8mhTX6129PHzFJHyBqTDyM1QkY"
    peer_connections = extract_connections_entries(log_content, shard_id, local_peer)

    # 打印结果
    # print("Block Entries:")
    # for entry in block_entries:
    #     print(entry)
    
    print("\nPeer Connections:")
    for connection in peer_connections:
        print(connection)

if __name__ == "__main__":
    main()



# if are_shards_connected(shard1, shard2):
#     print(f"Shard {shard1} and Shard {shard2} are connected.")
# else:
#     print(f"Shard {shard1} and Shard {shard2} are not connected.")
