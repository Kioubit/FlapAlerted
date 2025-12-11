FROM golang:1.23.2-bookworm AS build

WORKDIR /go/src/project/
COPY . /go/src/project/

RUN make release MODULES="disable_mod_script"

FROM scratch
WORKDIR /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/src/project/bin/FlapAlerted /bin/FlapAlerted

EXPOSE 1790
EXPOSE 8699
LABEL description="FlapAlerted"
ENTRYPOINT ["/bin/FlapAlerted"]
