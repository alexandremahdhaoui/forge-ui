package controller

// RenderResult holds the rendered HTML for both the side navigation and the
// main content area.
type RenderResult struct {
	SideNav string
	Content string
}

// PageRenderer renders a route to HTML.
type PageRenderer interface {
	Render(route string) (RenderResult, error)
}
