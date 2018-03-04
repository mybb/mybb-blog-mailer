FROM alpine:3.6

# We need CA certificates to interact with Mailgun
RUN apk add --update ca-certificates

WORKDIR /app

COPY mybb-blog-mailer-linux-amd64 /app/mybb-blog-mailer
COPY templates/* /app/templates/

# These arguments can be specified when building the container, using the `--build-arg` flag
ARG mailgun_api_key
ARG mailgun_public_api_key
ARG mailing_list_address
ARG hook_secret
ARG mailgun_domain=mybb.com
ARG http_port=80
ARG xml_feed_url=https://blog.mybb.com/feed.xml

# Expose the port we're listening on
EXPOSE $http_port

# We define a volume to keep the last blog post file in
VOLUME ["/var/log/mybb-blog-mailer"]

# We set the required environment variables based on the build args
ENV BLOG_MAILER_MG_DOMAIN=$mailgun_domain
ENV BLOG_MAILER_MG_API_KEY=$mailgun_api_key
ENV BLOG_MAILER_MG_PUBLIC_API_KEY=$mailgun_public_api_key
ENV BLOG_MAILER_MG_MAILING_LIST_ADDRESS=$mailing_list_address
ENV BLOG_MAILER_HTTP_PORT=$http_port
ENV BLOG_MAILER_GH_HOOK_SECRET=$hook_secret
ENV BLOG_MAILER_XML_FEED_URL=$xml_feed_url
ENV BLOG_MAILER_LAST_POST_FILE_PATH /var/log/mybb-blog-mailer/last_blog_post.log

CMD ["/app/mybb-blog-mailer"]