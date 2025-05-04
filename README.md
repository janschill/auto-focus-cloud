# auto-focus-cloud

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

