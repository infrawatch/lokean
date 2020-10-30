# Lokean
Small tool to save logs from unix socket bus to Loki 

## Building Lokean
### Building with Golang
```
cd $GOPATH/src/github.com/infrawatch/lokean
go build -o lokean cmd/main.go
```

### Building with Docker
```
git clone --depth=1 --branch=master https://github.com/infrawatch/lokean.git lokean; rm -rf ./lokean/.git
cd lokean
docker build -t lokean .
```


## Running Lokean
Run Loki
Run Lokean
Run sg-bridge and point it to the socket used by Lokean
