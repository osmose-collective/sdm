FROM            golang:1.22-alpine as build
ENV             GO111MODULE=on
RUN             apk add --no-cache git make
WORKDIR         /go/src/github.com/ultreme/sdm/map/api
COPY            go.mod go.sum ./
RUN             go mod download
COPY            . .
RUN             go get -v .
RUN             go install -v .

FROM            alpine
RUN             apk add --update --no-cache ca-certificates
COPY            --from=build /go/bin/api /bin/
ENTRYPOINT      ["api"]
EXPOSE          3333
