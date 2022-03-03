package main

import (
	"dtm/dtmcli"
	"errors"
	"fmt"
	"gitee.com/yanggit123/tool"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GormUser struct {
	tool.Model2
	Name  string `json:"name" comment:""`
	Money int    `json:"money"`
}

/*
SUCCESS: 状态码 200 StatusOK
FAILURE: 状态码 409 StatusConflict
ONGOING: 状态码 425 StatusTooEarly
*/
func main() {
	db, _ := tool.EnableMysql2(tool.MysqlConf{
		Address:         "",
		DbName:          "mall",
		Username:        "",
		Password:        "",
		Prefix:          "",
		MaxOpenConns:    100,
		MaxIdleConns:    100,
		ConnMaxLifetime: 100,
		IsLogMode:       true,
		IsSingular:      true,
	})
	db.AutoMigrate(&GormUser{})
	db.Create(&[]GormUser{
		{Model2: tool.Model2{ID: 1}, Name: "gy", Money: 100},
		{Model2: tool.Model2{ID: 2}, Name: "gy1", Money: 100},
	})
	//db1:=db.Exec("insert into gorm_user(name,money) values('gy2',100)")
	//fmt.Println(db.RowsAffected,db1.RowsAffected)
	r := gin.New()
	r.POST("/api/busi_saga/TransOut", func(c *gin.Context) {
		barrier, err := dtmcli.BarrierFrom(c.Query("trans_type"), c.Query("gid"), c.Query("branch_id"), c.Query("op"))
		if err != nil {
			fmt.Println("TransOut1=============", err.Error())
			return
		}
		// 开启子事务屏障
		if err := barrier.CallWithDB(db, func(tx *gorm.DB) error {
			gormUser := GormUser{}
			db.Where("id=1").First(&gormUser)
			if gormUser.Money < 30 {
				return errors.New("money不够")
			}
			err := db.Model(&GormUser{}).Where("id=1").Update("money", gorm.Expr("money-?", 30)).Error
			if err != nil {
				return err
			}
			return nil
		}); err != nil {
			fmt.Println("TransOut3===============")
			c.JSON(409, "")
			return
		}

		c.JSON(200, "")

	})
	r.POST("/api/busi_saga/TransOutCompensate", func(c *gin.Context) {
		barrier, err := dtmcli.BarrierFrom(c.Query("trans_type"), c.Query("gid"), c.Query("branch_id"), c.Query("op"))
		if err != nil {
			fmt.Println("TransOut1=============", err.Error())
			return
		}
		// 开启子事务屏障
		if err := barrier.CallWithDB(db, func(tx *gorm.DB) error {
			err := db.Model(&GormUser{}).Where("id=1").Update("money", gorm.Expr("money+?", 30)).Error
			if err != nil {
				fmt.Println("TransOutCompensate2===============")
				return err
			}
			return nil
		}); err != nil {
			fmt.Println("TransOutCompensate3===============")
			c.JSON(500, "")
			return
		}

		c.JSON(200, "")
	})
	r.POST("/api/busi_saga/TransIn", func(c *gin.Context) {
		barrier, err := dtmcli.BarrierFrom(c.Query("trans_type"), c.Query("gid"), c.Query("branch_id"), c.Query("op"))
		if err != nil {
			fmt.Println("TransOut1=============", err.Error())
			return
		}
		// 开启子事务屏障
		if err := barrier.CallWithDB(db, func(tx *gorm.DB) error {
			gormUser := GormUser{}
			db.Where("id=2").First(&gormUser)
			if gormUser.Money > 100 {
				return errors.New("钱太多")
			}
			err := db.Model(&GormUser{}).Where("id=2").Update("money", gorm.Expr("money+?", 30)).Error
			if err != nil {
				return err
			}
			return nil
		}); err != nil {
			fmt.Println("TransIn2===============")
			c.JSON(409, "")
			return
		}
		c.JSON(200, "")
	})
	r.POST("/api/busi_saga/TransInCompensate", func(c *gin.Context) {
		barrier, err := dtmcli.BarrierFrom(c.Query("trans_type"), c.Query("gid"), c.Query("branch_id"), c.Query("op"))
		if err != nil {
			fmt.Println("TransOut1=============", err.Error())
			return
		}
		// 开启子事务屏障
		if err := barrier.CallWithDB(db, func(tx *gorm.DB) error {
			err := db.Model(&GormUser{}).Where("id=2").Update("money", gorm.Expr("money-?", 30)).Error
			if err != nil {
				fmt.Println("TransInCompensate2===============")
				return err
			}
			return nil
		}); err != nil {
			fmt.Println("TransInCompensate3===============")
			c.JSON(500, "")
			return
		}
		c.JSON(200, "")
	})
	r.Run("0.0.0.0:8081")
}
