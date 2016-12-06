package main

import (
        "database/sql"
        "fmt"
        "log"
        "strconv"
		"gopkg.in/appleboy/gin-jwt.v2"
		"time"
        "github.com/gin-gonic/gin"
        "github.com/gin-gonic/contrib/cache"
        _ "github.com/go-sql-driver/mysql"
        gorp "gopkg.in/gorp.v2"
)

type Blog struct {
        Id           int64  `db:"id" json:"id"`
        Created_time string `db:"createdtime" json:"createdtime"`
        Updated_time string `db:"updatedtime" json:"updatedtime"`
        Created_by   string `db:"createdby" json:"createdby"`
        Updated_by   string `db:"updatedby" json:"updatedby"`
        Title        string `db:"title" json:"title"`
        Text_html    string `db:"texthtml" json:"texthtml"`
        Author       string `db:"author" json:"author"`
}

type Portfolio struct {
        Id           int64  `db:"id" json:"id"`
        Created_time string `db:"createdtime" json:"createdtime"`
        Updated_time string `db:"updatedtime" json:"updatedtime"`
        Created_by   string `db:"createdby" json:"createdby"`
        Updated_by   string `db:"updatedby" json:"updatedby"`
        Name         string `db:"name" json:"name"`
        Description  string `db:"description" json:"description"`
        Text_html    string `db:"texthtml" json:"texthtml"`
        Demo_url     string `db:"demourl" json:"demourl"`
        Author       string `db:"author" json:"author"`
}

var dbmap = initDb()

func initDb() *gorp.DbMap {
        db, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/database?charset=utf8")
        checkErr(err, "sql.Open failed")
        dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
        err = addTable(dbmap, Blog{}, "Blog", "Id")
        checkErr(err, "Create table failed")
        err = addTable(dbmap, Portfolio{}, "Portfolio", "Id")
        checkErr(err, "Create table failed")
        return dbmap
}

func addTable(db *gorp.DbMap, i interface{}, name string, keyName string) (err error) {
        db.AddTableWithName(i, name).SetKeys(true, keyName)
        err = db.CreateTablesIfNotExists()
        return err
}

func checkErr(err error, msg string) {
        if err != nil {
                log.Fatalln(msg, err)
        }
}

func Cors() gin.HandlerFunc {
        return func(c *gin.Context) {
                c.Writer.Header().Add("Access-Control-Allow-Origin", "*")
                c.Next()
        }
}

func main() {
        r := gin.New()
        r.Use(gin.Logger())
        r.Use(gin.Recovery())
        r.Use(Cors())

        // the gin-gonic cache
        store := cache.NewInMemoryStore(time.Second)

        // the jwt middleware
		authMiddleware := &jwt.GinJWTMiddleware{
			Realm:      "test zone",
			Key:        []byte("secret key"),
			Timeout:    time.Hour,
			MaxRefresh: time.Hour,
			Authenticator: func(userId string, password string, c *gin.Context) (string, bool) {
				if (userId == "admin" && password == "admin") || (userId == "test" && password == "test") {
					return userId, true
				}

				return userId, false
			},
			Authorizator: func(userId string, c *gin.Context) bool {
				if userId == "admin" {
					return true
				}

				return false
			},
			Unauthorized: func(c *gin.Context, code int, message string) {
				c.JSON(code, gin.H{
					"code":    code,
					"message": message,
				})
			},
			// TokenLookup is a string in the form of "<source>:<name>" that is used
			// to extract token from the request.
			// Optional. Default value "header:Authorization".
			// Possible values:
			// - "header:<name>"
			// - "query:<name>"
			// - "cookie:<name>"
			TokenLookup: "header:Authorization",
			// TokenLookup: "query:token",
			// TokenLookup: "cookie:token",
		}
		
		r.POST("api/login", authMiddleware.LoginHandler)

        v1 := r.Group("api/v1")
        v1.Use(authMiddleware.MiddlewareFunc())
        {
       		v1.GET("/auth/refresh_token", authMiddleware.RefreshHandler)
                v1.GET("/blogs", cache.CachePage(store, time.Minute, GetBlogs))
                v1.GET("/blogs/:id", cache.CachePage(store, time.Minute, GetBlog))
                v1.POST("/blogs", PostBlog)
                v1.PUT("/blogs/:id", cache.CachePage(store, time.Minute, UpdateBlog))
                v1.DELETE("/blogs/:id", DeleteBlog)
                v1.OPTIONS("/blogs", cache.CachePage(store, time.Minute, OptionsBlog))     // POST
                v1.OPTIONS("/blogs/:id", cache.CachePage(store, time.Minute, OptionsBlog)) // PUT, DELETE
                v1.GET("/portfolios", cache.CachePage(store, time.Minute, GetPortfolios))
                v1.GET("/portfolios/:id", cache.CachePage(store, time.Minute, GetPortfolio))
                v1.POST("/portfolios", PostPortfolio)
                v1.PUT("/portfolios/:id", UpdatePortfolio)
                v1.DELETE("/portfolios/:id", DeletePortfolio)
                v1.OPTIONS("/portfolios", cache.CachePage(store, time.Minute, OptionsPortfolio))     // POST
                v1.OPTIONS("/portfolios/:id", cache.CachePage(store, time.Minute, OptionsPortfolio)) // PUT, DELETE
                v1.GET("/latest/portfolios/:count/:offset", cache.CachePage(store, time.Minute, GetPortfolioSet))
                v1.GET("/latest/blogs/:count/:offset", cache.CachePage(store, time.Minute, GetBlogSet))
        }

        r.Run(":8080")
}

func GetBlogs(c *gin.Context) {
        var blogs []Blog
        _, err := dbmap.Select(&blogs, "SELECT * FROM Blog")

        if err == nil {
                c.JSON(200, blogs)
        } else {
                c.JSON(404, gin.H{"error": "no blog(s) into the table"})
        }

        // curl -i http://localhost:8080/api/v1/blogs
}

func GetBlog(c *gin.Context) {
        id := c.Params.ByName("id")
        var blog Blog
        err := dbmap.SelectOne(&blog, "SELECT * FROM Blog WHERE id=? LIMIT 1", id)

        if err == nil {
                blog_id, _ := strconv.ParseInt(id, 0, 64)

                content := &Blog{
                        Id:           blog_id,
                        Created_time: blog.Created_time,
                        Updated_time: blog.Updated_time,
                        Created_by:   blog.Created_by,
                        Updated_by:   blog.Updated_by,
                        Title:        blog.Title,
                        Text_html:    blog.Text_html,
                        Author:       blog.Author,
                }
                c.JSON(200, content)
        } else {
                fmt.Println(err)
                c.JSON(404, gin.H{"error": "blog not found"})
        }

        // curl -i http://localhost:8080/api/v1/blogs/1
}

func PostBlog(c *gin.Context) {
        var blog Blog
        //x, _ := ioutil.ReadAll(c.Request.Body)
        //fmt.Printf("%s", string(x))

        c.Bind(&blog)

        log.Println(blog)

        if blog.Created_by == "" {
                blog.Created_by = "Anonymous"
        }

        if blog.Updated_by == "" {
                blog.Updated_by = blog.Created_by
        }

        if blog.Author == "" {
                blog.Author = blog.Created_by
        }

        log.Println(blog.Title)

        if blog.Created_by != "" && blog.Title != "" && blog.Text_html != "" && blog.Updated_by != "" && blog.Author != "" {

                if insert, _ := dbmap.Exec(`INSERT INTO Blog (createdby, updatedby, title, texthtml, author) VALUES (?, ?, ?, ?, ?)`, blog.Created_by, blog.Updated_by, blog.Title, blog.Text_html, blog.Author); insert != nil {
                        blog_id, err := insert.LastInsertId()
                        if err == nil {
                                content := &Blog{
                                        Id:           blog_id,
                                        Created_time: blog.Created_time,
                                        Updated_time: blog.Updated_time,
                                        Created_by:   blog.Created_by,
                                        Updated_by:   blog.Updated_by,
                                        Title:        blog.Title,
                                        Text_html:    blog.Text_html,
                                        Author:       blog.Author,
                                }
                                c.JSON(201, content)
                                return
                        } else {
                                checkErr(err, "Insert failed")
                        }
                }

        } else {
                c.JSON(400, gin.H{"error": "Fields are empty"})
                return
        }

        // curl -i -X POST -H "Content-Type: application/json" -d "{ \"firstname\": \"Thea\", \"lastname\": \"Queen\" }" http://localhost:8080/api/v1/blogs
}

func UpdateBlog(c *gin.Context) {
        id := c.Params.ByName("id")
        var blog Blog
        err := dbmap.SelectOne(&blog, "SELECT * FROM Blog WHERE id=?", id)

        if err == nil {
                var json Blog
                c.Bind(&json)

                blog_id, _ := strconv.ParseInt(id, 0, 64)

                blog := Blog{
                        Id:           blog_id,
                        Created_time: json.Created_time,
                        Updated_time: json.Updated_time,
                        Created_by:   json.Created_by,
                        Updated_by:   json.Updated_by,
                        Title:        json.Title,
                        Text_html:    json.Text_html,
                        Author:       json.Author,
                }

                if blog.Created_by == "" {
                        blog.Created_by = "Anonymous"
                }

                if blog.Updated_by == "" {
                        blog.Updated_by = blog.Created_by
                }

                if blog.Author == "" {
                        blog.Author = blog.Created_by
                }

                if blog.Created_by != "" && blog.Updated_by != "" && blog.Title != "" && blog.Text_html != "" {
                        _, err = dbmap.Update(&blog)

                        if err == nil {
                                c.JSON(200, blog)
                        } else {
                                checkErr(err, "Updated failed")
                        }

                } else {
                        c.JSON(400, gin.H{"error": "fields are empty"})
                }

        } else {
                c.JSON(404, gin.H{"error": "blog not found"})
        }

        // curl -i -X PUT -H "Content-Type: application/json" -d "{ \"firstname\": \"Thea\", \"lastname\": \"Merlyn\" }" http://localhost:8080/api/v1/users/1
}

func DeleteBlog(c *gin.Context) {
        id := c.Params.ByName("id")

        var blog Blog
        err := dbmap.SelectOne(&blog, "SELECT * FROM Blog WHERE id=?", id)

        if err == nil {
                _, err = dbmap.Delete(&blog)

                if err == nil {
                        c.JSON(200, gin.H{"id #" + id: "deleted"})
                } else {
                        checkErr(err, "Delete failed")
                }

        } else {
                c.JSON(404, gin.H{"error": "blog not found"})
        }

        // curl -i -X DELETE http://localhost:8080/api/v1/users/1
}

func GetPortfolios(c *gin.Context) {
        var portfolios []Portfolio
        _, err := dbmap.Select(&portfolios, "SELECT * FROM Portfolio")

        if err == nil {
                c.JSON(200, portfolios)
        } else {
                c.JSON(404, gin.H{"error": "no portfolio(s) into the table"})
        }

        // curl -i http://localhost:8080/api/v1/portfolios
}

func GetPortfolioSet(c *gin.Context) {
        count := c.Params.ByName("count")
        offset := c.Params.ByName("offset")

        var portfolios []Portfolio
        _, err := dbmap.Select(&portfolios, "SELECT * FROM Portfolio ORDER BY createdtime DESC LIMIT ? OFFSET ?", count, offset)

        if err == nil {
                c.JSON(200, portfolios)
        } else {
                c.JSON(404, gin.H{"error": "no portfolio(s) into the table"})
        }
}

func GetBlogSet(c *gin.Context) {
        count := c.Params.ByName("count")
        offset := c.Params.ByName("offset")

        var blogs []Blog
        _, err := dbmap.Select(&blogs, "SELECT * FROM Blog ORDER BY createdtime DeSC LIMIT ? OFFSET ?", count, offset)

        if err == nil {
                c.JSON(200, blogs)
        } else {
                c.JSON(404, gin.H{"error": "no blog(s) into the table"})
        }
}

func GetPortfolio(c *gin.Context) {
        id := c.Params.ByName("id")
        var portfolio Portfolio
        err := dbmap.SelectOne(&portfolio, "SELECT * FROM Portfolio WHERE id=? LIMIT 1", id)

        if err == nil {
                portfolio_id, _ := strconv.ParseInt(id, 0, 64)

                content := &Portfolio{
                        Id:           portfolio_id,
                        Created_time: portfolio.Created_time,
                        Updated_time: portfolio.Updated_time,
                        Created_by:   portfolio.Created_by,
                        Updated_by:   portfolio.Updated_by,
                        Name:         portfolio.Name,
                        Description:  portfolio.Description,
                        Text_html:    portfolio.Text_html,
                        Demo_url:     portfolio.Demo_url,
                        Author:       portfolio.Author,
                }
                c.JSON(200, content)
        } else {
                fmt.Println(err)
                c.JSON(404, gin.H{"error": "portfolio not found"})
        }

        // curl -i http://localhost:8080/api/v1/portfolios/1
}

func PostPortfolio(c *gin.Context) {
        var portfolio Portfolio
        //x, _ := ioutil.ReadAll(c.Request.Body)
        //fmt.Printf("%s", string(x))

        c.Bind(&portfolio)

        log.Println(portfolio)

        if portfolio.Created_by == "" {
                portfolio.Created_by = "Anonymous"
        }

        if portfolio.Updated_by == "" {
                portfolio.Updated_by = portfolio.Created_by
        }

        if portfolio.Author == "" {
                portfolio.Author = portfolio.Created_by
        }
        log.Println(portfolio.Name)
        if portfolio.Created_by != "" && portfolio.Name != "" && portfolio.Text_html != "" && portfolio.Updated_by != "" {

                if insert, _ := dbmap.Exec(`INSERT INTO Portfolio (createdby, updatedby, name, description, texthtml, demourl, author) VALUES (?, ?, ?, ?, ?, ?, ?)`, portfolio.Created_by, portfolio.Updated_by, portfolio.Name, portfolio.Description, portfolio.Text_html, portfolio.Demo_url, portfolio.Author); insert != nil {
                        portfolio_id, err := insert.LastInsertId()
                        if err == nil {
                                content := &Portfolio{
                                        Id:           portfolio_id,
                                        Created_time: portfolio.Created_time,
                                        Updated_time: portfolio.Updated_time,
                                        Created_by:   portfolio.Created_by,
                                        Updated_by:   portfolio.Updated_by,
                                        Name:         portfolio.Name,
                                        Description:  portfolio.Description,
                                        Text_html:    portfolio.Text_html,
                                        Demo_url:     portfolio.Demo_url,
                                        Author:       portfolio.Author,
                                }
                                c.JSON(201, content)
                                return
                        } else {
                                checkErr(err, "Insert failed")
                        }
                }

        } else {
                c.JSON(400, gin.H{"error": "Fields are empty"})
                return
        }

        // curl -i -X POST -H "Content-Type: application/json" -d "{ \"firstname\": \"Thea\", \"lastname\": \"Queen\" }" http://localhost:8080/api/v1/portfolios
}

func UpdatePortfolio(c *gin.Context) {
        id := c.Params.ByName("id")
        var portfolio Portfolio
        err := dbmap.SelectOne(&portfolio, "SELECT * FROM Portfolio WHERE id=?", id)

        if err == nil {
                var json Portfolio
                c.Bind(&json)

                portfolio_id, _ := strconv.ParseInt(id, 0, 64)

                portfolio := Portfolio{
                        Id:           portfolio_id,
                        Created_time: json.Created_time,
                        Updated_time: json.Updated_time,
                        Created_by:   json.Created_by,
                        Updated_by:   json.Updated_by,
                        Name:         json.Name,
                        Description:  json.Description,
                        Text_html:    json.Text_html,
                        Demo_url:     json.Demo_url,
                        Author:       json.Author,
                }

                if portfolio.Created_by == "" {
                        portfolio.Created_by = "Anonymous"
                }

                if portfolio.Updated_by == "" {
                        portfolio.Updated_by = portfolio.Created_by
                }

                if portfolio.Author == "" {
                        portfolio.Author = portfolio.Created_by
                }

                if portfolio.Created_by != "" && portfolio.Updated_by != "" && portfolio.Name != "" && portfolio.Text_html != "" {
                        _, err = dbmap.Update(&portfolio)

                        if err == nil {
                                c.JSON(200, portfolio)
                        } else {
                                checkErr(err, "Updated failed")
                        }

                } else {
                        c.JSON(400, gin.H{"error": "fields are empty"})
                }

        } else {
                c.JSON(404, gin.H{"error": "portfolio not found"})
        }

        // curl -i -X PUT -H "Content-Type: application/json" -d "{ \"firstname\": \"Thea\", \"lastname\": \"Merlyn\" }" http://localhost:8080/api/v1/users/1
}

func DeletePortfolio(c *gin.Context) {
        id := c.Params.ByName("id")

        var portfolio Portfolio
        err := dbmap.SelectOne(&portfolio, "SELECT * FROM Portfolio WHERE id=?", id)

        if err == nil {
                _, err = dbmap.Delete(&portfolio)

                if err == nil {
                        c.JSON(200, gin.H{"id #" + id: "deleted"})
                } else {
                        checkErr(err, "Delete failed")
                }

        } else {
                c.JSON(404, gin.H{"error": "portfolio not found"})
        }

        // curl -i -X DELETE http://localhost:8080/api/v1/users/1
}

func OptionsPortfolio(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE,POST, PUT")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        c.Next()
}

func OptionsBlog(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE,POST, PUT")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        c.Next()
}

