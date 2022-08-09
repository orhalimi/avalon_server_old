package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	mongoUrl           = "localhost:27017"
	dbName             = "test_db"
	userCollectionName = "user"
)

var globalMutex sync.RWMutex





func wsPage(res http.ResponseWriter, req *http.Request) {
	jwtToken := req.URL.Query().Get("token")
	log.Println(jwtToken)

	claims, err := jwt.ParseWithClaims(jwtToken, &JWTData{}, func(token *jwt.Token) (interface{}, error) {
		if jwt.SigningMethodHS256 != token.Method {
			return nil, errors.New("Invalid signing algorithm")
		}
		return []byte(SECRET), nil
	})
	log.Println("err:", err)
	if err != nil {
		log.Println(err)
		http.Error(res, "Request failed!", http.StatusUnauthorized)
		return
	}
	log.Println("err")
	data := claims.Claims.(*JWTData)

	userName := data.CustomClaims["userName"]

	conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(res, req, nil)
	if error != nil {
		log.Println("err2")
		http.NotFound(res, req)
		return
	}

	//uuid,_:= uuid.NewV4()
	client := &Client{id: userName, socket: conn, send: make(chan []byte)}
	globalBoard.manager.register <- client

	log.Println("new socket. start client read and write threads")
	go client.read()
	go client.write()

}

type userRouter struct {
	userService *UserService1
}

func main() {
	fmt.Println("Starting server at http://localhost:12345...")
	f, _ := os.OpenFile("testlogfile.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	log.SetOutput(f)

	session, _ := NewSession(mongoUrl)
	defer func() {
		session.Close()
	}()

	hash := Hash{}
	userService := NewUserService(session.Copy(), dbName, userCollectionName, &hash)
	userRouter := userRouter{userService}

	go globalBoard.manager.start()
	router := mux.NewRouter()
	router.HandleFunc("/ws", wsPage).Methods("GET")

	router.HandleFunc("/register2", userRouter.createUserHandler).Methods("PUT", "OPTIONS", "POST")
	router.HandleFunc("/login", userRouter.login).Methods("POST", "OPTIONS")
	log.Fatal(http.ListenAndServe(":12345", cors.AllowAll().Handler(router)))
}

type user struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
}

type JWT struct {
	Token string `json:"token,omitempty"`
}

var dbUsers = map[string]user{}      // user ID, user
var dbSessions = map[string]string{} // session ID, user ID

func (ur *userRouter) createUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	user, err := decodeUser(r)

	if err != nil {
		log.Println("error", err)
		return
	}
	log.Println(user)
	err = ur.userService.Create(&user)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("good")
}

const (
	PORT   = "1337"
	SECRET = "42isTheAnswer"
)

func (ur *userRouter) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	var user struct {
		User     string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&user)
	dbUser, err := ur.userService.GetByUsername(user.User)
	if err != nil {

		log.Println(err)
		return
	}
	c := Hash{}
	log.Println("login start")
	compareError := c.Compare(dbUser.Password, user.Password)
	if compareError == nil {
		claims := JWTData{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour * 300).Unix(),
			},

			CustomClaims: map[string]string{
				"userName": dbUser.Username,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(SECRET))
		if err != nil {
			log.Println(err)
			http.Error(w, "Login failed!", http.StatusUnauthorized)
		}

		json, err := json.Marshal(struct {
			Token string `json:"token"`
			Name  string `json:"name"`
		}{
			tokenString,
			dbUser.Username,
		})

		if err != nil {
			log.Println(err)
			http.Error(w, "Login failed!", http.StatusUnauthorized)
		}

		w.Write(json)
	} else {
		http.Error(w, "Login failed!", http.StatusUnauthorized)
	}
	log.Println("login end")
}

func decodeUser(r *http.Request) (User, error) {
	var u User
	if r.Body == nil {
		return u, errors.New("no request body")
	}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&u)
	return u, err
}

type JWTData struct {
	// Standard claims are the standard jwt claims from the IETF standard
	// https://tools.ietf.org/html/rfc7519
	jwt.StandardClaims
	CustomClaims map[string]string `json:"custom,omitempty"`
}