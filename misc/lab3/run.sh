#!/usr/bin/env bash
docker-compose up -d
docker build -t lab3 . && docker run --network hznet lab3