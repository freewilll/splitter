# Demo Group Expenses REST API

# Quick start
Start Postgresql
```
$ docker run --name stream-postgres -e POSTGRES_PASSWORD=stream -d -p5432:5432 postgres
```

Create the database schema
```
$ go run main.go -create-schema
2021/01/16 21:10:07 Creating database schema
2021/01/16 21:10:07 Database schema has been created
```

Start Redis
```
$ docker run --name stream-redis -d -p 6379:6379 redis
```

Run the web server
```
$ go run main.go
2021/01/16 21:10:12 Listening on port 8080
```

The database comes with three users pre-created with emails:

- `test1@getstream.io`
- `test2@getstream.io`
- `test3@getstream.io`

The password is `secret` for all three. See the SQL in [database/postgres.go](database/postgres.go) for more details.

Authenticate all three users and save their cookies to `/tmp`
```
curl -X POST -c /tmp/cookies1.txt http://localhost:8080/signin -d '{"email": "test1@getstream.io", "password": "secret"}'
curl -X POST -c /tmp/cookies2.txt http://localhost:8080/signin -d '{"email": "test2@getstream.io", "password": "secret"}'
curl -X POST -c /tmp/cookies3.txt http://localhost:8080/signin -d '{"email": "test3@getstream.io", "password": "secret"}'
```

User 1 buys a meal with €42 for the other two users
```
curl -sb /tmp/cookies1.txt -X POST  http://localhost:8080/expenses -d '{"description":"Dinner","amount":42,"created_at":"2016-01-02T15:04:05Z", "users":[{"id": 2}, {"id":3}]}'
```

User 2 buys a coffee worth €8 for user 1.
```
curl -sb /tmp/cookies2.txt -X POST  http://localhost:8080/expenses -d '{"description":"Coffee","amount":8,"created_at":"2016-01-03T15:04:05Z", "users":[{"id": 1}]}'
```

To see the balance for all three users:
```
$ curl -b /tmp/cookies1.txt http://localhost:8080/balance
{"balance":24,"debit":[],"credit":[{"user_id":2,"amount":10},{"user_id":3,"amount":14}]}
```

```
$ curl -b /tmp/cookies2.txt http://localhost:8080/balance
{"balance":-10,"debit":[{"user_id":1,"amount":10}],"credit":[]}
```

```
$ curl -b /tmp/cookies3.txt http://localhost:8080/balance
{"balance":-14,"debit":[{"user_id":1,"amount":14}],"credit":[]}
```

# Implementation
- HTTP REST JSON API based on [net/http](https://golang.org/pkg/net/http/) with validation
- Postgresql backend database for users and expenses
- Redis cache with read/write through for the balance
- Authentication with JWT tokens.
- Unit and integration tests

# ERD

- users
    - id
    - email
    - password

- expenses
    - id
    - user_id
    - description
    - amount
    - created_at

- expenses_users
    - expense_id -> expenses
    - user_id -> users

# Future Improvements
- API
    - Use a web framework with before/after web functions & context for db handle & authentication information
    - Use context instead of passing `userID` around in wrappers
    - Improve routing
    - Swagger docs
    - Require `application/json` content type
    - Central logging of requests & request times etc
    - More routes
        - `GET /users/{id}`
        - `GET /expenses`
        - `GET /expenses/{id}`
    - Better authorization model, so not any user can register
    - Catch panics and report 500s
    - Respond with JSON instead of `text/plain` for errors such as 404s
    - Reasonable error messages when parsing json. If a parse fails, the response is unhelpful to the user.
    - JSON responses in `POST` APIs instead of 201s. In principle the equivalent response of the single GET endpoints should be used.
    - Move postgresql & redis flags to their own packages

- Postgresql
    - Use better pq package, [github.com/lib/pq](https://github.com/lib/pq) is unmaintaned
    - Database connection pooling
    - Postgresql schema migrations
    - Don't use auto commit but have transactions & auto rollback on panics

- Security
    - Better password requirements
    - Unhardcode default database/cache url and credentials in flags code
    - Unhardcode test users in schema creation
    - Store user passwords elsewhere, e.g. [Vault](https://www.vaultproject.io/)
    - Reduce token expiration time and add jwt token refresh
    - Unhardcode `"my-secret-stream-key"` in jwt code
    - Add a refresh endpoint for jwt tokens

- Application
    - Multi tenancy, allow people to create groups of users to share expenses with
    - Keeping track of who has already settled their debts up and reflect this in the balance
    - Multiple currencies
    - Use some kind of money values instead of `float64`. This requires working on JSON conversions, application logic and postgresql conversions
    - Prevent adding expenses in the future
    - There is a chance of a race condition leading to a stale cache if there are many concurrent writes, however a TTL mitigates this. Improve caching model to prevent this.
    - Minimise the amount of money transfers a group has to do to settle up

- Go
    - Add types for the integer values used for `UserID` and `ExpenseID` for better readability and compilation-time type checking
    - `VerifyToken` should perhaps return an error instead of `ok` so that the caller can handle distinct failure scenarios

- Areas that need better test coverage
    - postgres
    - redis
    - authentication
    - jwt token generation and verification
    - Authentication tests: in memory database should also implement authentication and unit tests can use that to check the auth handler

# Dependencies
The following third party packages were used:

- [github.com/lib/pq](https://github.com/lib/pq)
- [github.com/go-redis/redis](https://github.com/go-redis/redis)
- [github.com/dgrijalva/jwt-go](https://github.com/dgrijalva/jwt-go)
