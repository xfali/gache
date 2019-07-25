// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package cluster

import (
    "fmt"
    "gache/config"
    "io"
    "io/ioutil"
    "net/http"
    "os"
)

func Join(conf *config.Config) error {
    url := fmt.Sprintf("http://%s/join?addr=%s",
        conf.RaftJoinAddr,
        conf.RaftTcpAddr)
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    io.Copy(ioutil.Discard, resp.Body)

    return nil
}

// 判断文件夹是否存在
func IsPathExists(path string) bool {
    _, err := os.Stat(path)
    if err == nil {
        return true
    }
    if os.IsNotExist(err) {
        return false
    }
    return false
}

func Mkdir(path string) error {
    return os.MkdirAll(path, os.ModePerm)
}
