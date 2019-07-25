// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package cluster

import (
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

func New(conf *config.Config, db *db.GacheDb, notifyChan chan bool) (*raft.Raft, error) {
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
    return raft.NewRaft(raftConfig, &GacheFSM{db: db}, logStore, stableStore, snapshotStore, transport)
}

func DoJoin(addr string, cluster *raft.Raft) error {
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
