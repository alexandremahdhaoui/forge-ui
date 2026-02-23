package controller

// PageRenderer renders a route to HTML.
type PageRenderer interface {
	Render(route string) (string, error)
}
