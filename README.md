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

## Useful Links

* https://swagger.io/resources/articles/best-practices-in-api-design/
* https://swagger.io/docs/specification/describing-parameters/#query-parameters
* https://swagger.io/docs/specification/data-models/data-types/#string
* https://gin-gonic.com/docs/examples/binding-and-validation/
* https://pkg.go.dev/github.com/go-playground/validator/v10#section-readme
