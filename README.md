Example command:

    docker run -d -e CONTAINER_NAME=test -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 latenssi/docker-pause-api
