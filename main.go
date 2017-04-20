// vim:set ts=2 sts=2 et sw=2:
package main

import (
	"bufio"
	"encoding/json"
	"flag"
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

const listenPort = "8080"
const hourToExpireSession = 1

var onStage = true

type Session struct {
	sessionID  uuid.UUID
	issuedTime time.Time
}

var sessions []Session

func main() {
	localTestingFlg := flag.Bool("l", false, "local testing")
	flag.Parse()
	onStage = !(*localTestingFlg)

	if err := os.RemoveAll("working/"); err != nil {
		fmt.Println(err.Error())
	}
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/post-score", scoreHandler)
	http.HandleFunc("/get-score/", getScoreHandler)
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})
	fmt.Println("Listening port " + listenPort + "...")
	var err error
	if onStage {
		//Stage
		err = http.ListenAndServeTLS(":"+listenPort, "/etc/letsencrypt/live/norisio.net/fullchain.pem", "/etc/letsencrypt/live/norisio.net/privkey.pem", nil)
	} else {
		err = http.ListenAndServe(":"+listenPort, nil)
	}
	if err != nil {
		fmt.Println(err.Error())
		return
	}

}

type Initial struct {
	SessionID string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.template.html")
	if err != nil {
		fmt.Println(err.Error())
	}
	var newuuid uuid.UUID = uuid.NewV4()
	var newuuidstr string = newuuid.String()
	var newtime time.Time = time.Now()
	thisSession := Session{newuuid, newtime}
	sessions = append(sessions, thisSession)
	initial := Initial{SessionID: newuuidstr}
	err = tmpl.Execute(w, initial)
	if err != nil {
		fmt.Println(err.Error())
	}
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

	dirname := "working/" + id_str
	err = os.MkdirAll(dirname, 0744)
	if err != nil {
		output.Message = "cannot create directory"
		return
	}
	file, err := os.OpenFile(dirname+"/score.ly", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		output.Message = "cannot open ly file"
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
	//defer os.Remove("working/" + id_str + ".ly")

	wd, err := os.Getwd()
	wd = wd + "/" + dirname
	if err != nil {
		output.Message = err.Error()
		return
	}
	//docker run -it --rm -v $(pwd)/working/0d329b63-4818-4f7c-9e1d-eb36d4910853/:/app -w /app iskaron/lilypond lilypond score.ly
	cmd := exec.Command("docker", "run", "--rm", "-v", wd+":/app", "-w", "/app", "iskaron/lilypond", "lilypond", "score.ly")
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

	//filename := "output/" + id_str + ".pdf"
	//midifilename := "output/" + id_str + ".mid"
	filename := "working/" + id_str + "/score.pdf"
	midifilename := "working/" + id_str + "/score.mid"
	file, err := os.Open(filename)
	if err != nil {
		w.WriteHeader(404)
		//fmt.Fprintf(w, err.Error())
		fmt.Fprintf(w, "PDF出力がありません")
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
	_ = os.Remove(midifilename)
}
