// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package cluster

import (
    "errors"
    "fmt"
    "gache/config"
    "io"
    "io/ioutil"
    "net/http"
    "os"
    "strconv"
    "strings"
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

func GetSlots(slotStr string) (uint32, uint32, error) {
    slots := strings.Split(slotStr, "-")
    beginSlot, endSlot := 0, 0
    if len(slots) > 1 {
        i, err := strconv.Atoi(slots[0])
        if err != nil {
            return 0, 0, err
        }
        beginSlot = i
        j, err := strconv.Atoi(slots[1])
        if err != nil {
            return 0, 0, err
        }
        endSlot = j
    } else {
        return 0, 0, errors.New("Parse slot error")
    }
    return uint32(beginSlot), uint32(endSlot), nil
}
