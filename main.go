package main

import (
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
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/post-score", scoreHandler)
	http.HandleFunc("/get-score/", getScoreHandler)
	http.ListenAndServe(":8080", nil)
}

type Initial struct {
	SessionID string
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("index.template.html")
	var uuidstr string = uuid.NewV4().String()
	initial := Initial{SessionID: uuidstr}
	tmpl.Execute(w, initial)
}

type PostedScore struct {
	SessionID string
	Score     string
}
type Output struct {
	Out string
}

var uuid_regex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[0-5][a-fA-F0-9]{3}-[089aAbB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)

func scoreHandler(w http.ResponseWriter, r *http.Request) {
	output := Output{"default value"}
	defer func() {
		fmt.Fprintf(w, output.Out)
	}()
	if r.Method != "POST" {
		output.Out = "illegal method"
		w.WriteHeader(400)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		output.Out = "illegal content type"
		w.WriteHeader(400)
		return
	}
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		output.Out = err.Error()
		fmt.Println(err.Error())
		return
	}
	postedScore := PostedScore{}
	err = json.Unmarshal(requestBody, &postedScore)
	if err != nil {
		output.Out = err.Error()
		fmt.Println(err.Error())
		return
	}

	const binPath = "/Users/nao/bin/lilypond"
	var id string = postedScore.SessionID
	var valid_id bool = uuid_regex.MatchString(id)
	if !valid_id {
		output.Out = "invalid SessionID"
		return
	}
	cmd := exec.Command(binPath, "-o", "output/"+id, "/dev/stdin")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		output.Out = err.Error()
		return
	}
	fmt.Print("Processing: uuid " + string(requestBody))
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, postedScore.Score)
	}()
	combinedOutput, _ := cmd.CombinedOutput()
	output.Out = string(combinedOutput)
}

func getScoreHandler(w http.ResponseWriter, r *http.Request) {
	var id string = r.URL.Path[len("/get-score/"):]
	var valid_id bool = uuid_regex.MatchString(id)

	if !valid_id {
		w.WriteHeader(403)
		fmt.Fprintf(w, "invalid id!")
		return
	}
	file, err := os.Open("output/" + id + ".pdf")
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	_, err = io.Copy(w, file)
}
