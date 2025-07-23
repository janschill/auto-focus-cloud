# auto-focus-cloud

## To-dos

- [ ] Webhook for Stripe checkout completed
- [ ] Customer management
- [ ] License management
- [ ] Deployment
- [ ] Cloudflare proxy

## Getting Started

```sh
go run main.go
```

## Stripe CLI

```sh
stripe login
stripe listen --forward-to localhost:8080/api/webhooks/stripe
stripe trigger customer.subscription.created
```

## Plan

Phase 1: Core HTTP Server (Go Book Chapter 7)

- Use net/http standard library only
- Simple multiplexer with http.ServeMux
- Learn HTTP handlers, request/response fundamentals
- Environment variables with os package

Phase 2: JSON & Data Handling (Go Book Chapter 4)

- Manual JSON parsing with encoding/json
- Custom types and struct tags
- Input validation without external libraries
- Error handling patterns

Phase 3: File-Based Storage (Go Book Chapter 1, 3)

- Start with simple file storage (JSON files)
- Learn os, io, bufio packages
- File locking for concurrent access
- Progress to database/sql with SQLite later

Phase 4: Concurrency & Testing (Go Book Chapter 8, 9, 11)

- Goroutines for handling requests
- Channels for coordination
- sync package for shared state
- Unit tests with testing package
