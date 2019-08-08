// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package handler

import (
    "encoding/json"
    "hash/crc32"
    "log"
    "sort"
    "sync"
    "sync/atomic"
)

const (
    OK = iota
    ERROR
    NOT_READY
    MOVE
)

type NodeInfo struct {
    ApiAddr   string `json:"apiAddr,omitempty"`
    Addr      string `json:"addr,omitempty"`
    SlotBegin uint32 `json:"slotBegin,omitempty"`
    SlotEnd   uint32 `json:"slotEnd,omitempty"`
    Master    bool   `json:"leader,omitempty"`
}

type NodeList []NodeInfo

type ClusterManager struct {
    mu          sync.Mutex
    Nodes       NodeList
    LeaderNodes NodeList

    state int32
}

var CRC32Q *crc32.Table

func init() {
    CRC32Q = crc32.MakeTable(0xD5828281)
}

func (n *NodeList) Len() int {
    return len(*n)
}

func (n *NodeList) Swap(i, j int) {
    (*n)[i], (*n)[j] = (*n)[j], (*n)[i]
}

func (n *NodeList) Less(i, j int) bool {
    return (*n)[i].SlotBegin < (*n)[j].SlotBegin
}

func (cm *ClusterManager) Update(node NodeInfo) {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    found := false
    for i := range cm.Nodes {
        if cm.Nodes[i].Addr == node.Addr {
            cm.Nodes[i].Master = node.Master
            cm.Nodes[i].SlotBegin = node.SlotBegin
            cm.Nodes[i].SlotEnd = node.SlotEnd
            found = true
            break
        }
    }
    if !found {
        cm.Nodes = append(cm.Nodes, node)
    }
    cm.checkNode()
}

func (cm *ClusterManager) Leave(node NodeInfo) {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    remove := -1
    for i := range cm.Nodes {
        if cm.Nodes[i].Addr == node.Addr {
            remove = i
            break
        }
    }

    if remove >= 0 {
        cm.Nodes = append(cm.Nodes[:remove], cm.Nodes[remove+1:]...)
    }
    cm.checkNode()
}

func (cm *ClusterManager) Join(node NodeInfo) {
    if node.Addr == "" {
        return
    }
    cm.mu.Lock()
    defer cm.mu.Unlock()

    for i := range cm.Nodes {
        if cm.Nodes[i].Addr == node.Addr {
            log.Printf("Join same node: %s\n", node.Addr)
            return
        }
    }

    cm.Nodes = append(cm.Nodes, node)

    cm.checkNode()
}

func (cm *ClusterManager) Enable() bool {
    return atomic.LoadInt32(&cm.state) == OK
}

func (cm *ClusterManager) Leaders() ([]NodeInfo, error) {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    ret := make([]NodeInfo, len(cm.LeaderNodes))
    for i := range cm.LeaderNodes {
        ret[i] = cm.LeaderNodes[i]
    }
    return ret, nil
}

func (cm *ClusterManager) refreshLeaders() {
    cm.LeaderNodes = NodeList{}
    for i := range cm.Nodes {
        if cm.Nodes[i].Master {
            cm.LeaderNodes = append(cm.LeaderNodes, cm.Nodes[i])
        }
    }
}

func (cm *ClusterManager) checkNode() bool {
    cm.refreshLeaders()
    sort.Sort(&cm.LeaderNodes)

    log.Printf("leader node %v\n", cm.LeaderNodes)
    log.Printf("all node %v\n", cm.Nodes)
    length := len(cm.LeaderNodes)
    if length == 0 {
        atomic.StoreInt32(&cm.state, ERROR)
        log.Printf("checkNode status: ERROR\n")
        return false
    }
    if cm.LeaderNodes[0].SlotBegin != 0 {
        atomic.StoreInt32(&cm.state, NOT_READY)
        log.Printf("checkNode status: NOT_READY, reason: SlotBegin is not 0\n")
        return false
    }
    if cm.LeaderNodes[length-1].SlotEnd != 16383 {
        atomic.StoreInt32(&cm.state, NOT_READY)
        log.Printf("checkNode status: NOT_READY, reason: SlotEnd is not 16383\n")
        return false
    }

    for i := 0; i < length-1; i++ {
        if cm.LeaderNodes[i].SlotEnd+1 != cm.LeaderNodes[i+1].SlotBegin {
            atomic.StoreInt32(&cm.state, NOT_READY)
            log.Printf("checkNode status: NOT_READY, reason: cm.Nodes[i].SlotEnd is %d cm.Nodes[i+1].SlotBegin is %d\n",
                cm.LeaderNodes[i].SlotEnd, cm.LeaderNodes[i+1].SlotBegin)
            return false
        }
    }
    atomic.StoreInt32(&cm.state, OK)
    log.Printf("checkNode status: OK\n")
    return true
}

func (cm *ClusterManager) FindNode(key string, master bool) (string, int32) {
    if !cm.Enable() {
        return "", cm.state
    }

    slot := CalcSlot(key)

    list := cm.LeaderNodes
    if !master {
        list = cm.Nodes
    }
    for _, v := range list {
        if v.CheckSlot(slot) {
            return v.ApiAddr, OK
        }
    }
    return "", ERROR
}

func CalcSlot(key string) uint32 {
    sum := crc32.Checksum([]byte(key), CRC32Q)
    return sum % 16384
}

func (n *NodeInfo)CheckSlot(slot uint32) bool {
    if n.SlotBegin > slot || n.SlotEnd < slot {
        return false
    } else {
        return true
    }
}

func marshalMeta(node NodeInfo) []byte {
    b, err := json.Marshal(node)
    log.Printf("%v\n", err)
    return b
}

func unmarshalMeta(meta []byte) NodeInfo {
    ret := NodeInfo{}
    err := json.Unmarshal(meta, &ret)
    log.Printf("%v\n", err)
    return ret
}
