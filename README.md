# Caddy Jet

[![Build Status](https://travis-ci.org/9999years/caddy-jet.svg?branch=master)](https://travis-ci.org/9999years/caddy-jet)

A [Caddy][caddyserver] plugin enabling support for the [Jet template
engine][jet] meant to provide functionality (namely template inheritance)
missing or difficult to use with Go/Caddy’s default
[`text/template`][text/template] package/directive. The usage is similar to the
default [templates][templates] directive:

    jet [path [extensions...]]

But the default extensions are `.jet` and `.html`.

This is a direct fork of Caddy’s `templates` directive.

# Roadmap

* Make the plugin work instead of crash horribly
* Block directive format.
* Custom file inclusions / exclusions by glob or regex.
* Some way to supply a custom context with a `fastcgi` proxy; this is out of
  scope for now.

[caddyserver]: https://github.com/mholt/caddy
[jet]: https://github.com/CloudyKit/jet
[text/template]: https://golang.org/pkg/text/template/
[templates]: https://caddyserver.com/docs/templates
