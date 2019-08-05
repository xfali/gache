// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package handler

import (
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
    ApiAddr   string
    Addr      string
    SlotBegin uint32
    SlotEnd   uint32
    Master    bool
}

type NodeList []NodeInfo

type ClusterManager struct {
    mu    sync.Mutex
    Nodes NodeList

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

func (cm *ClusterManager) checkNode() bool {
    sort.Sort(&cm.Nodes)
    log.Printf("checkNode %v\n", cm.Nodes)
    length := len(cm.Nodes)
    if length == 0 {
        atomic.StoreInt32(&cm.state, ERROR)
        log.Printf("checkNode status: ERROR\n")
        return false
    }
    if cm.Nodes[0].SlotBegin != 0 {
        atomic.StoreInt32(&cm.state, NOT_READY)
        log.Printf("checkNode status: NOT_READY, reason: SlotBegin is not 0\n")
        return false
    }
    if cm.Nodes[length-1].SlotEnd != 16383 {
        atomic.StoreInt32(&cm.state, NOT_READY)
        log.Printf("checkNode status: NOT_READY, reason: SlotEnd is not 16383\n")
        return false
    }

    for i := 0; i < length-1; i++ {
        if cm.Nodes[i].SlotEnd + 1 != cm.Nodes[i+1].SlotBegin {
            atomic.StoreInt32(&cm.state, NOT_READY)
            log.Printf("checkNode status: NOT_READY, reason: cm.Nodes[i].SlotEnd is %d cm.Nodes[i+1].SlotBegin is %d\n",
                cm.Nodes[i].SlotEnd, cm.Nodes[i+1].SlotBegin)
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
    v := crc32.Checksum([]byte(key), CRC32Q)
    slot := v % 16384
    for _, v := range cm.Nodes {
        if v.SlotBegin > slot || v.SlotEnd < slot {
            continue
        } else {
            if master {
                if !v.Master {
                    continue
                }
            }
            return v.ApiAddr, OK
        }
    }
    return "", ERROR
}
