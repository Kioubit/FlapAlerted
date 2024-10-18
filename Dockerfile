FROM golang:1.23.2-bookworm AS build

WORKDIR /go/src/project/
COPY . /go/src/project/

RUN make release

FROM scratch
WORKDIR /
COPY --from=build /go/src/project/bin/FlapAlerted /bin/FlapAlerted

EXPOSE 1790:1790
EXPOSE 8699:8699
LABEL description="FlapAlerted"
ENTRYPOINT ["/bin/FlapAlerted"]
