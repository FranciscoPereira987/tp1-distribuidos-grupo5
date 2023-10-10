FROM golang:1.21 AS builder
# Client uses docker multistage builds feature https://docs.docker.com/develop/develop-images/multistage-build/
# First stage is used to compile golang binary and second stage is used to only copy the 
# binary generated to the deploy image. 
# Docker multi stage does not delete intermediate stages used to build our image, so we need 
# to delete it by ourselves. Since docker does not give a good alternative to delete the intermediate images
# we are adding a very specific label to the image to then find these kind of images and delete them
LABEL intermediateStageToBeDeleted=true

RUN mkdir -p /build
WORKDIR /build/
COPY cmd/filtroDistancia/lib/ cmd/filtroDistancia/lib/
COPY cmd/filtroDistancia/main.go cmd/filtroDistancia/main.go
COPY pkg/ pkg/
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod tidy
RUN go mod vendor
RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -o bin/filtroDistancia github.com/franciscopereira987/tp1-distribuidos/cmd/filtroDistancia


FROM busybox:latest
COPY --from=builder /build/bin/filtroDistancia /filter
ENTRYPOINT ["/bin/sh"]