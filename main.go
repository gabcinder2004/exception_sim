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

func simulate(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var players Players
	err := decoder.Decode(&players)
	if err != nil {
		panic(err)
	}

	defer req.Body.Close()

	// var wg sync.WaitGroup
	// for index := range players {
	// 	wg.Add(1)
	// 	var p = players[index]
	// 	go func() {
	// 		defer wg.Done()
	// 		p.DPS = getDps(p.Country, p.Realm, p.Name)
	// 	}()
	// }

	// wg.Wait()
	json.NewEncoder(rw).Encode(players)
}

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
	router.HandleFunc("/simulate", simulate)
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

//////////////////////////////////////

// package main

// import (
// 	"encoding/json"
// 	"log"
// 	"net/http"

// 	"github.com/gorilla/mux"
// )

// type Person struct {
// 	ID        string   `json:"id,omitempty"`
// 	Firstname string   `json:"firstname,omitempty"`
// 	Lastname  string   `json:"lastname,omitempty"`
// 	Address   *Address `json:"address,omitempty"`
// }

// type Address struct {
// 	City  string `json:"city,omitempty"`
// 	State string `json:"state,omitempty"`
// }

// var people []Person

// func GetPersonEndpoint(w http.ResponseWriter, req *http.Request) {
// 	params := mux.Vars(req)
// 	for _, item := range people {
// 		if item.ID == params["id"] {
// 			json.NewEncoder(w).Encode(item)
// 			return
// 		}
// 	}
// 	json.NewEncoder(w).Encode(&Person{})
// }

// func GetPeopleEndpoint(w http.ResponseWriter, req *http.Request) {
// 	json.NewEncoder(w).Encode(people)
// }

// func CreatePersonEndpoint(w http.ResponseWriter, req *http.Request) {
// 	params := mux.Vars(req)
// 	var person Person
// 	_ = json.NewDecoder(req.Body).Decode(&person)
// 	person.ID = params["id"]
// 	people = append(people, person)
// 	json.NewEncoder(w).Encode(people)
// }

// func DeletePersonEndpoint(w http.ResponseWriter, req *http.Request) {
// 	params := mux.Vars(req)
// 	for index, item := range people {
// 		if item.ID == params["id"] {
// 			people = append(people[:index], people[index+1:]...)
// 			break
// 		}
// 	}
// 	json.NewEncoder(w).Encode(people)
// }

// func main() {
// 	router := mux.NewRouter()
// 	people = append(people, Person{ID: "1", Firstname: "Nic", Lastname: "Raboy", Address: &Address{City: "Dublin", State: "CA"}})
// 	people = append(people, Person{ID: "2", Firstname: "Maria", Lastname: "Raboy"})
// 	router.HandleFunc("/people", GetPeopleEndpoint).Methods("GET")
// 	router.HandleFunc("/people/{id}", GetPersonEndpoint).Methods("GET")
// 	router.HandleFunc("/people/{id}", CreatePersonEndpoint).Methods("POST")
// 	router.HandleFunc("/people/{id}", DeletePersonEndpoint).Methods("DELETE")
// 	log.Fatal(http.ListenAndServe(":12345", router))
// }
