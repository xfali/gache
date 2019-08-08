// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package gossip

import (
    "gache/cluster"
    "gache/config"
    "github.com/hashicorp/memberlist"
    "os"
    "strconv"
    "strings"
    "time"
)

type members memberlist.Memberlist

type Cluster interface {
    LocalAddr() string
    UpdateLocal(meta []byte) error
    UpdateAndWait(meta []byte, timeout time.Duration) error
    Close() error
    Enabled() bool
}

func Startup(conf *config.Config, delegate *NodeDelegate) (Cluster, error) {
    hostname, _ := os.Hostname()
    config := memberlist.DefaultLocalConfig()
    config.Name = hostname + "-" + strconv.Itoa(conf.ClusterPort)
    // config := memberlist.DefaultLocalConfig()
    config.BindPort = conf.ClusterPort
    config.AdvertisePort = conf.ClusterPort
    config.Events = delegate

    list, err := memberlist.Create(config)
    if err != nil {
        return nil, err
    }

    _, _, sloterr := cluster.GetSlots(conf.ClusterSlot)
    if sloterr != nil {
        list.Shutdown()
        return nil, sloterr
    }

    memList := strings.Split(conf.ClusterMemebers, ",")
    var validMembers []string
    for _, v := range memList {
        mem := strings.TrimSpace(v)
        if mem != "" {
            validMembers = append(validMembers, mem)
        }
    }
    if len(validMembers) > 0 {
        list.Join(validMembers)
    }

    return (*members)(list), nil
}

func (c *members)LocalAddr() string {
    return (*memberlist.Memberlist)(c).LocalNode().Address()
}

func (c *members)Enabled() bool {
    return true
}

func (c *members) UpdateLocal(meta []byte) error {
    (*memberlist.Memberlist)(c).LocalNode().Meta = meta
    return nil
}

func (c *members) UpdateAndWait(meta []byte, timeout time.Duration) error {
    (*memberlist.Memberlist)(c).LocalNode().Meta = meta
    return (*memberlist.Memberlist)(c).UpdateNode(timeout)
}

func (c *members) Close() error {
    return (*memberlist.Memberlist)(c).Shutdown()
}

type DummyCluster int

func (c *DummyCluster) LocalAddr() string {
    return ""
}

func (c *DummyCluster) UpdateLocal(meta []byte) error {
    return nil
}

func (c *DummyCluster) UpdateAndWait(meta []byte, timeout time.Duration) error {
    return nil
}

func (c *DummyCluster)Enabled() bool {
    return false
}

func (c *DummyCluster) Close() error {
    return nil
}
