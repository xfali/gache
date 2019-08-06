// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package gossip

import (
    "github.com/hashicorp/memberlist"
    "log"
    "sync"
)

type handle func(meta []byte)

type NodeDelegate struct {
    mu         sync.Mutex
    Enabled    bool
    JoinFunc   handle
    LeaveFunc  handle
    UpdateFunc handle
}

func defaultJoin(meta []byte) {
    log.Printf("defaultJoin meta: %s\n", string(meta))
}

func defaultLeave(meta []byte) {
    log.Printf("defaultLeave meta: %s\n", string(meta))
}

func defaultUpdate(meta []byte) {
    log.Printf("defaultUpdate meta: %s\n", string(meta))
}

func DefaultNodeDelegate() *NodeDelegate {
    return &NodeDelegate{
        Enabled:    true,
        JoinFunc:   defaultJoin,
        LeaveFunc:  defaultLeave,
        UpdateFunc: defaultUpdate,
    }
}

func (d *NodeDelegate) NotifyJoin(n *memberlist.Node) {
    d.mu.Lock()
    defer d.mu.Unlock()
    if d.Enabled {
        d.JoinFunc(n.Meta)
    }
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (d *NodeDelegate) NotifyLeave(n *memberlist.Node) {
    d.mu.Lock()
    defer d.mu.Unlock()
    if d.Enabled {
        d.LeaveFunc(n.Meta)
    }
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (d *NodeDelegate) NotifyUpdate(n *memberlist.Node) {
    d.mu.Lock()
    defer d.mu.Unlock()
    if d.Enabled {
        d.UpdateFunc(n.Meta)
    }
}

