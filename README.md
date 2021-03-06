# SCP Battle

Vote for your favourite SCP at [scpbattle.tk](http://scpbattle.tk/) (redirects to Heroku).

This is essentially a FaceMash clone for SCPs built using the [Echo web framework](https://github.com/labstack/echo) over a few days to learn [Go](https://golang.org/).

Open-sourced because there weren't any good examples of how to implement best-practices like cache-control headers when I started this.

## Development:

- Install [Go v1.13](https://golang.org/dl/) (later versions may or may not work)
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

- Updating Heroku deployment:
```
go mod tidy
go mod vendor

# Commit any new go packages
# git add -A .
# git commit

git push heroku master
```

### Links:

- SCP Foundation: http://www.scp-wiki.net/
- Echo: https://github.com/labstack/echo
- GORM: https://github.com/jinzhu/gorm
- Air: https://github.com/cosmtrek/air
