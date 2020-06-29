package main

import (
	"crypto/sha1"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/olahol/go-imageupload"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
	"tweet/crypto"
)

type Tweet struct {
	gorm.Model
	Content string	`form:"content" binding:"required"`
	Image   *multipart.FileHeader  `form:"image"`
}

type User struct {
	gorm.Model
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func connectGorm() *gorm.DB {
	dbUser := "norun"
	dbPass := "tomoya0128"
	dbProtocol := "tcp(127.0.0.1:3306)"
	dbName := "tweet_test"
	parseTime := "parseTime=true"
	connectTemplate := "%s:%s@%s/%s?%s"
	connect := fmt.Sprintf(connectTemplate, dbUser, dbPass, dbProtocol, dbName, parseTime)
	db, err := gorm.Open("mysql", connect)

	if err != nil {
		panic(err.Error())
	}
	return db
}

func dbInit() {
	db := connectGorm()
	defer db.Close()
	db.AutoMigrate(&Tweet{}, &User{}) //構造体に基づいてテーブル作成
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

//ユーザー登録処理
func createUser(username, password string) []error {
	passwordEncrypt, _ := crypto.PasswordEncrypt(password)
	db := connectGorm()
	defer db.Close()
	// Insert処理
	if err := db.Create(&User{Username: username, Password: passwordEncrypt}).GetErrors(); err != nil {
		return err
	}
	return nil
}

func getUser(username string) User {
	db := connectGorm()
	defer db.Close()
	var user User
	db.First(&user, "username = ?", username)
	return user
}

/*
-------------------------------------------------------------------------
*/

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("views/*.html")

	//router.Use(static.Serve("/", static.LocalFile("./assets", true)))

	dbInit()

	//一覧
	router.GET("/", func(c *gin.Context) {
		tweets := dbGetAll()
		c.HTML(200, "index.html", gin.H{"tweets": tweets})
	})

	//登録
	router.POST("/new", func(c *gin.Context) {
		var form Tweet
		img, err := imageupload.Process(c.Request, "image") //fieldにはname属性をいれる
		if err != nil {
			panic(err)
		}
		//validation
		if err := c.Bind(&form); err != nil {
			tweets := dbGetAll()
			c.HTML(http.StatusBadRequest, "index.html", gin.H{"tweets": tweets, "err": err})
			c.Abort()
		} else {
			thumb, err := imageupload.ThumbnailPNG(img, 300, 300)
			if err != nil {
				panic(err)
			}
			h := sha1.Sum(thumb.Data)
			thumb.Save(fmt.Sprintf("files/%s_%x.png",
				time.Now().Format("20060102150405"), h[:4]))
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

	//ユーザー登録画面
	router.GET("/signup", func(c *gin.Context) {
		c.HTML(200, "signup.html", gin.H{})
	})

	//ユーザー登録
	router.POST("/signup", func(c *gin.Context) {
		var form User
		//validation
		if err := c.Bind(&form); err != nil {
			c.HTML(http.StatusBadRequest, "signup.html", gin.H{"err": err})
			c.Abort()
		} else {
			username := c.PostForm("username")
			password := c.PostForm("password")
			//登録ユーザーが重複していた場合に弾く処理
			if err := createUser(username, password); err != nil {
				c.HTML(http.StatusBadRequest, "signup.html", gin.H{"err": err})
			}
			c.Redirect(302, "/")
		}
	})

	router.GET("/login", func(c *gin.Context) {
		c.HTML(200, "login.html", gin.H{})
	})

	//ユーザーログイン
	router.POST("/login", func(c *gin.Context) {
		//DBから取得したユーザーパスワード(Hash)
		dbPassword := getUser(c.PostForm("username")).Password
		log.Println(dbPassword)
		//フォームから取得したユーザーパスワード
		formPassword := c.PostForm("password")

		hash := []byte(dbPassword)

		fmt.Println(string(hash))
		//ユーザーパスワードの比較
		if err := crypto.CompareHashAndPassword(dbPassword, formPassword); err != nil {
			log.Println("Login ERROR")
			c.HTML(http.StatusBadRequest, "login.html", gin.H{"err": err})
			c.Abort()
		} else {
			log.Println("Success!!")
			c.Redirect(302, "/")
		}
	})

	router.Run()
}

