FROM golang:1.27-trixie AS build

WORKDIR /src
COPY . .

RUN make release MODULES=disable_mod_script

FROM scratch
WORKDIR /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /src/bin/FlapAlerted /bin/FlapAlerted

USER 65534:65534

EXPOSE 1790
EXPOSE 8699
LABEL description="FlapAlerted"
ENTRYPOINT ["/bin/FlapAlerted"]
