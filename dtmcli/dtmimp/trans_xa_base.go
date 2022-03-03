/*
 * Copyright (c) 2021 yedf. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package dtmimp

import (
	"gorm.io/gorm"
	"strings"
)

// XaClientBase XaClient/XaGrpcClient base. shared by http and grpc
type XaClientBase struct {
	Server    string
	Conf      DBConf
	NotifyURL string
}

// HandleCallback Handle the callback of commit/rollback
func (xc *XaClientBase) HandleCallback(gid string, branchID string, action string) error {
	db, err := StandaloneDB(xc.Conf)
	if err != nil {
		return err
	}
	defer func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}()
	xaID := gid + "-" + branchID
	//_, err = DBExec(db, GetDBSpecial().GetXaSQL(action, xaID))
	err = db.Exec(GetDBSpecial().GetXaSQL(action, xaID)).Error
	if err != nil &&
		(strings.Contains(err.Error(), "XAER_NOTA") || strings.Contains(err.Error(), "does not exist")) { // Repeat commit/rollback with the same id, report this error, ignore
		err = nil
	}
	return err
}

// HandleLocalTrans public handler of LocalTransaction via http/grpc
func (xc *XaClientBase) HandleLocalTrans(xa *TransBase, cb func(*gorm.DB) error) (rerr error) {
	xaBranch := xa.Gid + "-" + xa.BranchID
	db, rerr := StandaloneDB(xc.Conf)
	if rerr != nil {
		return
	}
	defer func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}()
	defer DeferDo(&rerr, func() error {
		//_, err := DBExec(db, GetDBSpecial().GetXaSQL("prepare", xaBranch))
		err := db.Exec(GetDBSpecial().GetXaSQL("prepare", xaBranch)).Error
		return err
	}, func() error {
		return nil
	})
	//	_, rerr = DBExec(db, GetDBSpecial().GetXaSQL("start", xaBranch))
	rerr = db.Exec(GetDBSpecial().GetXaSQL("start", xaBranch)).Error
	if rerr != nil {
		return
	}
	defer func() {
		//_, _ = DBExec(db, GetDBSpecial().GetXaSQL("end", xaBranch))
		db.Exec(GetDBSpecial().GetXaSQL("end", xaBranch))
	}()
	rerr = cb(db)
	return
}

// HandleGlobalTrans http/grpc GlobalTransaction的公共方法
func (xc *XaClientBase) HandleGlobalTrans(xa *TransBase, callDtm func(string) error, callBusi func() error) (rerr error) {
	rerr = callDtm("prepare")
	if rerr != nil {
		return
	}
	// 小概率情况下，prepare成功了，但是由于网络状况导致上面Failure，那么不执行下面defer的内容，等待超时后再回滚标记事务失败，也没有问题
	defer DeferDo(&rerr, func() error {
		return callDtm("submit")
	}, func() error {
		return callDtm("abort")
	})
	rerr = callBusi()
	return
}
