# SCP Battle

Vote for your favourite SCP at [scpbattle.craigbester.com](http://scpbattle.craigbester.com)

This is essentially a FaceMash clone for SCPs built using the [Echo web framework](https://github.com/labstack/echo) over a few days to learn [Go](https://golang.org/).

Open-sourced because there weren't any good examples of how to implement best-practices like cache-control headers when I started this.

## Development:

- Install [Go](https://golang.org/dl/) (^1.13).
- Start the webserver locally ([localhost:1323](http://localhost:1323/)):
```
go run main.go
```

- [Air](https://github.com/cosmtrek/air) can be used for live reloading during development:
```
go get -u github.com/cosmtrek/air
cd scpbattle
air
```

- Run tests:
```
go test ./store -cover
```

## Deployment

- Vendor dependencies:
```shell
go mod tidy
go mod vendor
```

- Build the executable:
```shell
go build -tags netgo -mod vendor -ldflags '-s -w' -o app
```

- Configure database and port environment variables:
```shell
export DATABASE_URL="postgres://user:password@address/database"
export PORT="8080"
```

- Start the server:
```shell
./app
```

### Links:

- SCP Foundation: http://www.scp-wiki.net/
- Echo: https://github.com/labstack/echo
- GORM: https://github.com/jinzhu/gorm
- Air: https://github.com/cosmtrek/air
