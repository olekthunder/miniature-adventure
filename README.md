# lab 5

RabbitMQ GUI is running on localhost:15672

```
+------------------+
| login    | guest |
|----------|-------|
| password | guest |
+------------------+
```

## task 1

Producer-consumer:

```
docker-compose up consumer1 producer1
```

Publisher-Subscriber:

```
docker-compose up publisher1 subscriber11 subscriber12
```

## task 2


```
docker-compose up client2 worker2
```

```docker-compose up client2```

**wait for producing**

```docker-compose up worker2```

## task 3

#### With max-length and `x-overflow=drop-head` (default)

```
docker-compose up producer31
```
**wait for command completion**

```docker-compose up consumer31```

#### With max-length and `x-overflow=reject-publish`

```
docker-compose up producer32
```
**wait for command completion**

```docker-compose up consumer32```

## task4

```docker-compose up producer4```

**wait for command completion**

```docker-compose stop rmq```

```docker-compose up consumer4 rmq```

## task5

```docker-compose up producer5 consumer5```

## task6

```docker-compose up consumer6_no_ack consumer6_ack```

**wait for consuming to start**

```docker-compose up producer6```

**wait for producing ends**

```docker-compose stop consumer6_no_ack```