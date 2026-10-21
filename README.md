# Site Dev Backend
Backend of the Site Development module (site survey, AI location insight, approval), rebuilt on the FamilyMart Go boilerplate (`boilerplate-backendgo`). Same API as `site-dev-api`; structure and rules follow the boilerplate below.

## Contents
- [Requirement](#requirement)
- [Description](#description)
- [Project Structure](#project-structure)
- [Migration DB](#migration-db)
- [Local Infrastructure](#local-infrastructure)
- [Tools](#tools)
- [Testing](#testing)
- [Benchmarks](#benchmarks)

## Requirement
- Go 1.18+
- Docker
- Git
- VS Code / Goland
- Coffee & Cake

## Description
This is the first boilerplate project with [Clean Architechture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) 

Main feature is :

| Features | Library |
| -------- | ------- |
| HTTP Server | echo, edenlabs |
| HTTP Client | edenlabs, hystrix |
| GRPC Server | edenlabs |
| GRPC Client | edenlabs |
| Pub/Sub     | watermill, sarama |
| ORM         | beego client, mysql |
| Redis       | |
| MongoDB     | |
| Logging     | logrus |
| Exeption Tracking | lentry |
| Discovery | health check |
| Command     | cobra |
| Config File | viper |
| Observability | otlp, jaeger, prometheus |


## Project Structure
```
cmd
config
docker
internal
├── app
│   ├── client
│   ├── constants
│   ├── consumer
│   ├── provider
│   ├── dto
│   ├── handler
│   ├── middleware
│   ├── mocks
│   ├── model
│   ├── producer
│   ├── repository
│   ├── scheduler
│   ├── server
│   └── service
└── pkg
vendor
```

## Local Infrastructure
- Start the requirement server
```
docker-compose up -d
```
- Check server, docker will be created :
  - MySQL Server
  - MongoDB
  - Redis
  - Elasticsearch
  - Kibana
  - Zookeeper
  - Kafka
  - Jaeger
  - Prometheus
  - Opentelemetry Collector

## Migration DB
- Open [Migration Project](https://git.edenfarm.id/project-version3/erp-databases)
- Follow the instruction

## Tools
- Install [Edenlabs CLI](https://git.edenfarm.id/edenlabs/cli)
- Start Debug & Run your instance
```
edenlabs run
# or
edenlabs grpc
# or
edenlabs consumer
```

## Testing
- Install mockery for automated create mocks
- Create file test
- Run Test

## Benchmark

