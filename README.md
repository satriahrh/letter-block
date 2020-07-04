# letter-block

[![Build Status](https://travis-ci.com/satriahrh/letter-block.svg?branch=master)](https://travis-ci.com/satriahrh/letter-block)
[![codecov](https://codecov.io/gh/satriahrh/letter-block/branch/master/graph/badge.svg)](https://codecov.io/gh/satriahrh/letter-block)

## About

Back end API of [Letter Block Game](https://letter-block.herokuapp.com)

## Requirements

For development, you will needing these following applications:
- Go 1.12.5 or later
- Redis 5.0.4
- Mysql 5.7

And please use the appropriate [Editorconfig](http://editorconfig.org/) plugin for your Editor (not mandatory).

### Configure app

Copy `env.sample` to `.env` then edit it with the url where you have setup. Do not forget to make your own RSA 256 key pair.

## Languages & tools

### Go

- [Go](https://golang.org/doc/devel/release.html#go1.12) is used for API.

### Data

- [MySQL](https://dev.mysql.com/doc/relnotes/mysql/5.7/en/) is used to store the data.
- [Redis](https://redis.io/) is used to cache lemma validation.

### Dictionaries

- [KBBI Daring](https://kbbi.kemdikbud.go.id/) is used to validate Indonesian lemma.

## Contribution

### Contacting the owner

Email via [satriah\<at\>gmail\<dot\>com](mailto:satriahrh@gmail.com), telegram to [t.me/satriahrh](https://t.me/satriahrh)

### Public Trello board

You can access the public trello board in https://trello.com/b/FH4mhddb. To join the board as member, please contact the owner.

### Issues

Please provide your thought in github issue tab.
