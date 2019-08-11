# gache

## 介绍 

  gache是一个高可用的Key-Value型nosql数据库最小实现，使用raft实现高可用。

## 安装

```
  go get github.com/xfali/gache
```

## 运行（主从复制）

测试示例如下：

leader:
```
./gache --raft-addr 127.0.0.1:7001 --raft-dir ./tmp/node1 -p 8001
```

follower1:
```
./gache --raft-addr 127.0.0.1:7002 --raft-dir ./tmp/node2 -p 8002 --raft-join 127.0.0.1:8001
```

follower2:
```
./gache --raft-addr 127.0.0.1:7003 --raft-dir ./tmp/node3 -p 8003 --raft-join 127.0.0.1:8001
```

## 运行（分片）

### 命令
node1:
```
./gache -p 8001 --cluster-port 9001 --cluster-slot 0-5000
```

node2:
```
./gache -p 8002 --cluster-port 9002 --cluster-slot 5001-10000 --cluster-members 127.0.0.1:9001
```

node3:
```
./gache -p 8003 --cluster-port 9003 --cluster-slot 10001-16383 --cluster-members 127.0.0.1:9001
```

### 测试

集群状态OK之后执行设置值：
```
curl localhost:8001/key/2 -X POST -L -d "key2-value" -v
```
查询：
```
curl localhost:8001/key/2  -L -v
```

## 高可用分片集群

### 命令
raft cluster 1:
```
./gache -p 8001 --raft-addr 127.0.0.1:7001 --raft-dir ./tmp/node1 --cluster-port 9001 --cluster-slot 0-5000
```
```
./gache -p 8002 --raft-addr 127.0.0.1:7002 --raft-dir ./tmp/node2 --raft-join 127.0.0.1:8001 --cluster-port 9002 --cluster-slot 0-5000 --cluster-members 127.0.0.1:9001
```
```
./gache -p 8003 --raft-addr 127.0.0.1:7003 --raft-dir ./tmp/node3 --raft-join 127.0.0.1:8001 --cluster-port 9003 --cluster-slot 0-5000 --cluster-members 127.0.0.1:9001
```

raft cluster 2:
```
./gache -p 8004 --raft-addr 127.0.0.1:7004 --raft-dir ./tmp/node4 --cluster-port 9004 --cluster-slot 5001-10000 --cluster-members 127.0.0.1:9001
```
```
./gache -p 8005 --raft-addr 127.0.0.1:7005 --raft-dir ./tmp/node5 --raft-join 127.0.0.1:8004 --cluster-port 9005 --cluster-slot 5001-10000 --cluster-members 127.0.0.1:9004
```
```
./gache -p 8006 --raft-addr 127.0.0.1:7006 --raft-dir ./tmp/node6 --raft-join 127.0.0.1:8004 --cluster-port 9006 --cluster-slot 5001-10000 --cluster-members 127.0.0.1:9004
```
raft cluster 3:
```
./gache -p 8007 --raft-addr 127.0.0.1:7007 --raft-dir ./tmp/node7 --cluster-port 9007 --cluster-slot 10001-16383 --cluster-members 127.0.0.1:9001
```
```
./gache -p 8008 --raft-addr 127.0.0.1:7008 --raft-dir ./tmp/node8 --raft-join 127.0.0.1:8007 --cluster-port 9008 --cluster-slot 10001-16383 --cluster-members 127.0.0.1:9007
```
```
./gache -p 8009 --raft-addr 127.0.0.1:7009 --raft-dir ./tmp/node9 --raft-join 127.0.0.1:8007 --cluster-port 9009 --cluster-slot 10001-16383 --cluster-members 127.0.0.1:9007
```

### 测试

集群状态OK之后执行设置值：
```
curl localhost:8001/key/2 -X POST -L -d "key2-value" -v
```
查询：
```
curl localhost:8001/key/2  -L -v
```

## 访问

地址：
http://127.0.0.1:8001/key/${KEY}

### 操作数据库

* POST：

   保存${KEY} 和 body的值
   
* DELETE：

   删除${KEY}
   
* GET:
  
   获得${KEY}对应的值

### Benchmark
```
go test -v -cpu=8 -run=^$ -bench=. ./test -args ${HOST}:${PORT}
```