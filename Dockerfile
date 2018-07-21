# build container to build the app
FROM golang:alpine AS build

# need git to install dep
RUN apk add --no-cache git

WORKDIR /go/src/github.com/mybb/mybb-blog-mailer

ADD . /go/src/github.com/mybb/mybb-blog-mailer

RUN cd /go/src/github.com/mybb/mybb-blog-mailer && \
    go get -u github.com/golang/dep/cmd/dep && \
    dep ensure && \
    go build -o mybb-blog-mailer

# runtime container
FROM alpine

# need CA certificates to interact with Mailgun
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /go/src/github.com/mybb/mybb-blog-mailer/mybb-blog-mailer .
COPY --from=build /go/src/github.com/mybb/mybb-blog-mailer/templates ./templates

# set the listen port to 80 by default
ENV PORT=80

# Expose the port we're listening on
EXPOSE $PORT

# We define a volume to keep the last blog post file in
VOLUME [ "/var/log/mybb-blog-mailer" ]

CMD [ "/app/mybb-blog-mailer", \
    "-config=", \
    "-csrf_key_path=/var/log/mybb-blog-mailer/csrf_key", \
    "-session_key_path=/var/log/mybb-blog-mailer/session_key", \
    "-last_post_path=/var/log/mybb-blog-mailer/last_post_date" ]