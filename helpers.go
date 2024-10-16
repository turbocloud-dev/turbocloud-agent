package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"
)

func createTemplate(name, t string) *template.Template {
	return template.Must(template.New(name).Parse(t))
}

type malformedRequest struct {
	status int
	msg    string
}

func (mr *malformedRequest) Error() string {
	return mr.msg
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if mediaType != "application/json" {
			msg := "Content-Type header is not application/json"
			return &malformedRequest{status: http.StatusUnsupportedMediaType, msg: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5000)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{status: http.StatusBadRequest, msg: msg}
	}

	return nil
}

func executeScriptString(scriptString string) {

	scriptContents := []byte(scriptString)

	home_dir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf(" Cannot get home directory: %s\n", err.Error())
	}

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Deployment:", err)
	}
	fileName := home_dir + "/" + id + ".sh"

	err = os.WriteFile(fileName, scriptContents, 0644)
	if err != nil {
		fmt.Printf(" Cannot save script: %s\n", err.Error())
	}

	cmd := exec.Command("/bin/sh", fileName)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	cmd.Start()
	/*
		scannerStd := bufio.NewScanner(stdout)
		//scannerStd.Split(bufio.ScanLines)
		for scannerStd.Scan() {
			m := scannerStd.Text()
			fmt.Println(m)
		}

		scannerErr := bufio.NewScanner(stderr)
		//scannerErr.Split(bufio.ScanLines)
		for scannerErr.Scan() {
			m := scannerErr.Text()
			fmt.Println(m)
		}
		cmd.Wait()*/
	var wg sync.WaitGroup

	outch := make(chan string, 10)

	scannerStdout := bufio.NewScanner(stdout)
	wg.Add(1)
	go func() {
		for scannerStdout.Scan() {
			text := scannerStdout.Text()
			if strings.TrimSpace(text) != "" {
				outch <- text
			}
		}
		wg.Done()
	}()
	scannerStderr := bufio.NewScanner(stderr)
	wg.Add(1)
	go func() {
		for scannerStderr.Scan() {
			text := scannerStderr.Text()
			if strings.TrimSpace(text) != "" {
				outch <- text
			}
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(outch)
	}()

	for t := range outch {
		fmt.Println(t)
	}

	err = os.Remove(fileName) //remove the script file
	if err != nil {
		fmt.Printf(" Cannot remove script: %s\n", err.Error())
	}

}
