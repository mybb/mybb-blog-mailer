# mybb-blog-mailer

This is an app to send email notifications via MailGun for new blog posts posted to the MyBB blog.

It works by reciving a GitHub webhook for the page build action, then reads the ATOM XML feed from the MyBB blog to get the most recently posted post. It compares the publish date of the most recent post to ensure it's actally new, then sends an email to a configured mailing list.

## Configuration

Configuration is done via a set of environment variables:

- `BLOG_MAILER_MG_DOMAIN` - **required** - the domain to send the email from, which should be configured within MailGun.
- `BLOG_MAILER_MG_API_KEY` - **required** - the API key for the MailGun account to send the email from.
- `BLOG_MAILER_MG_PUBLIC_API_KEY` - **required** - the public API key for the MailGun account to send the email from.
- `BLOG_MAILER_MG_MAILING_LIST_ADDRESS` - **required** - the address of the MailGun mailing list to send the email to.
- `BLOG_MAILER_HTTP_PORT` - the HTTP port for the server to listen on for incoming HTTP connections - defaults to `80`.
- `BLOG_MAILER_GH_HOOK_SECRET` - the secret used for the GitHub web hook - defaults to an empty string. This should be configured to a secret value to ensure only legitimate requests are processed.
- `BLOG_MAILER_XML_FEED_URL` - the URL of the XML feed to read blog posts from. Defaults to `https://blog.mybb.com/feed.xml`.
- `BLOG_MAILER_LAST_POST_FILE_PATH` - the path to the file to store the date of the last sent email in. Defaults to `./last_blog_post.txt`.
- `BLOG_MAILER_FROM_NAME` - the name to use when sending emails. Defaults to `MyBB Blog`.

## Building

This project uses [`dep`](https://github.com/golang/dep) to manage dependencies. Make sure you've installed `dep`, then run `dep ensure` to create the `vendor` directory with all of the vendor libraries.

You can then build the project for Linux, Mac and Windows x86_64 by running `make all`.