package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/satori/go.uuid"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const lilypondBinPath = "/Users/nao/bin/lilypond"
const listenPort = "8080"
const hourToExpireSession = 5

type Session struct {
	sessionID  uuid.UUID
	issuedTime time.Time
}

var sessions []Session

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/post-score", scoreHandler)
	http.HandleFunc("/get-score/", getScoreHandler)
	http.HandleFunc("/pdfjs/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})
	http.ListenAndServe(":"+listenPort, nil)
}

type Initial struct {
	SessionID string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("index.template.html")
	var newuuid uuid.UUID = uuid.NewV4()
	var newuuidstr string = newuuid.String()
	var newtime time.Time = time.Now()
	thisSession := Session{newuuid, newtime}
	sessions = append(sessions, thisSession)
	initial := Initial{SessionID: newuuidstr}
	tmpl.Execute(w, initial)
}

func delete_s(s []Session, i int) []Session {
	s = append(s[:i], s[i+1:]...)
	n := make([]Session, len(s))
	copy(n, s)
	return n
}

var sessionsMutex sync.Mutex

func deleteOldSessions() {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	for i := 0; i < len(sessions); {
		if sessions[i].issuedTime.Before(time.Now().Add(-time.Duration(hourToExpireSession) * time.Hour)) {
			delete_s(sessions, i)
		} else {
			i += 1
		}
	}
}
func checkValidSession(sessionID uuid.UUID) bool {
	deleteOldSessions()
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	for i := range sessions {
		if uuid.Equal(sessions[i].sessionID, sessionID) {
			return true
		}
	}
	return false
}

type PostedScore struct {
	SessionID string
	Score     string
}
type Output struct {
	Success bool
	Message string
}

var uuid_regex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[0-5][a-fA-F0-9]{3}-[089aAbB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)

func scoreHandler(w http.ResponseWriter, r *http.Request) {
	output := Output{false, "default value(not shown)"}
	defer func() {
		json_string, _ := json.Marshal(output)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, string(json_string))
	}()
	if r.Method != "POST" {
		output.Message = "illegal method"
		w.WriteHeader(400)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		output.Message = "illegal content type"
		w.WriteHeader(400)
		return
	}
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		output.Message = err.Error()
		fmt.Println(err.Error())
		return
	}
	postedScore := PostedScore{}
	err = json.Unmarshal(requestBody, &postedScore)
	if err != nil {
		output.Message = err.Error()
		fmt.Println(err.Error())
		return
	}

	var id_str string = postedScore.SessionID
	id, err := uuid.FromString(id_str)
	if err != nil {
		output.Message = "invalid SessionID"
		return
	}
	var valid_session bool = checkValidSession(id)
	if !valid_session {
		output.Message = "no such session"
		return
	}

	file, err := os.OpenFile("posted/"+id_str+".ly", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		output.Message = "cannot write ly file"
		return
	}
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(postedScore.Score)
	if err != nil {
		output.Message = "cannot write ly file"
		return
	}
	writer.Flush()
	file.Close()
	defer os.Remove("posted/" + id_str + ".ly")

	cmd := exec.Command(lilypondBinPath, "-o", "output/"+id_str, "posted/"+id_str+".ly")
	//stdin, err := cmd.StdinPipe()
	//if err != nil {
	//output.Message = err.Error()
	//return
	//}
	fmt.Println("Processing: uuid " + string(postedScore.SessionID))
	//go func() {
	//defer stdin.Close()
	//io.WriteString(stdin, postedScore.Score)
	//}()
	combinedOutput, commandError := cmd.CombinedOutput()
	if commandError == nil { //exit status
		output.Success = true
	} else {
		output.Success = false
	}
	output.Message = string(combinedOutput)
	//respond to client by deferred function call
}

func getScoreHandler(w http.ResponseWriter, r *http.Request) {
	var id_str string = r.URL.Path[len("/get-score/"):]
	id_str = strings.Split(id_str, "?")[0]
	id, err := uuid.FromString(id_str)
	//var valid_id bool = uuid_regex.MatchString(id_str)
	if err != nil {
		w.WriteHeader(403)
		fmt.Fprintf(w, "invalid id!")
		return
	}
	if !checkValidSession(id) {
		w.WriteHeader(403)
		fmt.Fprintf(w, "no such session")
		return
	}

	filename := "output/" + id_str + ".pdf"
	file, err := os.Open(filename)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	_, err = io.Copy(w, file)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	err = os.Remove(filename)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
}
