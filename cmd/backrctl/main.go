package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cli "github.com/jawher/mow.cli"
	survey "gopkg.in/AlecAivazis/survey.v1"
)

func main() {

	app := cli.App("backrctl", "Utility to talk to backr server")

	app.Version("v version", "backrctl 1 (build 1)")

	app.Command("fetch", "Fetch backups url", func(cmd *cli.Cmd) {

		cmd.Spec = "[--default-scheme] [--default-port]"

		defaultScheme := cmd.StringOpt("default-scheme", "http", "Scheme used by backr instance to talk to")
		defaultPort := cmd.StringOpt("default-port", "22257", "Port used by backr instance to talk to")

		cmd.Action = func() {
			server := ""
			serverPrompt := &survey.Input{
				Message: "server",
				Help:    "IP or domain name",
			}
			survey.AskOne(serverPrompt, &server, nil)

			if server == "" {
				os.Exit(0)
			}

			baseURL := *defaultScheme + "://" + server + ":" + *defaultPort
			client := http.Client{
				Timeout: 4 * time.Second,
			}

			statusURL := url.URL{Scheme: *defaultScheme, Host: server + ":" + *defaultPort, Path: "/status"}
			statusURLStr := statusURL.String()
			statusResp, err := client.Get(statusURLStr)
			if err != nil {
				fmt.Printf("ERROR: unable to fetch /status on %v\n", baseURL)
				os.Exit(1)
			}
			defer statusResp.Body.Close()

			status := Status{}
			err = json.NewDecoder(statusResp.Body).Decode(&status)
			if err != nil {
				fmt.Printf("ERROR: unable to decode response of /status on %v\n", baseURL)
				os.Exit(1)
			}

			// build the list
			projectsList := []string{}
			for _, p := range status.Projects {
				projectsList = append(projectsList, p.Name)
			}

			sort.Sort(sort.StringSlice(projectsList))

			projectName := ""
			projectPrompt := &survey.Select{
				Message: "Choose a project:",
				Options: projectsList,
			}
			survey.AskOne(projectPrompt, &projectName, nil)

			if projectName == "" {
				os.Exit(0)
			}

			token := ""
			tokenPrompt := &survey.Input{
				Message: "token",
				Help:    "Get a token using 'backr token' on the target server",
			}
			survey.AskOne(tokenPrompt, &token, nil)

			if token == "" {
				os.Exit(0)
			}

			// get backup archives for the selected project
			query := url.Values{}
			query.Add("name", projectName)
			query.Add("token", token)
			backupsURL := url.URL{Scheme: *defaultScheme, Host: server + ":" + *defaultPort, Path: "/backups", RawQuery: query.Encode()}
			backupsURLStr := backupsURL.String()

			backupsResp, err := client.Get(backupsURLStr)
			if err != nil {
				fmt.Printf("ERROR: unable to fetch /backups with url: %v\n", backupsURLStr)
				os.Exit(1)
			}
			defer backupsResp.Body.Close()

			if backupsResp.StatusCode == 400 {
				fmt.Printf("ERROR: the request has been refused (error 400): %v\n", backupsURLStr)
				os.Exit(1)
			}
			if backupsResp.StatusCode == 401 {
				fmt.Printf("✗ token is not valid\n")
				os.Exit(1)
			}

			backups := []Backup{}
			err = json.NewDecoder(backupsResp.Body).Decode(&backups)
			if err != nil {
				fmt.Printf("ERROR: unable to decode response of /backups on %v\n", baseURL)
				os.Exit(1)
			}

			for i, b := range backups {
				name := filepath.Base(b.Name)
				name = strings.Replace(name, ".tar.gz", "", 1)
				t, _ := time.Parse(time.RFC3339, name)
				backups[i].CreatedAt = t
			}

			sort.Slice(backups, func(i, j int) bool {
				return backups[i].CreatedAt.Unix() > backups[j].CreatedAt.Unix()
			})

			backupsList := []string{}
			backupsByMenuItem := map[string]Backup{}
			for i, b := range backups {
				formattedTime := b.CreatedAt.Format("02/01/2006 15:04:05")
				backupsList = append(backupsList, formattedTime)
				backupsByMenuItem[formattedTime] = backups[i]
			}

			backupSelected := ""
			backupsPrompt := &survey.Select{
				Message: "Choose a backup:",
				Options: backupsList,
			}
			survey.AskOne(backupsPrompt, &backupSelected, nil)

			if backupSelected == "" {
				os.Exit(0)
			}

			fmt.Println()
			fmt.Println(backupsByMenuItem[backupSelected].URL)
			fmt.Println()
		}

	})

	app.Run(os.Args)
}

type Status struct {
	Projects []Project `json:"projects,omitempty"`
}

type Project struct {
	Name string `json:"name,omitempty"`
}

type Backup struct {
	Name      string
	URL       string
	CreatedAt time.Time
}
