package shardctrler

import (
	"../raftServer/raft"
	"container/heap"
)
import "../raftServer/labrpc"
import "sync"
import "../raftServer/labgob"

type ShardCtrler struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	// Your data here.
	configs []Config // indexed by config num
}

// Less 这个是建堆的依据，建立最小堆，每次获取的元素都是shard数量最少的组
func (sc ShardCtrler) Less(i, j int) bool {
	return len(sc.configs[i].Shards) < len(sc.configs[j].Shards)
}

func (sc ShardCtrler) Len() int {
	return len(sc.configs)
}

func (sc ShardCtrler) Swap(i, j int) {
	sc.configs[i], sc.configs[j] = sc.configs[j], sc.configs[i]
}

func (sc *ShardCtrler) Push(x interface{}) {
	item := x.(Config)
	sc.configs = append(sc.configs, item)
}

func (sc *ShardCtrler) Pop() interface{} {
	l := len(sc.configs) - 1
	item := (sc.configs)[l]
	sc.configs = (sc.configs)[:l]
	return item
}

type Op struct {
	// Your data here.
	OpType      string
	SeqId       int
	ClientId    int64
	QueryNum    int
	JoinServers map[int][]string
	LeaveGids   []int
	MoveShard   int
	MoveGid     int
}

func (sc *ShardCtrler) Join(args *JoinArgs, reply *JoinReply) {
	_, ifLeader := sc.rf.GetState()
	if !ifLeader {
		reply.WrongLeader = true
		reply.Err = ErrWrongLeader
		return
	}
	// new GID -> servers mappings
	// 这里的GID是新组的ID
	// 将空的GID添加到config中，然后依次遍历args.Servers中的数据添加到config中
	// 添加的策略是每次取最少的
	heap.Push(sc, Config{Num: len(sc.configs),Groups: make(map[int][]string)})
	//for i,v := range args.Servers {
	//	for
	//}
}

func (sc *ShardCtrler) Leave(args *LeaveArgs, reply *LeaveReply) {

}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {

}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	_, ifLeader := sc.rf.GetState()
	if !ifLeader {
		reply.WrongLeader = true
		reply.Err = ErrWrongLeader
		return
	}
	lastIndex := len(sc.configs)-1
	if args.Num == -1 || args.Num >= len(sc.configs) {
		reply.Config = sc.configs[lastIndex]
		return
	}
	reply.Config = sc.configs[args.Num]
}

//
// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (sc *ShardCtrler) Kill() {
	sc.rf.Kill()
	// Your code here, if desired.
}

// needed by shardkv tester
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant shardctrler service.
// me is the index of the current server in servers[].
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardCtrler {
	sc := new(ShardCtrler)
	sc.me = me

	sc.configs = make([]Config, 1)
	sc.configs[0].Groups = map[int][]string{}

	labgob.Register(Op{})
	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)

	// Your code here.

	return sc
}


