FROM gitlab.tarifer.ru:4567/oracle/golang-oci8-gocker-image:1.16.2

WORKDIR /go

ENV GOPATH="/"
#ENV SOURCE_DSN=user:password@host:1521/SID?schema_name=SOURCE_SCHEMA

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY ora/*.go ./ora/
COPY formats/*.go ./formats/

RUN go build *.go

ENV INIFILE=/cp.ini

ENTRYPOINT ["/go/oracli"]
