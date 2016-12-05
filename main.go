package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"sync"

	"github.com/gorilla/mux"
)

var BattleNetAPI = "https://us.api.battle.net/wow/"
var apiKey = "qfkwuwv64smxuhcjzbuhmzbmhyh43v2y"

type Class struct {
	ID        int    `json:"id"`
	PowerType string `json:"powerType"`
	Name      string `json:"name"`
}

type Classes struct {
	List []Class `json:"classes"`
}

type Guild struct {
	Name         string       `json:"name"`
	Realm        string       `json:"realm"`
	Battlegroup  string       `json:"battlegroup"`
	GuildMembers GuildMembers `json:"members"`
}

type GuildMembers []GuildMember

type GuildMember struct {
	Character Character `json:"character"`
	Rank      int       `json:"rank"`
}

type Character struct {
	Name      string `json:"name"`
	Realm     string `json:"realm"`
	Level     int    `json:"level"`
	Class     int    `json:"class"`
	ClassName string `json:"className"`
	Race      int    `json:"race"`
	DPS       string `json:"dps"`
	Spec      Spec   `json:"spec"`
}

type Spec struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type Player struct {
	Name    string `json:"Name"`
	Realm   string `json:"realm"`
	Country string `json:"country"`
	DPS     string `json:"dps"`
}

type Players []Player

var classes Classes

func getClasses() {
	s := BattleNetAPI + "data/character/classes?locale=en_US&apikey=" + apiKey
	fmt.Println(s)

	resp, err := http.Get(s)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&classes)
}

func findClass(char *Character) {
	for i := range classes.List {
		c := classes.List[i]
		if c.ID == char.Class {
			char.ClassName = c.Name
		}
	}
}

func getGuild(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	s := BattleNetAPI + "guild/" + vars["realm"] + "/" + vars["guild"] + "?fields=members&locale=en_US&apikey=" + apiKey
	log.Println(s)

	resp, err := http.Get(s)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var g Guild
	json.NewDecoder(resp.Body).Decode(&g)

	var wg sync.WaitGroup
	for index := range g.GuildMembers {

		wg.Add(1)
		var ch = g.GuildMembers[index].Character

		go func() {
			defer wg.Done()
			findClass(&ch)

			if ch.Spec.Role == "DPS" && ch.Level == 110 {
				getDps(&ch)
			}
		}()
	}
	wg.Wait()
	json.NewEncoder(w).Encode(g)
}

func getDps(char *Character) {
	//TODO: don't hardcode this
	setEnv("PATH", "C:\\Simulationcraft(x64)\\710-03")
	path, err := exec.LookPath("simc.exe")

	if err != nil {
		log.Fatal("cannot find path")
	}

	fmt.Println("Simming " + char.Name)
	var args = fmt.Sprint("armory=us,", char.Realm, ",", char.Name)
	cmd := exec.Command(path, args)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile("DPS: [0-9]*.[0-9]*")
	test := re.FindString(out.String())
	words := strings.Fields(test)

	if len(words) == 0 {
		char.DPS = "n/a"
	}
	fmt.Println("Simulated: " + char.Name + "-" + char.Realm + " | " + words[1])
	char.DPS = words[1]
}

func setEnv(key, value string) {
	os.Setenv(key, value)
	if nowval := os.Getenv(key); value != nowval {
		println("Couldn't set `", key, "` env var, current value `", nowval, "`, wanted value `", value, "`")
	}
}

func main() {
	getClasses()

	port := "9343"
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/guild/{country}/{realm}/{guild}", getGuild)
	err := http.ListenAndServe(":"+port, router)

	//Look into why these logs don't work
	if err != nil {
		log.Fatal("Failed to start http server on port " + port)
		log.Fatal(err)
		return
	}

	log.Println("Successfully started HTTP server on port " + port)
}
