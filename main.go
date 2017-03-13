package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	"bitbucket.org/atlassianlabs/hipchat-golang-base/util"

	"github.com/gorilla/mux"
	"github.com/tbruyelle/hipchat-go/hipchat"
)

/*
Room 814008: Dev
Room 869823: Chat - Hangman Discussion Group
Room 875860: Screenwriter Team
Room 890277: Tech - SQLAlchemy
Room 890278: Tech - Postgres
Room 890313: Tech - Frontend
Room 1107815: Chat - DnD
Room 1138222: Product - Producer
Room 1141102: Chat - Cinema
Room 1164344: Product - AdFuser
Room 1268848: Tech - Docker
Room 1412918: Tech - General
Room 1632457: Chat - Music
Room 1807463: The Grand Emulator Project
Room 2069593: Tech- Virtual Machines
Room 2117980: Tech - iOS/Swift Discussion
Room 2242718: Chat - PS4 Gaming (Destiny, Diablo III etc.)
Room 2772221: Chat - Game of Thrones
Room 2779747: Engineering
Room 2943193: Chat - Board Games
Room 2957788: ThunderstormAlerts
Room 2974586: Parking Spot
Room 2999610: Product- Lifeguard Ticketing
Room 3019670: Chat - Roshambo
Room 3202140: Product- Auditor & Keymaster
Room 3282519: IS
Room 3381674: Sentry
Room 3423527: Wanda VPN
Room 3612574: test sentry
Room 3616654: mm
Room 3619724: I Ran So Far Away
Room 3667710: Gochatbot
Room 876627: Chat - Gif room
Room 879090: willbot test room
Room 1178087: Tech - Jenkins' pantry
Room 1459469: Scrum(ptious) coaching arena
Room 1854475: Blog Room
Room 2401331: adfuser version number discussion
*/
const DEVROOM = 814008
const SCREENWRITER_ROOM = 875860
const GOCHATROOM = 3667710

var (
	token string
	maxResults      = flag.Int("maxResults", 5, "Max results per request")
	includePrivate  = flag.Bool("includePrivate", false, "Include private rooms?")
	includeArchived = flag.Bool("includeArchived", false, "Include archived rooms?")
	// roomList = []int{3667710, 3619724, 3019670, 2999610, 2974586, 2957788, 2779747, 1138222, 814008,875860}
	//roomId = "875860"
)

type RoomConfig struct {
    token *hipchat.OAuthAccessToken
    hc    *hipchat.Client
    name  string
}

type Context struct {
    baseURL string
    static  string
    rooms   map[string]*RoomConfig
}

func (c *Context) healthcheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode([]string{"OK"})
}

func (c *Context) atlassianConnect(w http.ResponseWriter, r *http.Request) {
	lp := path.Join("./static", "atlassian-connect.json")
	vals := map[string]string{
		"LocalBaseUrl": c.baseURL,
	}
	tmpl, err := template.ParseFiles(lp)
	if err != nil {
		log.Fatalf("%v", err)
	}
	tmpl.ExecuteTemplate(w, "config", vals)
}

func (c *Context) installable(w http.ResponseWriter, r *http.Request) {
	authPayload, err := util.DecodePostJSON(r, true)
	if err != nil {
		log.Fatalf("Parsed auth data failed:%v\n", err)
	}

	credentials := hipchat.ClientCredentials{
		ClientID:     authPayload["oauthId"].(string),
		ClientSecret: authPayload["oauthSecret"].(string),
	}
	roomName := strconv.Itoa(int(authPayload["roomId"].(float64)))
	newClient := hipchat.NewClient("")
	tok, _, err := newClient.GenerateToken(credentials, []string{hipchat.ScopeSendNotification})
	if err != nil {
		log.Fatalf("Client.GetAccessToken returns an error %v", err)
	}
	rc := &RoomConfig{
		name: roomName,
		hc:   tok.CreateClient(),
	}
	c.rooms[roomName] = rc

	util.PrintDump(w, r, false)
	json.NewEncoder(w).Encode([]string{"OK"})
}

func (c *Context) hook(w http.ResponseWriter, r *http.Request) {
	payLoad, err := util.DecodePostJSON(r, true)
	if err != nil {
		log.Fatalf("Parsed auth data failed:%v\n", err)
	}
	roomID := strconv.Itoa(int((payLoad["item"].(map[string]interface{}))["room"].(map[string]interface{})["id"].(float64)))

	util.PrintDump(w, r, true)

	log.Printf("Sending notification to %s\n", roomID)
	notifRq := &hipchat.NotificationRequest{
		Message:       "nice <strong>Happy Hook Day!</strong>",
		MessageFormat: "html",
		Color:         "red",
	}
	if _, ok := c.rooms[roomID]; ok {
		_, err = c.rooms[roomID].hc.Room.Notification(roomID, notifRq)
		if err != nil {
			log.Printf("Failed to notify HipChat channel:%v\n", err)
		}
	} else {
		log.Printf("Room is not registered correctly:%v\n", c.rooms)
	}
}

func (c *Context) routes() *mux.Router {
    r := mux.NewRouter()
    //healthcheck route required by Micros
    r.Path("/healthcheck").Methods("GET").HandlerFunc(c.healthcheck)
    //descriptor for Atlassian Connect
    r.Path("/").Methods("GET").HandlerFunc(c.atlassianConnect)
    r.Path("/atlassian-connect.json").Methods("GET").HandlerFunc(c.atlassianConnect)

    // HipChat specific API routes
    r.Path("/installable").Methods("POST").HandlerFunc(c.installable)
    r.Path("/hook").Methods("POST").HandlerFunc(c.hook)

    r.PathPrefix("/").Handler(http.FileServer(http.Dir(c.static)))
    return r
}

func main() {
	var (
		port    = flag.String("port", os.Getenv("PORT"), "web server port")
		static  = flag.String("static", "./static/", "static folder")
		baseURL = flag.String("baseurl", os.Getenv("BASE_URL"), "local base url")
	)
	flag.Parse()

	context := &Context{
		baseURL: *baseURL,
		static:  *static,
		rooms:   make(map[string]*RoomConfig),
	}

	r := context.routes()
	http.Handle("/", r)
	http.ListenAndServe(":" + *port, nil)
}
