package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
	"net/http"
	"strconv"
)

type Tweet struct {
	gorm.Model
	Content string	`form:"content" binding:"required"`
}

func connectGorm() *gorm.DB {
	dbUser := "norun"
	dbPass := "tomoya0128"
	dbProtocol := "tcp(127.0.0.1:3306)"
	dbName := "tweet_test"
	connectTemplate := "%s:%s@%s/%s"
	connect := fmt.Sprintf(connectTemplate, dbUser, dbPass, dbProtocol, dbName)
	db, err := gorm.Open("mysql", connect)

	if err != nil {
		panic(err.Error())
	}
	return db
}

func dbInit() {
	db := connectGorm()
	defer db.Close()
	db.AutoMigrate(&Tweet{}) //構造体に基づいてテーブル作成
}

//データインサート
func dbInsert(content string) {
	db := connectGorm()
	defer db.Close()
	//Insert処理
	db.Create(&Tweet{Content: content})
}

//db更新
func dbUpdate(id int, tweetContent string) {
	db := connectGorm()
	var tweet Tweet
	db.First(&tweet, id)
	tweet.Content = tweetContent
	db.Save(&tweet)
	db.Close()
}

func dbGetAll() []Tweet {
	db := connectGorm()
	defer db.Close()
	var tweets []Tweet
	db.Order("created_at desc").Find(&tweets)
	return tweets
}

//db一つ取得
func dbGetOne(id int) Tweet {
	db := connectGorm()
	defer db.Close()
	var tweet Tweet
	db.First(&tweet, id)
	db.Close()
	return tweet
}

//db削除
func dbDelete(id int) {
	db := connectGorm()
	defer db.Close()
	var tweet Tweet
	db.First(&tweet, id)
	db.Delete(&tweet)
}

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("views/*.html")

	dbInit()

	//一覧
	router.GET("/", func(c *gin.Context) {
		tweets := dbGetAll()
		c.HTML(200, "index.html", gin.H{"tweets": tweets})
	})

	//登録
	router.POST("/new", func(c *gin.Context) {
		var form Tweet

		//validation
		if err := c.Bind(&form); err != nil {
			tweets := dbGetAll()
			c.HTML(http.StatusBadRequest, "index.html", gin.H{"tweets": tweets, "err": err})
			c.Abort()
		} else {
			content := c.PostForm("content")
			dbInsert(content)
			//302一時的なリダイレクト
			c.Redirect(302, "/")
		}
	})

	//投稿詳細
	router.GET("/detail/:id", func(c *gin.Context) {
		n := c.Param("id")
		id, err := strconv.Atoi(n)
		if err != nil {
			panic(err)
		}
		tweet := dbGetOne(id)
		c.HTML(200, "detail.html", gin.H{"tweet": tweet})
	})

	//更新
	router.POST("/update/:id", func(c *gin.Context) {
		n := c.Param("id")
		id, err := strconv.Atoi(n)
		if err != nil {
			panic(err)
		}
		tweet := c.PostForm("tweet")
		dbUpdate(id, tweet)
		c.Redirect(302, "/")
	})

	//削除確認
	router.GET("/delete_check/:id", func(c *gin.Context) {
		n := c.Param("id")
		id , err := strconv.Atoi(n)
		if err != nil {
			panic(err)
		}
		tweet := dbGetOne(id)
		c.HTML(200, "delete.html", gin.H{"tweet": tweet})
	})

	//削除
	router.POST("/delete/:id", func(c *gin.Context) {
		n := c.Param("id")
		id, err := strconv.Atoi(n)
		if err != nil {
			panic(err)
		}
		dbDelete(id)
		c.Redirect(302, "/")
	})

	router.Run()
}

