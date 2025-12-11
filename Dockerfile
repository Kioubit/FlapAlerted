FROM golang:1.23.2-bookworm AS build

WORKDIR /go/src/project/
COPY . .

RUN make release-docker

FROM scratch
WORKDIR /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/src/project/bin/FlapAlerted /bin/FlapAlerted

USER 65534:65534

EXPOSE 1790
EXPOSE 8699
LABEL description="FlapAlerted"
ENTRYPOINT ["/bin/FlapAlerted"]
