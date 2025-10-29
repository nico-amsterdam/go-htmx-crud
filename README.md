# Go HTMX CRUD

## About this project

Rebuild of this [Vue CRUD Nuxt application](https://github.com/nico-amsterdam/vue-crud-nuxt) with [HTMX](https://htmx.org) and [Go](https://go.dev).
There is a demo on https://vue-crud-nuxt.nuxt.dev/. 

You can use this project for learning purposes and demo's.
Download, clone or fork the source from https://github.com/nico-amsterdam/go-htmx-crud.

It is deliberately kept simple, without abstractions, and all logic in main.go.
The shared state is not thread-safe.

A more structured example of HTMX with GO in a bigger project can be found [here](https://github.com/blackfyre/wga) 

Recommended HTMX reading material: [Following up "Mother of all htmx demos"](https://david.guillot.me/en/posts/tech/following-up-mother-of-all-htmx-demos/)

## Instructions

- install golang
- install air. air will automatically recompile changes
  - go install github.com/air-verse/air@latest
  - or with asdf: asdf install air latest
- git clone this repostory, or download the source from github
- cd vue-crud-nuxt
- compile and start: 
  - during development: air
  - or run: go run cmd/main.go
- open the browser
  - http://localhost:8778

