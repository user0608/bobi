---
name: httpserver
description: Use Bobi httpserver with Echo for routes, handlers, middleware, and Fx-managed HTTP servers. Trigger when using httpserver.NewServer, Route, AsRoute, PublicHandler, or setup.Service.Run.
---

# Bobi HTTP Server

Import `github.com/user0608/bobi/httpserver`. The server uses Echo v5 and registers values implementing `httpserver.Route`.

## Route constructors

```go
type UserHandler struct {
	service *UserService
}

func (handler *UserHandler) List(c *echo.Context) error {
	users, err := handler.service.List(c.Request().Context())
	if err != nil {
		return answer.Err(c, err)
	}

	return answer.Ok(c, users)
}

func ListUsersRoute(handler *UserHandler) *httpserver.PublicHandler {
	return &httpserver.PublicHandler{
		Method:  http.MethodGet,
		Path:    "/users",
		Handler: handler.List,
	}
}
```

`bobi` only provides `httpserver.PublicHandler` as a ready-to-use route. Use it
for public endpoints and define one named constructor per endpoint that returns
`*httpserver.PublicHandler`. Keep business dependencies in an application
handler or service, not in a new route struct for every public endpoint. The
constructor name does not need the `New` prefix.

Do not instantiate `PublicHandler` inside another handler, create a custom route
struct only to hold method and path for a public endpoint, or register an
anonymous constructor with `AsRoute`.

## Protected routes

`bobi` does not provide a protected-route type or an authorization middleware.
For a protected endpoint, the application must define a type that implements
`httpserver.Route` and, when needed,
`BeforeSecurityMiddlewareProvider` or `AfterSecurityMiddlewareProvider`:

```go
type AdminRoute struct {
	service *AdminService
}

func AdminRouteProvider(service *AdminService) *AdminRoute {
	return &AdminRoute{service: service}
}

func (route *AdminRoute) GetMethod() string {
	return http.MethodGet
}

func (route *AdminRoute) GetPath() string {
	return "/admin/users"
}

func (route *AdminRoute) HandleRequest(c *echo.Context) error {
	users, err := route.service.List(c.Request().Context())
	if err != nil {
		return answer.Err(c, err)
	}

	return answer.Ok(c, users)
}

func (route *AdminRoute) BeforeSecurityMiddlewares() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{RequireAdmin}
}
```

Register the protected route constructor in the same route group:

```go
fx.Provide(
	httpserver.AsRoute(AdminRouteProvider),
)
```

Only create an application-specific route struct when the route needs behavior
that `PublicHandler` cannot represent, such as protected-route middleware.

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
)
```

`httpserver.Module` creates the Echo server. `setup.Service.Run` invokes `httpserver.StartWebServer`, which reads `address` and `log_fmt` from Viper configuration and performs graceful shutdown.

## Middleware

Implement `BeforeSecurityMiddlewareProvider` or `AfterSecurityMiddlewareProvider` when a route supplies middleware. `BeforeMiddlewares` and `Middlewares` on `PublicHandler` are placed in those respective phases. Do not assume Bobi provides authentication or authorization middleware; wire those in the application.
