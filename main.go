package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/Netflix/go-env"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Environment struct {
	ContainerName string `env:"CONTAINER_NAME,required=true"`
}

type ContainerStatus struct {
	State string `json:"state"`
}

func getContainer(ctx context.Context, cli *client.Client, name string) (*types.Container, error) {
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, c := range containers {
		if c.Names[0] == fmt.Sprintf("/%s", name) {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("container with name %s not found", name)
}

func statusHandler(w http.ResponseWriter, r *http.Request, cli *client.Client, container *types.Container) {
	fmt.Println("GET /status")
	json.NewEncoder(w).Encode(ContainerStatus{container.State})
}

func startHandler(w http.ResponseWriter, r *http.Request, cli *client.Client, container *types.Container) {
	fmt.Println("POST /start")
	if container.State == "paused" {
		if err := cli.ContainerUnpause(r.Context(), container.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(ContainerStatus{"unpausing"})
	} else {
		json.NewEncoder(w).Encode(ContainerStatus{container.State})
	}
}

func stopHandler(w http.ResponseWriter, r *http.Request, cli *client.Client, container *types.Container) {
	fmt.Println("POST /stop")
	if container.State == "running" {
		if err := cli.ContainerPause(r.Context(), container.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(ContainerStatus{"pausing"})
	} else {
		json.NewEncoder(w).Encode(ContainerStatus{container.State})
	}
}

var validPath = regexp.MustCompile("^/(status|start|stop)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, *client.Client, *types.Container), method string, cli *client.Client, containerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.NotFound(w, r)
			return
		}

		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}

		container, err := getContainer(r.Context(), cli, containerName)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		fn(w, r, cli, container)
	}
}

func main() {
	var environment Environment
	_, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		panic(err)
	}

	fmt.Println("Starting docker-pause-api...")
	fmt.Printf("Configured target container name: %s\n", environment.ContainerName)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	cli.NegotiateAPIVersion(context.Background())

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}

	fmt.Println("Currently found containers:")
	for _, c := range containers {
		fmt.Printf("ID: %s, Names: %s\n", c.ID, c.Names)
	}

	http.HandleFunc("/status", makeHandler(statusHandler, http.MethodGet, cli, environment.ContainerName))
	http.HandleFunc("/start", makeHandler(startHandler, http.MethodPost, cli, environment.ContainerName))
	http.HandleFunc("/stop", makeHandler(stopHandler, http.MethodPost, cli, environment.ContainerName))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
