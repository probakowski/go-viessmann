# go-viessmann

go-viessmann is a Go client library for accessing the [Viessmann Cloud API](https://developer.viessmann.com/)

[![Build](https://github.com/probakowski/go-viessmann/actions/workflows/build.yml/badge.svg)](https://github.com/probakowski/go-viessmann/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/probakowski/go-viessmann)](https://goreportcard.com/report/github.com/probakowski/go-viessmann)

## Installation

go-viessmann is compatible with modern Go releases in module mode, with Go installed:

```bash
go get github.com/probakowski/go-viessmann
```

will resolve and add the package to the current development module, along with its dependencies.

Alternatively the same can be achieved if you use import in a package:

```go
import "github.com/probakowski/go-viessmann"
```

and run `go get` without parameters.

Finally, to use the top-of-trunk version of this repo, use the following command:

```bash
go get github.com/probakowski/go-viessmann@master
```

## Usage ##

```go
import "github.com/probakowski/go-viessmann"
```

Construct a new viessmann client, then you can use different method from [API](https://developer.viessmann.com/), for
example:

```go
client := viessmann.Client{
    ClientId: "<your_client_id"
    RefreshToken "<OAuth refresh token"
    HttpClient: client, //optional, HTTP client to use, http.DefaultClient will be used if nil
}

...
```

### Authentication

You have to register as developer on [Viessmann Developer Portal](https://developer.viessmann.com) and create
[API key](https://developer.viessmann.com/en/clients)

`viessmann.Client` uses OAuth refresh token to obtain access token that is required for API access. This token can be
obtained manually as described in [Viessmann documentation](https://developer.viessmann.com/en/doc/authentication)
or you can use server located in `example/server/go`:

1. Start server: `go run example/server.go`
2. Go to http://localhost:3000
3. Provide Client ID and click `Log in`, you'll be redirected to Viessmann login page when you have to log in with
   credentials you used during registration
4. If everything is OK you will see overview of all your installations, gateways, devices and their features
5. Refresh token will be stored in `config` file in current directory
