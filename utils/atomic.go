// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description: 

package utils

import "sync/atomic"

type AtomicBool int32

func (b *AtomicBool) IsSet() bool { return atomic.LoadInt32((*int32)(b)) == 1 }
func (b *AtomicBool) Set()        { atomic.StoreInt32((*int32)(b), 1) }
func (b *AtomicBool) Unset()      { atomic.StoreInt32((*int32)(b), 0) }
