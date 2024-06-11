# Notifique
A Notification Service made in Go


## API Spec

Can be found on the `docs/openapi.yaml` file. Also, it can be visualized using swagger-ui:

```bash
docker run -p 80:8080 \
    -e SWAGGER_JSON=/app/openapi.yaml \
    -v ./api:/app \
    swaggerapi/swagger-ui
```

## DynamoDB

### Useful Commands

```bash
aws dynamodb scan \
    --endpoint-url http://localhost:8000 \
    --table-name notifications
```

```bash
go run github.com/google/wire/cmd/wire
```

## Useful Links

* https://swagger.io/resources/articles/best-practices-in-api-design/
* https://swagger.io/docs/specification/describing-parameters/#query-parameters
* https://swagger.io/docs/specification/data-models/data-types/#string
* https://gin-gonic.com/docs/examples/binding-and-validation/
* https://pkg.go.dev/github.com/go-playground/validator/v10#section-readme

* https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/gov2/dynamodb/actions/table_basics.go
* https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/GettingStarted.Query.html
* https://aws.github.io/aws-sdk-go-v2/docs/getting-started/
* https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/DynamoDBLocal.DownloadingAndRunning.html
* https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/best-practices.html
* https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue#Marshal
* https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/dynamodb#GetItemOutput

* https://www.alexdebrie.com/posts/dynamodb-single-table/

//go:build wireinject
// +build wireinject
postgres://postgres:postgres@localhost:5432/notifique?sslmode=disable