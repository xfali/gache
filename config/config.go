// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package config

type Config struct {
    RaftTcpAddr  string
    RaftDir      string
    RaftJoinAddr string

    ClusterPort     int
    ClusterMemebers string
    ClusterSlot     string

    ApiPort int
}
