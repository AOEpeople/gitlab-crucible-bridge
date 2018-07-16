package main

import (
	"net/http"
	"time"
	"fmt"
	"log"
	"os"
	"strconv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
)

var client http.Client

func healthHandler(crucibleSettings *CrucibleSettings) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projects := crucibleSettings.getProjects()
		if len(projects) > 0 {
			return
		} else {
			http.Error(w, "not enough projects", http.StatusPreconditionFailed)
			return
		}
	})
}

func handler(crucibleSettings *CrucibleSettings, gitLabSettings GitLabSettings, histogram prometheus.HistogramVec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			return
		}

		start := time.Now()
		gitUrl, err := GetNormalizedGitUrlFromRequest(r, gitLabSettings)
		if err != nil {
			duration := time.Since(start)
			histogram.WithLabelValues(fmt.Sprintf("%d", http.StatusBadRequest)).Observe(duration.Seconds())

			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if gitUrl == "" {
			duration := time.Since(start)
			histogram.WithLabelValues(fmt.Sprintf("%d", http.StatusBadRequest)).Observe(duration.Seconds())
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
			duration := time.Since(start)
			histogram.WithLabelValues(fmt.Sprintf("%d", http.StatusNotFound)).Observe(duration.Seconds())
			http.Error(w, "project not found", http.StatusNotFound)
			return
		}

		err = TriggerCrucibleSync(projectId, client, *crucibleSettings)
		if err != nil {
			duration := time.Since(start)
			histogram.WithLabelValues(fmt.Sprintf("%d", http.StatusInternalServerError)).Observe(duration.Seconds())
		} else {
			duration := time.Since(start)
			histogram.WithLabelValues(fmt.Sprintf("%d", http.StatusOK)).Observe(duration.Seconds())
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

	crucibleSettings := &CrucibleSettings{
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

	crucibleSettings.updateCachedProjects()
	go cron(crucibleSettings.updateCachedProjects, time.Minute*time.Duration(crucibleSettings.ProjectRefreshInterval))

	log.Println(fmt.Sprintf("downloading Crucible project list every %d minute(s)", crucibleSettings.ProjectRefreshInterval))
	client = http.Client{
		Timeout: 5 * time.Second,
	}

	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gitlab_crucible_bridge_request_duration_seconds",
		Help: "Duration of HTTP requests in seconds",
	}, []string{"status"})
	prometheus.Register(histogram)
	http.Handle("/", handler(crucibleSettings, gitLabSettings, *histogram))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/health", healthHandler(crucibleSettings))
	log.Fatal(http.ListenAndServe(":8888", nil))
}
