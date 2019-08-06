// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package cluster

import (
    "errors"
    "gache/config"
    "gache/db"
    "github.com/hashicorp/go-hclog"
    "github.com/hashicorp/raft"
    "github.com/hashicorp/raft-boltdb"
    "net"
    "os"
    "path/filepath"
    "time"
)

type Replication interface {
    Apply(cmd []byte, timeout time.Duration) error
    Join(addr string) error
    Listen(listener func(bool))
    Shutdown() error
}

type RaftReplication struct {
    r *raft.Raft
    c chan bool
}

func (r *RaftReplication) Apply(cmd []byte, timeout time.Duration) error {
    return r.r.Apply(cmd, timeout).Error()
}

func (r *RaftReplication) Join(addr string) error {
    return DoJoin(addr, r.r)
}

func (r *RaftReplication) Shutdown() error {
    return r.r.Shutdown().Error()
}

func (r *RaftReplication) Listen(listener func(bool)) {
    go func() {
        for {
            select {
            case v, ok := <-r.c:
                if !ok {
                    return
                }
                listener(v)
            default:
                break
            }
        }
    }()
}

func New(conf *config.Config, db *db.GacheDb, notifyChan chan bool) (Replication, error) {
    raftConfig := raft.DefaultConfig()
    raftConfig.Logger = hclog.New(&hclog.LoggerOptions{
        Name:   "",
        Output: os.Stdout,
    })
    raftConfig.LocalID = raft.ServerID(conf.RaftTcpAddr)
    if notifyChan != nil {
        raftConfig.NotifyCh = notifyChan
    }

    if !IsPathExists(conf.RaftDir) {
        Mkdir(conf.RaftDir)
    }

    logStore, err := raftboltdb.NewBoltStore(filepath.Join(conf.RaftDir, "raft-log.bolt"))
    if err != nil {
        return nil, err
    }

    stableStore, err := raftboltdb.NewBoltStore(filepath.Join(conf.RaftDir, "raft-stable.bolt"))
    if err != nil {
        return nil, err
    }

    snapshotStore, err := raft.NewFileSnapshotStore(conf.RaftDir, 1, os.Stderr)
    if err != nil {
        return nil, err
    }

    transport, err := newRaftTransport(conf)
    if err != nil {
        return nil, err
    }

    if conf.RaftJoinAddr == "" {
        configuration := raft.Configuration{
            Servers: []raft.Server{
                {
                    ID:      raftConfig.LocalID,
                    Address: transport.LocalAddr(),
                },
            },
        }
        raft.BootstrapCluster(raftConfig, logStore, stableStore, snapshotStore, transport, configuration)
    }
    r, err := raft.NewRaft(raftConfig, &GacheFSM{db: db}, logStore, stableStore, snapshotStore, transport)
    return &RaftReplication{r: r, c: notifyChan}, err
}

func DoJoin(addr string, cluster *raft.Raft) error {
    if cluster == nil {
        return errors.New("raft is nil")
    }
    addPeerFuture := cluster.AddVoter(raft.ServerID(addr),
        raft.ServerAddress(addr),
        0, 0)

    return addPeerFuture.Error()
}

func newRaftTransport(conf *config.Config) (*raft.NetworkTransport, error) {
    address, err := net.ResolveTCPAddr("tcp", conf.RaftTcpAddr)
    if err != nil {
        return nil, err
    }
    transport, err := raft.NewTCPTransport(address.String(), address, 3, 10*time.Second, os.Stderr)
    if err != nil {
        return nil, err
    }
    return transport, nil
}
