package middleware

import "github.com/tedsuo/rata"

type CORS struct {
}

// AddOptionsRoutes appends the rataRoutes to support OPTIONS methods on each endpoint
func (c CORS) AddOptionsRoutes(handlerName string, routes rata.Routes) rata.Routes {
	for _, route := range routes {
		optionRoute := rata.Route{
			Name:   handlerName,
			Method: "OPTIONS",
			Path:   route.Path,
		}
		if !routeInRoutes(optionRoute, routes) {
			routes = append(routes, optionRoute)
		}
	}

	return routes
}

func routeInRoutes(route rata.Route, routes rata.Routes) bool {
	for _, r := range routes {
		if r == route {
			return true
		}
	}
	return false
}
