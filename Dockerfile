FROM golang:1.20-bullseye AS build

WORKDIR /go/src/project/
COPY . /go/src/project/

RUN CGO_ENABLED=0 go build -tags=mod_httpAPI -trimpath -o /bin/FlapAlerted

FROM scratch
WORKDIR /
COPY --from=build /bin/FlapAlerted /bin/FlapAlerted

EXPOSE 1790:1790
EXPOSE 8699:8699
EXPOSE 8700:8700
LABEL description="FlapAlerted"
ENTRYPOINT ["/bin/FlapAlerted"]