FROM golang:1.18.0-bullseye AS build

WORKDIR /go/src/project/
COPY ./* /go/src/project/

ARG CGO_ENABLED=0
RUN go build -o /bin/FlapAlertedPro

FROM scratch
WORKDIR /
COPY --from=build /bin/FlapAlertedPro /bin/FlapAlertedPro

EXPOSE 1790:1790
EXPOSE 8699:8699
ENTRYPOINT ["/bin/FlapAlertedPro"]
