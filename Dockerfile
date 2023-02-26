FROM golang:1.18-bullseye AS build

WORKDIR /go/src/project/
COPY . /go/src/project/

RUN CGO_ENABLED=0 go build -tags=mod_httpAPI -trimpath -o /bin/FlapAlertedPro

FROM scratch
WORKDIR /
COPY --from=build /bin/FlapAlertedPro /bin/FlapAlertedPro

EXPOSE 1790:1790
EXPOSE 8699:8699
EXPOSE 8700:8700
LABEL description="FlapAlertedPro"
ENTRYPOINT ["/bin/FlapAlertedPro"]