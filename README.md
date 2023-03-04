# foxyshot-indexer

An experimental addition to the foxyshot project. Search for text on your screenshots!

## Install

For initial install, run:
```
$ docker compose up -d
```

For continuous delivery run:
```
$ docker compose -f docker-compose-operations.yml up -d
```

## Development

The project is designed for development in docker. 

To start the dev environment, run:
```
$ make compose/dev/up
```
To run linters and tests, run:
```
$ make check/all
```
