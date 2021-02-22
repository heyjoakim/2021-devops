package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/heyjoakim/devops-21/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"golang.org/x/crypto/bcrypt"
)

// PageData defines data on page whatever and request
type PageData map[string]interface{}
type layoutPage struct {
	Layout string
}
type Result struct {
	Text     string
	PubDate  int64
	Email    string
	Username string
}

// App defines the application
type App struct {
	db *gorm.DB
}

// configuration
var (
	database  = "./tmp/minitwit.db"
	perPage   = 30
	debug     = true
	secretKey = []byte("development key")
	store     = sessions.NewCookieStore(secretKey)
	latest    = 0
)

var staticPath string = "/static"
var cssPath string = "/css"
var timelinePath string = "./templates/timeline.html"
var layoutPath string = "./templates/layout.html"
var loginPath string = "./templates/login.html"
var registerPath string = "./templates/register.html"

func (d *App) connectDb() (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(database), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
}

// initDb creates the database tables.
func (d *App) initDb() {
	var user models.User
	var follower models.Follower
	var message models.Message

	d.db.AutoMigrate(&user, &follower, &message)
}

// getUserID returns user ID for username
func (d *App) getUserID(username string) (uint, error) {
	var user models.User
	err := d.db.First(&user, "username = ?", username).Error
	if err != nil {
		fmt.Println(err)
	}
	return user.UserID, err
}

// formatDatetime formats a timestamp for display.
func (d *App) formatDatetime(timestamp int64) string {
	timeObject := time.Unix(timestamp, 0)
	return timeObject.Format("2006-02-01 @ 02:04")
}

// gravatarURL return the gravatar image for the given email address.
func (d *App) gravatarURL(email string, size int) string {
	encodedEmail := hex.EncodeToString([]byte(strings.ToLower(strings.TrimSpace(email))))
	hashedEmail := fmt.Sprintf("%x", sha256.Sum256([]byte(encodedEmail)))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?d=identicon&s=%d", hashedEmail, size)
}

func (d *App) notReqFromSimulator(r *http.Request) []byte {
	// authHeader := r.Header.Get("Authorization")
	// if authHeader != "Basic c2ltdWxhdG9yOnN1cGVyX3NhZmUh" {
	// 	data := map[string]interface{}{
	// 		"status":    http.StatusForbidden,
	// 		"error_msg": "You are not authorized to use this resource!",
	// 	}

	// 	json, _ := d.serialize(data)
	// 	return json
	// }
	return nil
}

func getQueryParameter(r *http.Request, name string) string {
	queryValue := r.URL.Query().Get(name)
	return queryValue
}

func (d *App) getUser(userID uint) models.User {
	var user models.User
	err := d.db.First(&user, "user_id = ?", userID).Error
	if err != nil {
		fmt.Println(err)

	}
	return user
}

func (d *App) serialize(input interface{}) ([]byte, error) {
	js, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	return js, nil
}

// beforeRequest make sure we are connected to the database each request and look
// up the current user so that we know he's there.
func (d *App) beforeRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// database, _ := d.connectDb()
		// d.db = database
		session, _ := store.Get(r, "_cookie")
		userID := session.Values["user_id"]
		if userID != nil {
			id := userID.(uint)
			tmpUser := d.getUser(id)
			session.Values["user_id"] = tmpUser.UserID
			session.Values["username"] = tmpUser.Username
			session.Save(r, w)
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// Closes the database again at the end of the request.
func (d *App) afterRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Entered: " + r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// timelineHandler a users timeline or if no user is logged in it will
// redirect to the public timeline.  This timeline shows the user's
// messages as well as all the messages of followed users.
func (d *App) timelineHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "_cookie")
	fmt.Println(session)
	if session.Values["user_id"] != nil {
		routeName := fmt.Sprintf("/%s", session.Values["username"])
		http.Redirect(w, r, routeName, http.StatusFound)
		return
	}

	http.Redirect(w, r, "/public", http.StatusFound)
}

func (d *App) publicTimelineHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(timelinePath, layoutPath)
	if err != nil {
		log.Fatal(err)
	}

	var results []Result
	d.db.Model(&models.Message{}).Select("message.text, message.pub_date, user.email, user.username").Joins("left join user on user.user_id = message.author_id").Where("message.flagged=0").Order("pub_date desc").Limit(perPage).Scan(&results)

	var messages []models.MessageViewModel
	for _, result := range results {
		message := models.MessageViewModel{
			Email:   d.gravatarURL(result.Email, 48),
			User:    result.Username,
			Content: result.Text,
			PubDate: d.formatDatetime(result.PubDate),
		}
		messages = append(messages, message)
	}

	session, err := store.Get(r, "_cookie")
	username := session.Values["username"]

	data := PageData{
		"username": username,
		"messages": messages,
		"msgCount": len(messages),
	}

	tmpl.Execute(w, data)
}

func (d *App) userTimelineHandler(w http.ResponseWriter, r *http.Request) {
	//Display's a users tweets.
	params := mux.Vars(r)
	profileUsername := params["username"]

	profileUserID, err := d.getUserID(profileUsername)
	if err != nil {
		w.WriteHeader(404)
		fmt.Println(err)
		return
	}

	session, err := store.Get(r, "_cookie")
	sessionUserID := session.Values["user_id"].(uint)
	data := PageData{"followed": false}

	if sessionUserID != 0 {
		var follower models.Follower
		d.db.Where("who_id = ?", sessionUserID).Where("whom_id = ?", profileUserID).Find(&follower)
		if follower.WhoID != 0 {
			data["followed"] = true
		}
	}

	tmpl, err := template.ParseFiles(timelinePath, layoutPath)
	if err != nil {
		log.Fatal(err)
	}

	messages := d.getPostsForUser(profileUserID)

	var msgS []models.MessageViewModel
	if sessionUserID != 0 {
		if sessionUserID == profileUserID {
			followlist := d.getFollowedUsers(sessionUserID)
			for _, v := range followlist {
				msgS = append(d.getPostsForUser(v), msgS...)
			}
		}
	}

	messages = append(msgS, messages...)
	data["messages"] = messages
	data["title"] = fmt.Sprintf("%s's Timeline", profileUsername)
	data["profileOwner"] = profileUsername

	if session.Values["username"] == profileUsername {
		data["ownProfile"] = true
	} else {
		currentUser := session.Values["user_id"]
		otherUser, err := d.getUserID(profileUsername)
		if err != nil {
			http.Error(w, "User does not exist", 400)
			return
		}

		var follower models.Follower
		d.db.Where("who_id = ?", otherUser).Where("whom_id = ?", currentUser).First(&follower)
		if follower.WhoID != 0 && follower.WhomID != 0 {
			data["followed"] = true
		}
	}

	data["msgCount"] = len(messages)
	data["username"] = session.Values["username"]
	data["MsgInfo"] = session.Flashes("Info")
	data["MsgWarn"] = session.Flashes("Warn")
	session.Save(r, w)

	tmpl.Execute(w, data)
}

func (d *App) getPostsForUser(id uint) []models.MessageViewModel {
	var results []Result
	d.db.Model(models.Message{}).Order("pub_date desc").Limit(perPage).Select("message.text,message.pub_date, user.email, user.username").Joins("left join user on user.user_id = message.author_id").Where("user.user_id=?", id).Scan(&results)

	var messages []models.MessageViewModel
	for _, result := range results {
		message := models.MessageViewModel{
			Email:   d.gravatarURL(result.Email, 48),
			User:    result.Username,
			Content: result.Text,
			PubDate: d.formatDatetime(result.PubDate),
		}
		messages = append(messages, message)
	}

	return messages
}

// get ID's of all users that are followed by some user
func (d *App) getFollowedUsers(id uint) []uint {
	var followers []models.Follower
	d.db.Where("who_id = ?", id).Find(&followers)

	var followlist []uint
	for _, follower := range followers {
		followlist = append(followlist, follower.WhomID)
	}

	return followlist
}

// follow user
func (d *App) followUserHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "_cookie")
	currentUserID := session.Values["user_id"].(uint)
	params := mux.Vars(r)
	username := params["username"]
	userToFollowID, _ := d.getUserID(username)

	follower := models.Follower{WhoID: currentUserID, WhomID: userToFollowID}
	result := d.db.Create(&follower)
	if result.Error != nil {
		log.Fatal(result.Error)
		fmt.Println("database error: ", result.Error)
		return
	}

	session.AddFlash("You are now following "+username, "Info")
	session.Save(r, w)
	http.Redirect(w, r, "/"+username, http.StatusFound)
}

// Unfollow user - relies on a query string
func (d *App) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "_cookie")
	loggedInUser := session.Values["user_id"].(uint)
	params := mux.Vars(r)
	username := params["username"]

	if username == "" {
		session.AddFlash("No query parameter present", "Warn")
		session.Save(r, w)
		http.Redirect(w, r, "timeline", http.StatusFound)
		return
	}

	id2, user2Err := d.getUserID(username)
	if user2Err != nil {
		session.AddFlash("User does not exist", "Warn")
		session.Save(r, w)
		http.Redirect(w, r, "timeline", http.StatusFound)
		return
	}

	var follower models.Follower
	err := d.db.Where("who_id = ?", loggedInUser).Where("whom_id = ?", id2).Delete(&follower).Error

	if err != nil {
		session.AddFlash("Error following user", "Warn")
		session.Save(r, w)
		fmt.Println("db error: ", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	session.AddFlash("You are no longer following "+username, "Info")
	session.Save(r, w)
	http.Redirect(w, r, "/"+username, http.StatusFound)
	return
}

func (d *App) addMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		userID, _ := d.getUserID(r.FormValue("token"))
		message := models.Message{AuthorID: userID, Text: r.FormValue("text"), PubDate: time.Now().Unix(), Flagged: 0}
		err := d.db.Create(&message).Error
		if err != nil {
			log.Fatal(err)
			fmt.Println("database error: ", err)
			return
		}

		http.Redirect(w, r, "/"+r.FormValue("token"), http.StatusFound)
	}
}

func (d *App) loginHandler(w http.ResponseWriter, r *http.Request) {

	session, err := store.Get(r, "_cookie")
	if ok := session.Values["user_id"] != nil; ok {
		routeName := fmt.Sprintf("/%s", session.Values["username"])
		http.Redirect(w, r, routeName, http.StatusFound)
	}

	var loginError string
	if r.Method == "POST" {
		var user models.User
		err := d.db.Where("username = ?", r.FormValue("username")).First(&user).Error
		if err != nil {
			loginError = "User does not exist"
			fmt.Println(err)
		}

		if r.FormValue("username") != user.Username {
			loginError = "Invalid username"
		} else if err := bcrypt.CompareHashAndPassword([]byte(user.PwHash), []byte(r.FormValue("password"))); err != nil {
			loginError = "Invalid password"
		} else {
			session.AddFlash("You were logged in")
			session.Values["user_id"] = user.UserID
			session.Save(r, w)

			http.Redirect(w, r, "/"+user.Username, http.StatusFound)
		}
	}

	tmpl, err := template.ParseFiles(loginPath, layoutPath)
	if err != nil {
		log.Fatal(err)
		return
	}
	data := PageData{
		"error":    loginError,
		"username": session.Values["username"],
	}
	tmpl.Execute(w, data)

}

func (d *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "_cookie")
	if ok := session.Values["user_id"] != nil; ok {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	var registerError string
	if r.Method == "POST" {
		if len(r.FormValue("username")) == 0 {
			registerError = "You have to enter a username"
		} else if len(r.FormValue("email")) == 0 || strings.
			Contains(r.FormValue("email"), "@") == false {
			registerError = "You have to enter a valid email address"
		} else if len(r.FormValue("password")) == 0 {
			registerError = "You have to enter a password"
		} else if r.FormValue("password") != r.FormValue("password2") {
			registerError = "The two passwords do not match"
		} else if _, err := d.getUserID(r.FormValue("username")); err == nil {
			registerError = "The username is already taken"
		} else {

			hash, err := bcrypt.
				GenerateFromPassword([]byte(r.FormValue("password")), bcrypt.DefaultCost)
			if err != nil {
				log.Fatal(err)
				return
			}
			username := r.FormValue("username")
			email := r.FormValue("email")
			user := models.User{Username: username, Email: email, PwHash: string(hash)}
			error := d.db.Create(&user).Error

			if error != nil {
				fmt.Println(error)
			}

			session.AddFlash("You are now registered ?", username)
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}

	t, err := template.ParseFiles(registerPath, layoutPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		"error": registerError,
	}

	t.Execute(w, data)
}

func (d *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "_cookie")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	session.Values["user_id"] = ""
	session.Values["username"] = ""
	session.Options.MaxAge = -1

	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (d *App) faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/dev.png")
	// http.Redirect(w, r, "/public", http.StatusFound)
}

func updateLatest(r *http.Request) {
	tryLatestQuery := r.URL.Query().Get("latest")

	if tryLatestQuery == "" {
		latest = -1
	} else {
		tryLatest, _ := strconv.Atoi(tryLatestQuery)
		latest = tryLatest
	}
}

// GetLatest godoc
// @Summary Get the latest x
// @Description Get the latest x
// @Produce  json
// @Success 200 {object} interface{}
// @Router /latest [get]
func (d *App) GetLatestHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{ // could also be an array
		"latest": latest,
	}

	json, err := d.serialize(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

// Messages godoc
// @Summary Registers a user
// @Description Registers a user, provided that the given info passes all checks.
// @Accept json
// @Produce json
// @Success 203
// @Failure 400 {string} string "unauthorized"
// @Router /register [post]
func (d *App) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var registerRequest models.RegisterRequest
	json.NewDecoder(r.Body).Decode(&registerRequest)

	updateLatest(r)
	var registerError string
	if r.Method == "POST" {
		if len(registerRequest.Username) == 0 {
			registerError = "You have to enter a username"
		} else if len(registerRequest.Email) == 0 || strings.Contains(registerRequest.Email, "@") == false {
			registerError = "You have to enter a valid email address"
		} else if len(registerRequest.Password) == 0 {
			registerError = "You have to enter a password"
		} else if _, err := d.getUserID(registerRequest.Username); err == nil {
			registerError = "The username is already taken"
		} else {
			hash, err := bcrypt.GenerateFromPassword([]byte(registerRequest.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Print(err)
			}

			user := models.User{Username: registerRequest.Username, Email: registerRequest.Email, PwHash: string(hash)}
			err = d.db.Create(&user).Error

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if registerError != "" {
			error := map[string]interface{}{
				"status":    400,
				"error_msg": registerError,
			}
			json, _ := d.serialize(error)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(json)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)

	}
}

// Messages godoc
// @Summary Gets the latest messages
// @Description Gets the latest messages in descending order.
// @Param no query int false "Number of results returned"
// @Produce  json
// @Success 200 {object} interface{}
// @Failure 401 {string} string "unauthorized"
// @Router /msgs [get]
func (d *App) MessagesHandler(w http.ResponseWriter, r *http.Request) {
	updateLatest(r)

	notFromSimResponse := d.notReqFromSimulator(r)
	if notFromSimResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(notFromSimResponse)
		return
	}

	var noMsgs int
	noMsgsQuery := getQueryParameter(r, "no")
	if noMsgsQuery == "" {
		noMsgs = 100
	} else {
		noMsgs, _ = strconv.Atoi(noMsgsQuery)
	}

	if r.Method == "GET" {
		var results []models.MessageResponse
		d.db.Model(&models.Message{}).Select("message.text, message.pub_date, user.email, user.username").Joins("left join user on user.user_id = message.author_id").Where("message.flagged=0").Order("pub_date desc").Limit(noMsgs).Scan(&results)

		// var messages []models.MessageResult
		// for _, result := range results {
		// 	message := models.MessageResponse{
		// 		Content: result.Text,
		// 		PubDate: result.PubDate,
		// 		User:    result.Username,
		// 	}
		// 	messages = append(messages, message)
		// }

		json, _ := d.serialize(results)

		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}

// MessagesPerUser godoc
// @Summary Gets the latest messages per user
// @Description Gets the latest messages per user
// @Param no query int false "Number of results returned"
// @Param latest query int false "Something about latest"
// @Produce  json
// @Success 200 {object} interface{}
// @Failure 401 {string} string "unauthorized"
// @Failure 500 {string} string response.Error
// @Router /msgs/{username} [get]
// @Router /msgs/{username} [post]
func (d *App) MessagesPerUserHandler(w http.ResponseWriter, r *http.Request) {
	updateLatest(r)
	params := mux.Vars(r)
	username := params["username"]

	userID, _ := d.getUserID(username)

	notFromSimResponse := d.notReqFromSimulator(r)
	if notFromSimResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(notFromSimResponse)
		return
	}

	var noMsgs int
	noMsgsQuery := getQueryParameter(r, "no")
	if noMsgsQuery == "" {
		noMsgs = 100
	} else {
		noMsgs, _ = strconv.Atoi(noMsgsQuery)
	}

	if r.Method == "GET" {
		var results []models.MessageResponse
		d.db.Model(models.Message{}).Order("pub_date desc").Select("message.text,message.pub_date, user.email, user.username").Joins("left join user on user.user_id = message.author_id").Where("user.user_id=?", userID).Limit(noMsgs).Scan(&results)

		var messages []models.MessageResponse
		for _, result := range results {
			message := models.MessageResponse{
				User:    result.User,
				Content: result.Content,
				PubDate: result.PubDate,
			}
			messages = append(messages, message)
		}

		json, _ := d.serialize(messages)

		w.Header().Set("Content-Type", "application/json")
		w.Write(json)

	} else if r.Method == "POST" {
		var messageRequest models.MessageRequest
		err := json.NewDecoder(r.Body).Decode(&messageRequest)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		message := models.Message{AuthorID: userID, Text: messageRequest.Content, PubDate: time.Now().Unix(), Flagged: 0}
		err = d.db.Create(&message).Error

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}

}

// FollowHandler godoc
// @Summary Follow, unfollow or get followers
// @Description Eiter follows a user, unfollows a user or returns a list of users's followers
// @Param no query int false "Number of results returned"
// @Param latest query int false "Something about latest"
// @Accept  json
// @Produce json
// @Success 200 {object} interface{}
// @Success 204 {object} interface{}
// @Failure 401 {string} string "unauthorized"
// @Failure 500 {string} string response.Error
// @Router /fllws/{username} [get]
// @Router /fllws/{username} [post]
func (d *App) FollowHandler(w http.ResponseWriter, r *http.Request) {
	updateLatest(r)

	notFromSimResponse := d.notReqFromSimulator(r)
	if notFromSimResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(notFromSimResponse)
		return
	}

	username := mux.Vars(r)["username"]
	userID, err := d.getUserID(username)
	if err != nil {
		http.Error(w, fmt.Sprintf("User not found: %s", username), http.StatusNotFound)
		return
	}

	var followRequest models.FollowRequest
	json.NewDecoder(r.Body).Decode(&followRequest)
	if followRequest.Follow != "" && followRequest.Unfollow != "" {
		http.Error(w, "Invalid input. Can ONLY handle either follow OR unfollow.", http.StatusUnprocessableEntity)
	} else if r.Method == "POST" && followRequest.Follow != "" {
		followsUserID, err := d.getUserID(followRequest.Follow)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		follower := models.Follower{WhoID: userID, WhomID: followsUserID}
		err = d.db.Create(&follower).Error
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusNoContent)
		return
	} else if r.Method == "POST" && followRequest.Unfollow != "" {
		unfollowsUserID, err := d.getUserID(followRequest.Unfollow)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			log.Fatal(err)
		}

		var follower models.Follower
		err = d.db.Where("who_id = ?", userID).Where("whom_id = ?", unfollowsUserID).Delete(&follower).Error

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal()
		}

		w.WriteHeader(http.StatusNoContent)
		return

	} else if r.Method == "GET" {
		var noFollowers int
		noFollowersQuery := getQueryParameter(r, "no")
		if noFollowersQuery == "" {
			noFollowers = 100
		} else {
			noFollowers, _ = strconv.Atoi(noFollowersQuery)
		}

		res, _ := db.Query("SELECT user.username FROM user "+
			"INNER JOIN follower ON follower.whom_id=user.user_id "+
			"WHERE follower.who_id=? "+
			"LIMIT ?", userID, noFollowers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		followers := make([]string, 0)
		for res.Next() {
			var username string
			err := res.Scan(&username)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			followers = append(followers, username)
		}

		json, _ := d.serialize(map[string]interface{}{"follows": followers})
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}

func (d *App) init() {
	fmt.Println("init")
	db, err := d.connectDb()
	if err != nil {
		log.Panic(err)
	}
	d.db = db
}

func main() {
	router := mux.NewRouter()
	var app App
	app.init()

	s := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	router.PathPrefix("/static/").Handler(s)

	router.Use(app.beforeRequest)
	router.Use(app.afterRequest)
	router.HandleFunc("/", app.timelineHandler)
	router.HandleFunc("/{username}/unfollow", app.unfollowUserHandler)
	router.HandleFunc("/{username}/follow", app.followUserHandler)
	router.HandleFunc("/login", app.loginHandler).Methods("GET", "POST")
	router.HandleFunc("/logout", app.logoutHandler)
	router.HandleFunc("/addMessage", app.addMessageHandler).Methods("GET", "POST")
	router.HandleFunc("/register", app.registerHandler).Methods("GET", "POST")
	router.HandleFunc("/public", app.publicTimelineHandler)
	router.HandleFunc("/favicon.ico", app.faviconHandler)
	router.HandleFunc("/{username}", app.userTimelineHandler)

	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/latest", app.GetLatestHandler)
	apiRouter.HandleFunc("/register", app.RegisterHandler).Methods("POST")
	apiRouter.HandleFunc("/msgs", app.MessagesHandler)
	apiRouter.HandleFunc("/msgs/{username}", app.MessagesPerUserHandler).Methods("GET", "POST")
	apiRouter.HandleFunc("/fllws/{username}", app.FollowHandler).Methods("GET", "POST")

	fmt.Println("Server running on port http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", router))
}
