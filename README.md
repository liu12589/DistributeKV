# DistributeKV
#### 【项目介绍】

本项目是 MIT 6.824 分布式课程项目，该系统包括领导选举,日志共识,日志快照,数据持久化,shard分片,负载均衡功能。具备高容错,故障、恢复,高吞吐,持久化,负载均衡的能力。

#### 【课程规定】

尽量自己设计，可以参考本人设计思路，课程规定不要参考他人的源代码，这样你的收获才是最大的。

#### 【项目核心目录介绍】

```
raftServer                 // raft 分布式共识服务模块
├── raft
    ├── raft.go            // raft 服务调用接口Start，初始化
    ├── election           // 领导选举核心模块
    └── persistence.go     // 日志持久化核心模块
    └── snapshoot.go       // 日志快照
    └── consensus.go       // 日志共识模块
    └── ticker.go          // 检测器
shardctrler                // shard 分片、负载均衡服务模块
├── client                 // 对外提供分布式系统分片配置服务
├── server                 // 实现负载均衡功能模块
shardkv。                  // shard 分片、负载均衡服务模块
├── client                 // 对外提供分布式 KV 系统查询、插入、删除、修改功能
├── server                 // 实现分布式 KV 系统查询、插入、删除、修改功能
```

#### 【核心代码逻辑梳理】

##### 检测器 ticker

###### electionTicker

​       raft 能够正常运转的核心逻辑就是检测器部分。这逻辑正确后各功能模块都是在此基础上添加的。这一部分理想的设计模式是状态机，也就是当该节点的服务状态改变时立即触发该状态对应的事件。这样就不需要检测器来实时检测当前节点应该执行哪个事件。

```go
func (rf *Raft) electionTicker(){ }
```

​       选举检测器，follower ——> Candidate 这一部分有个重要逻辑是何时进行选举。论文中详细描述了该部分逻辑，可概括为设置两个选举时延。第一个时延是选举间隔，每个服务的间隔都是随机的，防止选举割裂。第二时延是选举信号，可以理解为在这个时延内接收到 leader 消息后会被重置时间，则被抑制了本次选举。核心逻辑如下：

```go
time.Sleep(time.Duration(generateOverTime(int64(rf.me))) * time.Millisecond)
if rf.votedTimer.Before(nowTime) && rf.status != Leader {
  //进行选举逻辑代码
}
```

###### appendTicker

​        这个逻辑非常简单，如果该节点是 leader，定时向从节点发送心跳。

###### committedTicker

​        定时监测，哪些命令已经由分布式服务达成共识，达成共识的部分发送给调用者。这部分实现使用了 Chan 管道进行通信。

##### 领导选举核心模块 election  

该部分有两大部分：发送选票、进行投票。这部分论文中描述的非常清楚。由于使用 rpc 的方式进行通信，因此发送选票的逻辑和判断自己是否能够当选 leader 的逻辑封装在统一模块中。

- 发送选票

  1. 发送本节点当前信息，发送信息如下，在论文中规定的比较详细

  ```go
  args := RequestVoteArgs{
  				rf.currentTerm,
  				rf.me,
  				rf.getLastIndex(),
  				rf.getLastTerm(),
  }
  ```

  2. 判断当前能否当选 leader，当前选票多于半数以上当选。

  ```go
  if reply.VoteGranted == true && rf.currentTerm == args.Term {
    rf.voteNum += 1
    if rf.voteNum >= len(rf.peers)/2+1 {
      rf.status = Leader
      rf.votedFor = -1
      rf.voteNum = 0
      rf.persist()
      rf.votedTimer = time.Now()
      rf.mu.Unlock()
      return
    }
    rf.mu.Unlock()
    return
    }
  }
  ```

- 进行投票

  这一部分论文中详细解释了，主要逻辑如下

  1. 投票对象任期，比本节点任期小，直接返回。可以修复网络分区。
  2. 任期大于本节点，本节点的信息进行同步。
  3. 判断投票对象的日志与本节点已经存储的日志是否冲突。
  4. 判断本节点是否已经投票。

##### 日志共识 consensus

日志共识部分主要是两部分内容：主节点向从节点发送要共识的内容，从节点接收内容并更新本节点的存储内容。这一模块主要逻辑如下：

- start
  1. 客户端将信息发送给leader。leader首先将该信息作为日志保存。

- Leader send
  1. 如果 last log index >= 从节点的 nextIndex。那么发送的日志从 nextIndex 开始
  2. 如果一个leader看到更高的选举term，将会下台。
  3. 如果成功，更新 follower 的 nextIndex 和 matchIndex。
  4. 如果存在一个N，使得 N＞commitIndex，如果matchIndex[i] 中大于 N 的个数多余一般，并且 log[N].term == currentTerm。设置commitIndex = N。
  5. 如果AppendEntries由于日志不一致而失败：减少nextIndex并重试。这里想法是，寻找正确的位置进行更新。或者检测到不一致直接nextIndex--

- Follower receive
  1. 如果 term < currentTerm。return false。并且返回当前term
  2. 更新该节点的选举时间。这还有个很重要的点。就是收到 leader 的心跳信息后不仅要更新选举时间。还要重置选举信息。
  3. 如果当前节点的日志与发送过来的 preLogIndex 和 prevLogTerm 不匹配。返回false
  4. 如果现有条目与新条目冲突（相同索引，但术语不同），请删除现有条目及其后面的所有条目
  5. 附加日志中尚未添加的任何新条目
  6. 如果leaderCommit>commitIndex，则设置CommitIndx=min（leaderCommit，最后一个新条目的索引）

##### 日志持久化

​      该部分内容就是为了能在机器crash的时候能够restore原来的状态。这部分内容在论文中有详细描述。当currentTerm、voteFor、log[]三个state改变时直接持久化。对于此次部分来说首先应该实现：persist、与readPersist两个函数。分别是编码（encode）、解码（[decode](https://so.csdn.net/so/search?q=decode&spm=1001.2101.3001.7020)）。总的来说就是状态变化时持久化（这里使用的是缓存，而不是数据库），当某节点宕机恢复时读取持久化的内容。

编码缓存

```go
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.logs)
	e.Encode(rf.lastIncludeIndex)
	e.Encode(rf.lastIncludeTerm)
	data := w.Bytes()
```

解码恢复现场

```go
if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&logs) != nil ||
		d.Decode(&lastIncludeIndex) != nil ||
		d.Decode(&lastIncludeTerm) != nil {
		fmt.Println("decode error")
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.logs = logs
		rf.lastIncludeIndex = lastIncludeIndex
		rf.lastIncludeTerm = lastIncludeTerm
	}
```

##### 日志快照

该部分的作用是：Raft 的日志在正常运行时会增长，以容纳更多的客户端请求，但在实际系统中，它不能无限制地增长。 随着日志变长，它会占用更多空间并需要更多时间来重播。 如果没有某种机制来丢弃日志中积累的过时信息，这最终会导致可用性问题。
也因此我们使用Snapshot（快照）来简单的实现日志压缩。这里跟持久化有个区别。注意持久化是各个节点状态变化时的状态数据，不是存储的日志。而日志快照只是为了持久化已经共识的日志部分。也可以是节点宕机恢复后调用日志快照。

- 什么时候需要快照

​       就是leader发送给follower时的日志已经被抛弃了。那么此时需要发送快照。那么因此发送快照的调用入口应该在进行日志增量时的日志检查。而查询的条件就是rf.nextIndex[server]-1 < rf.lastIncludeIndex比自身的快照还低。

```go
if rf.nextIndex[server]-1 < rf.lastIncludeIndex {
    go rf.leaderSendSnapShot(server)
    rf.mu.Unlock()
    return
}
```

- 安装快照部分逻辑，按照论文中的逻辑没问题。

![f3fab4f99fa44c16a7936fcabbbda5f1](https://user-images.githubusercontent.com/46433529/189481144-19f7dc29-f83f-458f-89a7-ab7fdd587848.png)

##### shard 分片、负载均衡

##### shardkv

这两部分内容可以参考 https://github.com/OneSizeFitsQuorum/MIT6.824-2021/blob/master/docs/lab4.md#%E6%97%A5%E5%BF%97%E7%B1%BB%E5%9E%8B
