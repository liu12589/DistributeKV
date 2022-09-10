package shardctrler

// The number of shards.
const NShards = 10

// Config A configuration -- an assignment of shards to groups.
// Please don't change this.
// 最多有10个分片
type Config struct {
	Num    int              // config number
	Shards [NShards]int     // shard -> gid
	Groups map[int][]string // gid -> servers[]
}

const (
	OK             = "OK"
	ErrWrongLeader = "ErrWrongLeader"
)

type Err string

type JoinArgs struct {
	Servers  map[int][]string // new GID -> servers mappings
	SeqId    int
	ClientId int64
}

type JoinReply struct {
	WrongLeader bool
	Err         Err
}

type LeaveArgs struct {
	GIDs     []int
	SeqId    int
	ClientId int64
}

type LeaveReply struct {
	WrongLeader bool
	Err         Err
}

type MoveArgs struct {
	Shard    int
	GID      int
	SeqId    int
	ClientId int64
}

type MoveReply struct {
	WrongLeader bool
	Err         Err
}

type QueryArgs struct {
	Num      int // desired config number
	SeqId    int
	ClientId int64
}

type QueryReply struct {
	WrongLeader bool
	Err         Err
	Config      Config
}
