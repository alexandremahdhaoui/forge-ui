package httpdriver

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockcontroller"
	"github.com/stretchr/testify/assert"
)

// newTestHandler creates a Handler with minimal templates and mock services.
func newTestHandler(ps *mockcontroller.PortfolioService, ws *mockcontroller.WorkspaceService, fs *mockcontroller.ForgeService) *Handler {
	tmpl := template.Must(template.New("layout").Parse(`{{define "layout"}}ok{{end}}`))
	return &Handler{
		BaseDir: "/base",
		Templates: map[string]*template.Template{
			"portfolios": tmpl,
			"portfolio":  tmpl,
			"workspace":  tmpl,
			"forge":      tmpl,
		},
		HomeURL:          "/portfolios",
		PortfolioService: ps,
		WorkspaceService: ws,
		ForgeService:     fs,
	}
}

// --- HandlePortfolios ---

func TestHandlePortfolios_Success(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{
		Portfolios: []types.PortfolioSummary{{Name: "p1"}},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios", nil)
	w := httptest.NewRecorder()

	h.HandlePortfolios(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	ps.AssertExpectations(t)
}

func TestHandlePortfolios_SortParam(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "name").Return(types.PortfoliosPageData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios?sort=name", nil)
	w := httptest.NewRecorder()

	h.HandlePortfolios(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	ps.AssertExpectations(t)
}

func TestHandlePortfolios_Error(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{}, errors.New("disk error"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios", nil)
	w := httptest.NewRecorder()

	h.HandlePortfolios(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "disk error")
	ps.AssertExpectations(t)
}

// --- HandlePortfolio ---

func TestHandlePortfolio_Success(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "myp", "time").Return(types.PortfolioPageData{
		Name: "myp",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/myp", nil)
	req.SetPathValue("name", "myp")
	w := httptest.NewRecorder()

	h.HandlePortfolio(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	ps.AssertExpectations(t)
}

func TestHandlePortfolio_MissingName(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/", nil)
	// No path value set
	w := httptest.NewRecorder()

	h.HandlePortfolio(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing portfolio name")
}

func TestHandlePortfolio_Error(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "bad", "time").Return(types.PortfolioPageData{}, errors.New("not found"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios/bad", nil)
	req.SetPathValue("name", "bad")
	w := httptest.NewRecorder()

	h.HandlePortfolio(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
	ps.AssertExpectations(t)
}

// --- HandleWorkspace ---

func TestHandleWorkspace_Success(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.WorkspaceService)
	h := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "time").Return(types.WorkspacePageData{
		Name: "ws1",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	w := httptest.NewRecorder()

	h.HandleWorkspace(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	wsSvc.AssertExpectations(t)
}

func TestHandleWorkspace_MissingParams(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios//workspaces/", nil)
	w := httptest.NewRecorder()

	h.HandleWorkspace(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "portfolio and workspace name required")
}

func TestHandleWorkspace_Error(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.WorkspaceService)
	h := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "missing", "time").Return(types.WorkspacePageData{}, errors.New("not found"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/missing", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "missing")
	w := httptest.NewRecorder()

	h.HandleWorkspace(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
	wsSvc.AssertExpectations(t)
}

// --- HandleForge ---

func TestHandleForge_Success(t *testing.T) {
	t.Parallel()

	fsSvc := new(mockcontroller.ForgeService)
	h := newTestHandler(nil, nil, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "repo-a").Return(types.ForgePageData{
		RepoName: "repo-a",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1/repos/repo-a", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	req.SetPathValue("r", "repo-a")
	w := httptest.NewRecorder()

	h.HandleForge(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	fsSvc.AssertExpectations(t)
}

func TestHandleForge_MissingParams(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1/repos/", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	// No "r" path value
	w := httptest.NewRecorder()

	h.HandleForge(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "portfolio, workspace and repo name required")
}

func TestHandleForge_Error(t *testing.T) {
	t.Parallel()

	fsSvc := new(mockcontroller.ForgeService)
	h := newTestHandler(nil, nil, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "bad-repo").Return(types.ForgePageData{}, errors.New("no forge.yaml"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1/repos/bad-repo", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	req.SetPathValue("r", "bad-repo")
	w := httptest.NewRecorder()

	h.HandleForge(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "no forge.yaml")
	fsSvc.AssertExpectations(t)
}

// --- HandleRedirect ---

func TestHandleRedirect(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.HandleRedirect(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/portfolios", w.Header().Get("Location"))
}

// --- HandleToggleTheme ---

func TestHandleToggleTheme_LightToDark(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/theme/toggle", nil)
	req.Header.Set("Referer", "/portfolios")
	w := httptest.NewRecorder()

	h.HandleToggleTheme(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/portfolios", w.Header().Get("Location"))
	cookies := w.Result().Cookies()
	var themeCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "theme" {
			themeCookie = c
			break
		}
	}
	assert.NotNil(t, themeCookie)
	assert.Equal(t, "dark", themeCookie.Value)
}

func TestHandleToggleTheme_DarkToLight(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/theme/toggle", nil)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	req.Header.Set("Referer", "/portfolios/p1")
	w := httptest.NewRecorder()

	h.HandleToggleTheme(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	cookies := w.Result().Cookies()
	var themeCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "theme" {
			themeCookie = c
			break
		}
	}
	assert.NotNil(t, themeCookie)
	assert.Equal(t, "light", themeCookie.Value)
}

// --- isDarkMode ---

func TestIsDarkMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cookie   *http.Cookie
		expected bool
	}{
		{"no cookie", nil, false},
		{"light", &http.Cookie{Name: "theme", Value: "light"}, false},
		{"dark", &http.Cookie{Name: "theme", Value: "dark"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			got := isDarkMode(req)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// --- render ---

func TestRender_UnknownTemplate(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	w := httptest.NewRecorder()

	h.render(w, "nonexistent", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "template not found")
}
