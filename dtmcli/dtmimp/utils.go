/*
 * Copyright (c) 2021 yedf. All rights reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package dtmimp

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/guanyang-lab/dtm/dtmcli/logger"
)

// Logf an alias of Infof
// Deprecated: use logger.Errorf
var Logf = logger.Infof

// LogRedf an alias of Errorf
// Deprecated: use logger.Errorf
var LogRedf = logger.Errorf

// FatalIfError fatal if error is not nil
// Deprecated: use logger.FatalIfError
var FatalIfError = logger.FatalIfError

// LogIfFatalf fatal if cond is true
// Deprecated: use logger.FatalfIf
var LogIfFatalf = logger.FatalfIf

// AsError wrap a panic value as an error
func AsError(x interface{}) error {
	logger.Errorf("panic wrapped to error: '%v'", x)
	if e, ok := x.(error); ok {
		return e
	}
	return fmt.Errorf("%v", x)
}

// P2E panic to error
func P2E(perr *error) {
	if x := recover(); x != nil {
		*perr = AsError(x)
	}
}

// E2P error to panic
func E2P(err error) {
	if err != nil {
		panic(err)
	}
}

// CatchP catch panic to error
func CatchP(f func()) (rerr error) {
	defer P2E(&rerr)
	f()
	return nil
}

// PanicIf name is clear
func PanicIf(cond bool, err error) {
	if cond {
		panic(err)
	}
}

// MustAtoi is string to int
func MustAtoi(s string) int {
	r, err := strconv.Atoi(s)
	if err != nil {
		E2P(errors.New("convert to int error: " + s))
	}
	return r
}

// OrString return the first not empty string
func OrString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// If ternary operator
func If(condition bool, trueObj interface{}, falseObj interface{}) interface{} {
	if condition {
		return trueObj
	}
	return falseObj
}

// MustMarshal checked version for marshal
func MustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	E2P(err)
	return b
}

// MustMarshalString string version of MustMarshal
func MustMarshalString(v interface{}) string {
	return string(MustMarshal(v))
}

// MustUnmarshal checked version for unmarshal
func MustUnmarshal(b []byte, obj interface{}) {
	err := json.Unmarshal(b, obj)
	E2P(err)
}

// MustUnmarshalString string version of MustUnmarshal
func MustUnmarshalString(s string, obj interface{}) {
	MustUnmarshal([]byte(s), obj)
}

// MustRemarshal marshal and unmarshal, and check error
func MustRemarshal(from interface{}, to interface{}) {
	b, err := json.Marshal(from)
	E2P(err)
	err = json.Unmarshal(b, to)
	E2P(err)
}

// GetFuncName get current call func name
func GetFuncName() string {
	pc, _, _, _ := runtime.Caller(1)
	nm := runtime.FuncForPC(pc).Name()
	return nm[strings.LastIndex(nm, ".")+1:]
}

// MayReplaceLocalhost when run in docker compose, change localhost to host.docker.internal for accessing host network
func MayReplaceLocalhost(host string) string {
	if os.Getenv("IS_DOCKER") != "" {
		return strings.Replace(strings.Replace(host,
			"localhost", "host.docker.internal", 1),
			"127.0.0.1", "host.docker.internal", 1)
	}
	return host
}

var sqlDbs sync.Map

// PooledDB get pooled gorm.DB
func PooledDB(conf DBConf) (*gorm.DB, error) {
	dsn := GetDsn(conf)
	db, ok := sqlDbs.Load(dsn)
	if !ok {
		db2, err := StandaloneDB(conf)
		if err != nil {
			return nil, err
		}
		db = db2
		sqlDbs.Store(dsn, db)
	}
	return db.(*gorm.DB), nil
}

// StandaloneDB get a standalone db instance
func StandaloneDB(conf DBConf) (*gorm.DB, error) {
	dsn := GetDsn(conf)
	logger.Infof("opening standalone %s: %s", conf.Driver, strings.Replace(dsn, conf.Password, "****", 1))
	//sql.Open(conf.Driver, dsn)
	return gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,   // DSN data source name
		DefaultStringSize:         256,   // string 类型字段的默认长度
		DisableDatetimePrecision:  true,  // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,  // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,  // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false, // 根据版本自动配置
	}), &gorm.Config{
		//SkipDefaultTransaction: true,为了确保数据一致性，GORM 会在事务里执行写入操作（创建、更新、删除）。如果没有这方面的要求，您可以在初始化时禁用它。
		NamingStrategy: schema.NamingStrategy{ //GORM 允许用户通过覆盖默认的命名策略更改默认的命名约定，这需要实现接口 Namer
			TablePrefix:   "",   // 表名前缀，`User` 的表名应该是 `t_users`
			SingularTable: true, // 使用单数表名，启用该选项，此时，`User` 的表名应该是 `t_user`
		},
		DisableForeignKeyConstraintWhenMigrating: true, //注意 AutoMigrate 会自动创建数据库外键约束，您可以在初始化时禁用此功能
	})
}

// DBExec use raw db to exec
func DBExec(db DB, sql string, values ...interface{}) (affected int64, rerr error) {
	if sql == "" {
		return 0, nil
	}
	began := time.Now()
	sql = GetDBSpecial().GetPlaceHoldSQL(sql)
	r, rerr := db.Exec(sql, values...)
	used := time.Since(began) / time.Millisecond
	if rerr == nil {
		affected, rerr = r.RowsAffected()
		logger.Debugf("used: %d ms affected: %d for %s %v", used, affected, sql, values)
	} else {
		logger.Errorf("used: %d ms exec error: %v for %s %v", used, rerr, sql, values)
	}
	return
}

// GetDsn get dsn from map config
func GetDsn(conf DBConf) string {
	host := MayReplaceLocalhost(conf.Host)
	driver := conf.Driver
	dsn := map[string]string{
		"mysql": fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true",
			conf.User, conf.Password, host, conf.Port, ""),
		"postgres": fmt.Sprintf("host=%s user=%s password=%s dbname='%s' port=%d sslmode=disable",
			host, conf.User, conf.Password, "", conf.Port),
	}[driver]
	PanicIf(dsn == "", fmt.Errorf("unknow driver: %s", driver))
	return dsn
}

// RespAsErrorCompatible translate a resty response to error
// compatible with version < v1.10
func RespAsErrorCompatible(resp *resty.Response) error {
	code := resp.StatusCode()
	str := resp.String()
	if code == http.StatusTooEarly || strings.Contains(str, ResultOngoing) {
		return fmt.Errorf("%s. %w", str, ErrOngoing)
	} else if code == http.StatusConflict || strings.Contains(str, ResultFailure) {
		return fmt.Errorf("%s. %w", str, ErrFailure)
	} else if code != http.StatusOK {
		return errors.New(str)
	}
	return nil
}

// DeferDo a common defer do used in dtmcli/dtmgrpc
func DeferDo(rerr *error, success func() error, fail func() error) {
	defer func() {
		if x := recover(); x != nil {
			_ = fail()
			panic(x)
		} else if *rerr != nil {
			_ = fail()
		} else {
			*rerr = success()
		}
	}()
}
