# Prepaid Card Example

A basic prepaid card package and JSON API example.

Makefile targets:

- `make test` - run unit tests
- `make build` - build the API binary
- `make run` - build and run the API binary

API Endpoints:

- `GET /accounts` - get all accounts
- `POST /accounts {"id":123}` - create a new account
- `GET /accounts/{id}` - get the account for the given ID
- `GET /acounts/{id}/statement` - account statement for the given ID
- `POST /accounts/{id}/load {"amount":"10.50"}` - load money request
- `POST /accounts/{id}/authorize {"merchantID":321,"amount":"10.50"}` - authorize request
- `POST /accounts/{id}/capture {"merchantID":321,"amount":"10.50"}` - capture request
- `POST /accounts/{id}/reverse {"merchantID":321,"amount":"10.50"}` - reverse request
- `POST /accounts/{id}/refund {"merchantID":321,"amount":"10.50"}` - refund request
