---
name: httpserver
description: Use Bobi httpserver with Echo for routes, handlers, middleware, and Fx-managed HTTP servers. Trigger when using httpserver.NewServer, Route, AsRoute, PublicHandler, PrivateHandler, MiddlewareResolver, or setup.Service.Run.
---

# Bobi HTTP Server

Import `github.com/user0608/bobi/httpserver`. The server uses Echo v5 and registers values implementing `httpserver.Route`.

## Route constructors

```go
func ListUsersRoute(service *UserService) *httpserver.PublicHandler {
	return &httpserver.PublicHandler{
		Method: http.MethodGet,
		Path:   "/users",
		Handler: func(c *echo.Context) error {
			users, err := service.List(c.Request().Context())
			if err != nil {
				return answer.Err(c, err)
			}

			return answer.Ok(c, users)
		},
	}
}
```

`bobi` provides `httpserver.PublicHandler` and `httpserver.PrivateHandler` as
ready-to-use route types. Both have `Method`, `Path`, and `Handler` fields and
implement `httpserver.Route`. An empty `Method` defaults to GET; a nil `Handler`
returns HTTP 500. Define one named constructor per endpoint returning the
appropriate route type. Inject services and other dependencies directly into
the route constructor and define the handler inline in its `Handler` field. Do
not define a separate application handler and inject it into the route
constructor, or create a new route struct for every public endpoint. The
constructor name does not need the `New` prefix.

Do not instantiate `PublicHandler` inside another handler, create a custom route
struct only to hold method and path for a public endpoint, or register an
anonymous constructor with `AsRoute`.

## Private routes

Use `httpserver.PrivateHandler` for private endpoints, with the handler defined
inline just as for public endpoints:

```go
func ListPrivateUsersRoute(service *UserService) *httpserver.PrivateHandler {
	return &httpserver.PrivateHandler{
		Method: http.MethodGet,
		Path:   "/private/users",
		Handler: func(c *echo.Context) error {
			users, err := service.List(c.Request().Context())
			if err != nil {
				return answer.Err(c, err)
			}

			return answer.Ok(c, users)
		},
	}
}
```

`PrivateHandler` is a route classification, not built-in access control. Its
request handling is identical to `PublicHandler`. The application must provide
security middleware and select it through `httpserver.MiddlewareResolver`.
Without that resolver, private routes are registered without route-specific
middleware too.

## Additional route types

For another route category, define a reusable abstraction implementing
`httpserver.Route`, rather than a new type for each endpoint. For example,
embedding `PrivateHandler` reuses its fields and Route methods:

```go
type AdminHandler struct {
	httpserver.PrivateHandler
}

var _ httpserver.Route = (*AdminHandler)(nil)

func ListAdminUsersRoute(service *UserService) *AdminHandler {
	return &AdminHandler{
		PrivateHandler: httpserver.PrivateHandler{
			Method: http.MethodGet,
			Path:   "/admin/users",
			Handler: func(c *echo.Context) error {
				users, err := service.List(c.Request().Context())
				if err != nil {
					return answer.Err(c, err)
				}

				return answer.Ok(c, users)
			},
		},
	}
}
```

The resolver can distinguish `*AdminHandler` from `*httpserver.PrivateHandler`.
Embedding does not make an `*AdminHandler` match a type assertion for
`*httpserver.PrivateHandler`; handle each category explicitly.

## Fx registration

When using `setup.Service`, register the named constructor of each route with
`httpserver.AsRoute`. Fx then adds the result to the `http-api-routes` group:

```go
fx.Provide(
	httpserver.AsRoute(ListUsersRoute),
)
```

For multiple routes, provide each named constructor explicitly:

```go
fx.Provide(
	httpserver.AsRoute(CreateUserRoute),
	httpserver.AsRoute(ListUsersRoute),
	httpserver.AsRoute(DeleteUserRoute),
	httpserver.AsRoute(ListPrivateUsersRoute),
	httpserver.AsRoute(ListAdminUsersRoute),
)
```

`httpserver.Module` creates the Echo server. `httpserver.NewServer` receives a
`httpserver.ServerParams` value containing the grouped routes and an optional
`MiddlewareResolver`. `setup.Service.Run` invokes
`httpserver.StartWebServer`, which reads `address` and `log_fmt` from Viper
configuration and performs graceful shutdown.

## Middleware

Implement `httpserver.MiddlewareResolver` in the application when routes need
route-specific middleware:

```go
type MiddlewareResolver struct{}

func (resolver *MiddlewareResolver) Resolve(route httpserver.Route) []echo.MiddlewareFunc {
	switch route.(type) {
	case *httpserver.PublicHandler:
		return nil
	case *httpserver.PrivateHandler:
		return []echo.MiddlewareFunc{RequireAuth}
	case *AdminHandler:
		return []echo.MiddlewareFunc{RequireAuth, RequireAdmin}
	default:
		// Fail closed for route categories not explicitly supported.
		return []echo.MiddlewareFunc{func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				return echo.NewHTTPError(http.StatusForbidden)
			}
		}}
	}
}

func NewMiddlewareResolver() httpserver.MiddlewareResolver {
	return &MiddlewareResolver{}
}

// Register it when using setup.Service.
service.Run(fx.Provide(NewMiddlewareResolver))
```

`RequireAuth` and `RequireAdmin` above are application-defined
`echo.MiddlewareFunc` values, not Bobi-provided middleware. Their per-request
checks belong inside the middleware, not inside `Resolve`.

`NewServer` calls `Resolve` once for each registered route while creating the
Echo server, then passes the returned slice directly to `server.Add` alongside
the route's method, path, and `HandleRequest`. Return the middleware in the
desired Echo order, or `nil` when the route needs no additional middleware.
The server does not choose middleware based on route type itself; that policy
belongs entirely to the application resolver.

Register the resolver as `httpserver.MiddlewareResolver`, as shown above, so Fx
can inject it into the optional `ServerParams.MiddlResolver` field. For manual
server construction, set that field directly. Without a resolver, all route
types are registered without route-specific middleware. Global request logging,
recovery, and CORS middleware are still installed by `NewServer`.
