package main

import (
	"context"
	"example/connection"
	"example/middleware"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// membangung beberapa objek/properties di dalam Blog
type Blog struct {
	Id          int
	Title       string
	StartDate   string
	EndDate     string
	Duration    string
	Description string
	JavaScript  bool
	Html        bool
	Php         bool
	React       bool
	Image       string
	Author      string
}

// membangun beberapa objek/properties di dalam User
type User struct {
	Id       int
	Name     string
	Email    string
	Password string
}

// membangun beberapa objek/properties di dalam SessionData
type SessionData struct {
	IsLogin bool
	Name    string
}

var userData = SessionData{}

func main() {
	// konekni Database
	connection.DatabaseConnect()

	e := echo.New()

	// memnggunakan session dr echo/labstack
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("session"))))

	e.Static("/public", "public")
	e.Static("/uploads", "uploads")

	// routing
	// GET mendapatkan data
	e.GET("/", home)
	e.GET("/addproject", addProject)
	e.GET("/contact", contact)
	e.GET("detailproject/:id", detailProject)
	e.GET("updateproject/:id", updateProject)
	e.GET("form-register", formRegister)
	e.GET("form-login", formLogin)

	// POST mengirim data
	e.POST("/addblog", middleware.UploadFiles(addBlog))
	e.POST("/deleteblog/:id", deleteBlog)
	e.POST("/updateproject/:id", middleware.UploadFiles(updateProjectDone))
	e.POST("/login", login)
	e.POST("/register", register)
	e.POST("/logout", logout)

	e.Logger.Fatal(e.Start("localhost:5000"))
}

// menampilkan form-register
func formRegister(c echo.Context) error {
	var tmpl, err = template.ParseFiles("views/form-register.html")

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return tmpl.Execute(c.Response(), nil)
}

// mengePOST data register
func register(c echo.Context) error {
	// untuk memastikan isi permintaan adalah format data, bukan JSON, XML, dll
	err := c.Request().ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	name := c.FormValue("inputName")
	email := c.FormValue("inputEmail")
	password := c.FormValue("inputPassword")

	// membuat password menjadi acak
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	// point2  yang akan diisi
	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO tb_user(name, email, password) VALUES ($1, $2, $3)", name, email, passwordHash)

	if err != nil {
		redirectWithMessage(c, "Register failed, please try again.", false, "/form-register")
	}

	return redirectWithMessage(c, "Register success", true, "/form-login")
}

// menampilkan form-login
func formLogin(c echo.Context) error {
	sess, _ := session.Get("session", c)

	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["status"],
		"FlashMessage": sess.Values["message"],
	}

	delete(sess.Values, "message")
	delete(sess.Values, "status")
	sess.Save(c.Request(), c.Response())

	var tmpl, err = template.ParseFiles("views/form-login.html")

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return tmpl.Execute(c.Response(), flash)
}

// mengPOST data login
func login(c echo.Context) error {
	// untuk memastikan isi permintaan adalah format data, bukan JSON, XML, dll
	err := c.Request().ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	email := c.FormValue("inputEmail")
	password := c.FormValue("inputPassword")

	user := User{}
	// Scan membaca data
	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_user WHERE email=$1", email).Scan(&user.Id, &user.Name, &user.Email, &user.Password)
	if err != nil {
		return redirectWithMessage(c, "Email Incorrect!", false, "/form-login")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return redirectWithMessage(c, "Password Incorrect", false, "/form-login")
	}

	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = 10800 // 3 Jam
	sess.Values["message"] = "Login Succes!"
	sess.Values["status"] = true
	sess.Values["name"] = user.Name
	sess.Values["email"] = user.Email
	sess.Values["id"] = user.Id
	sess.Values["isLogin"] = true
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusMovedPermanently, "/")
}

// mengePOST data logout
func logout(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func redirectWithMessage(c echo.Context, message string, status bool, path string) error {
	sess, _ := session.Get("session", c)
	sess.Values["message"] = message
	sess.Values["status"] = status
	sess.Save(c.Request(), c.Response())
	return c.Redirect(http.StatusMovedPermanently, path)
}

func home(c echo.Context) error {
	sess, _ := session.Get("session", c)
	var result []Blog

	if sess.Values["isLogin"] != true {
		userData.IsLogin = false
		data, _ := connection.Conn.Query(context.Background(), "SELECT tb_project.id, title, start_date, end_date, duration, description, javascript, html, php, react, image, tb_user.name AS author FROM tb_project JOIN tb_user ON tb_project.author_id = tb_user.id ORDER BY tb_project.id DESC")
		for data.Next() {
			var each = Blog{}

			err := data.Scan(&each.Id, &each.Title, &each.StartDate, &each.EndDate, &each.Duration, &each.Description, &each.JavaScript, &each.Html, &each.Php, &each.React, &each.Image, &each.Author)
			if err != nil {
				fmt.Println(err.Error())
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
			}
			result = append(result, each)
		}
	} else {
		userData.IsLogin = sess.Values["isLogin"].(bool)
		userData.Name = sess.Values["name"].(string)
		id := sess.Values["id"].(int)
		data, _ := connection.Conn.Query(context.Background(), "SELECT tb_project.id, title, start_date, end_date, duration, description, javascript, html, php, react, image, tb_user.name AS author FROM tb_project JOIN tb_user ON tb_project.author_id = tb_user.id WHERE tb_user.id=$1 ORDER BY tb_project.id DESC", id)
		for data.Next() {
			var each = Blog{}

			err := data.Scan(&each.Id, &each.Title, &each.StartDate, &each.EndDate, &each.Duration, &each.Description, &each.JavaScript, &each.Html, &each.Php, &each.React, &each.Image, &each.Author)
			if err != nil {
				fmt.Println(err.Error())
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
			}
			result = append(result, each)
		}
	}

	blogs := map[string]interface{}{
		"Blogs":       result,
		"DataSession": userData,
	}

	delete(sess.Values, "message")
	delete(sess.Values, "status")
	sess.Save(c.Request(), c.Response())

	var tmpl, err = template.ParseFiles("views/index.html")

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return tmpl.Execute(c.Response(), blogs)
}

func addProject(c echo.Context) error {
	var tmpl, err = template.ParseFiles("views/addproject.html")

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	sess, _ := session.Get("session", c)

	if sess.Values["isLogin"] != true {
		userData.IsLogin = false
	} else {
		userData.IsLogin = sess.Values["isLogin"].(bool)
		userData.Name = sess.Values["name"].(string)
	}

	blogs := map[string]interface{}{
		"DataSession": userData,
	}

	sess.Save(c.Request(), c.Response())

	return tmpl.Execute(c.Response(), blogs)
}

func contact(c echo.Context) error {
	var tmpl, err = template.ParseFiles("views/contact.html")

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	sess, _ := session.Get("session", c)

	if sess.Values["isLogin"] != true {
		userData.IsLogin = false
	} else {
		userData.IsLogin = sess.Values["isLogin"].(bool)
		userData.Name = sess.Values["name"].(string)
	}

	blogs := map[string]interface{}{
		"DataSession": userData,
	}

	sess.Save(c.Request(), c.Response())

	return tmpl.Execute(c.Response(), blogs)
}

func detailProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	var detailProject = Blog{}

	err := connection.Conn.QueryRow(context.Background(), "SELECT tb_project.id, title, start_date, end_date, duration, description, javascript, html, php, react, image, tb_user.name as author FROM tb_project JOIN tb_user ON tb_project.author_id = tb_user.id WHERE tb_project.id=$1", id).Scan(
		&detailProject.Id, &detailProject.Title, &detailProject.StartDate, &detailProject.EndDate, &detailProject.Duration, &detailProject.Description, &detailProject.JavaScript, &detailProject.Html, &detailProject.Php, &detailProject.React, &detailProject.Image, &detailProject.Author)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	data := map[string]interface{}{
		"Blog": detailProject,
	}

	var tmpl, errTemplate = template.ParseFiles("views/detailproject.html")

	if errTemplate != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return tmpl.Execute(c.Response(), data)
}

func addBlog(c echo.Context) error {
	title := c.FormValue("project-name")
	startDate := c.FormValue("start-date")
	endDate := c.FormValue("end-date")
	duration := hitungDuration(startDate, endDate)
	description := c.FormValue("description")
	var javascript bool
	if c.FormValue("javascript") == "javascript" {
		javascript = true
	}
	var html bool
	if c.FormValue("html") == "html" {
		html = true
	}
	var php bool
	if c.FormValue("php") == "php" {
		php = true
	}
	var react bool
	if c.FormValue("react") == "react" {
		react = true
	}
	image := c.Get("dataFile").(string)

	sess, _ := session.Get("session", c)

	author := sess.Values["id"].(int)

	_, err := connection.Conn.Exec(context.Background(), "INSERT INTO tb_project (title, start_date, end_date, duration, description, javascript, html, php, react, image, author_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)", title, startDate, endDate, duration, description, javascript, html, php, react, image, author)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func deleteBlog(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	fmt.Println("index : ", id)

	_, err := connection.Conn.Exec(context.Background(), "DELETE FROM tb_project WHERE id=$1", id)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func updateProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	var detailProject = Blog{}

	err := connection.Conn.QueryRow(context.Background(), "SELECT id, title, description, image, start_date, end_date, javascript, html, php, react, duration FROM tb_project WHERE id=$1", id).Scan(&detailProject.Id, &detailProject.Title, &detailProject.Description, &detailProject.Image, &detailProject.StartDate, &detailProject.EndDate, &detailProject.JavaScript, &detailProject.Html, &detailProject.Php, &detailProject.React, &detailProject.Duration)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	data := map[string]interface{}{
		"Blog": detailProject,
	}

	var tmpl, errTemplate = template.ParseFiles("views/updateproject.html")

	if errTemplate != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return tmpl.Execute(c.Response(), data)
}

func updateProjectDone(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	title := c.FormValue("project-name")
	startDate := c.FormValue("start-date")
	endDate := c.FormValue("end-date")
	duration := hitungDuration(startDate, endDate)
	description := c.FormValue("description")
	var javascript bool
	if c.FormValue("javascript") == "javascript" {
		javascript = true
	}
	var html bool
	if c.FormValue("html") == "html" {
		html = true
	}
	var php bool
	if c.FormValue("php") == "php" {
		php = true
	}
	var react bool
	if c.FormValue("react") == "react" {
		react = true
	}

	image := c.Get("dataFile").(string)

	_, err := connection.Conn.Exec(context.Background(), "UPDATE tb_project SET title=$1, description=$2, start_date=$3, end_date=$4, javascript=$5, html=$6, php=$7, react=$8, duration=$9, image=$10 WHERE id=$11", title, description, startDate, endDate, javascript, html, php, react, duration, image, id)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func hitungDuration(StartDate, EndDate string) string {
	startTime, _ := time.Parse("2006-01-02", StartDate)
	endTime, _ := time.Parse("2006-01-02", EndDate)

	durationTime := int(endTime.Sub(startTime).Hours())
	durationDays := durationTime / 24
	durationWeeks := durationDays / 7
	durationMonths := durationWeeks / 4
	durationYears := durationMonths / 12

	var duration string

	if durationYears > 1 {
		duration = strconv.Itoa(durationYears) + " Tahun"
	} else if durationYears > 0 {
		duration = strconv.Itoa(durationYears) + " Tahun"
	} else {
		if durationMonths > 1 {
			duration = strconv.Itoa(durationMonths) + " Bulan"
		} else if durationMonths > 0 {
			duration = strconv.Itoa(durationMonths) + " Bulan"
		} else {
			if durationWeeks > 1 {
				duration = strconv.Itoa(durationWeeks) + " Minggu"
			} else if durationWeeks > 0 {
				duration = strconv.Itoa(durationWeeks) + " Minggu"
			} else {
				if durationDays > 1 {
					duration = strconv.Itoa(durationDays) + " Hari"
				} else {
					duration = strconv.Itoa(durationDays) + " Hari"
				}
			}
		}
	}

	return duration
}
