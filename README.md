# Micro basics (KPI lab)

IDK minimal go version, mine is 1.16

## Running

```
cd facade_service/
go mod download
go run main.go
```

```
cd logging_service/
go mod download
go run main.go
```

```
cd messages_service/
go run main.go
```

## Example requests

### Facade

#### Add message

```
curl localhost:8081/message/add -X POST --data '{"message": "foobarspam"}'
```

#### Index
```
curl localhost:8081/
```

### Messages

#### Get static message

```
curl localhost:8083/
```

### Logging (service)

### Add log

```
curl localhost:8082/log/add -X POST --data '{"message": "foobar", "uuid": "anystr"}'
```

### List logs

```
curl localhost:8082/log/list
```
