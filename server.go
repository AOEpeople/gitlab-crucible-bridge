package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var client http.Client

func healthHandler(cache *CrucibleRepositoriesCache) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cache.isEmpty() {
			http.Error(w, "not enough projects", http.StatusPreconditionFailed)
		}
	})
}

func handler(crucibleSettings CrucibleSettings, gitLabSettings GitLabSettings, crucibleCache *CrucibleRepositoriesCache) http.Handler {
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

		projectId := crucibleCache.getRepositoryName(gitUrl)

		if projectId == "" {
			http.Error(w, "project not found", http.StatusNotFound)
			return
		}

		err = TriggerCrucibleSync(projectId, client, crucibleSettings)
		if err != nil {
			log.Printf("error when sending request to Crucible: %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

	crucibleSettings := CrucibleSettings{
		ApiBaseUrl:             os.Getenv("CRUCIBLE_API_BASE_URL"),
		ApiKey:                 os.Getenv("CRUCIBLE_API_KEY"),
		Username:               os.Getenv("CRUCIBLE_USERNAME"),
		Password:               os.Getenv("CRUCIBLE_PASSWORD"),
		ProjectRefreshInterval: int(projectRefreshInterval),
		ProjectLimit:           int(projectLimit),
	}
	gitLabSettings := GitLabSettings{
		Token: os.Getenv("GITLAB_TOKEN"),
	}
	crucibleCache := &CrucibleRepositoriesCache{}
	updateCache := crucibleCache.updateFactory(crucibleSettings)
	updateCache()
	go cron(updateCache, time.Minute*time.Duration(crucibleSettings.ProjectRefreshInterval))

	log.Println(fmt.Sprintf("downloading Crucible project list every %d minute(s)", crucibleSettings.ProjectRefreshInterval))
	client = http.Client{
		Timeout: 5 * time.Second,
	}

	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gitlab_crucible_bridge_request_duration_seconds",
		Help: "Duration of HTTP requests in seconds",
		Buckets: prometheus.ExponentialBuckets(0.1, 3, 4),
	}, []string{"code"})
	prometheus.Register(histogram)
	http.Handle("/", promhttp.InstrumentHandlerDuration(
		histogram, handler(crucibleSettings, gitLabSettings, crucibleCache)))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/health", healthHandler(crucibleCache))
	log.Fatal(http.ListenAndServe(":8888", nil))
}
