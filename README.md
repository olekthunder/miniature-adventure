# Micro basics (KPI lab)

Requires docker, docker-compose

## Running

```
docker-compose up
```

## Example requests

### Facade

#### Add message

```
curl localhost:8081/message/add -X POST --data '{"message": "foobarspam"}'
```

#### Index (list all logs and messages)
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
curl localhost:8182/log/list
```
