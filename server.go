package main

import (
	"net/http"
	"time"
	"fmt"
	"log"
	"os"
	"strconv"
)

var client http.Client

func handler(crucibleSettings *CrucibleSettings, gitLabSettings GitLabSettings) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			return
		}

		gitUrl, err := GetNormalizedGitUrlFromRequest(r, gitLabSettings)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if gitUrl == "" {
			http.Error(w, "git url is empty. Is the hook in the proper format?", http.StatusBadRequest)
			return
		}

		var projectId string
		for _, project := range crucibleSettings.cachedCrucibleProjects {
			if project.Location == gitUrl {
				projectId = project.ID
			}
		}

		if projectId == "" {
			http.Error(w, "project not found", http.StatusNotFound)
			return
		}

		TriggerCrucibleSync(projectId, client, *crucibleSettings)
	})
}

func cron(f func(), d time.Duration) {
	ch := time.After(d)
	for {
		<-ch
		f()
		ch = time.After(d)
	}

}

func main() {
	projectRefreshInterval, err := strconv.ParseInt(os.Getenv("CRUCIBLE_PROJECT_REFRESH_INTERVAL"), 10, 32)
	if err != nil {
		panic(err)
	}
	projectLimit, err := strconv.ParseInt(os.Getenv("CRUCIBLE_PROJECT_LIMIT"), 10, 32)
	if err != nil {
		panic(err)
	}

	crucibleSettings := &CrucibleSettings{
		ApiBaseUrl: os.Getenv("CRUCIBLE_API_BASE_URL"),
		ApiKey:                 os.Getenv("CRUCIBLE_API_KEY"),
		Username:               os.Getenv("CRUCIBLE_USERNAME"),
		Password:               os.Getenv("CRUCIBLE_PASSWORD"),
		ProjectRefreshInterval: int(projectRefreshInterval),
		ProjectLimit:           int(projectLimit),
	}
	gitLabSettings := GitLabSettings{
		Token: os.Getenv("GITLAB_TOKEN"),
	}

	crucibleSettings.updateCachedProjects()
	go cron(crucibleSettings.updateCachedProjects, time.Minute*time.Duration(crucibleSettings.ProjectRefreshInterval))

	log.Println(fmt.Sprintf("downloading Crucible project list every %d minute(s)", crucibleSettings.ProjectRefreshInterval))
	client = http.Client{
		Timeout: 5 * time.Second,
	}

	http.Handle("/", handler(crucibleSettings, gitLabSettings))
	http.ListenAndServe(":80", nil)
}
