package generator

const modelTpl = `package {{.Snake}}

import (
{{if .NeedsTime}}	"time"
{{end}}	"gorm.io/gorm"
)

type {{.Name}} struct {
	gorm.Model
{{range .Fields}}	{{.GoName}} {{.GoType}} ` + "`json:\"{{.JSONName}}\" gorm:\"{{.GormTag}}\"`" + `
{{end}}{{range .Relations}}{{if eq .Type "belongs-to"}}	{{.ForeignKeyGo}} uint ` + "`json:\"{{.ForeignKey}}\" gorm:\"index\"`" + `
	{{.TargetField}} any ` + "`json:\"{{.TargetSnake}},omitempty\" gorm:\"-\"`" + `
{{else if eq .Type "has-one"}}	{{.TargetField}} any ` + "`json:\"{{.TargetSnake}},omitempty\" gorm:\"-\"`" + `
{{else if eq .Type "has-many"}}	{{.TargetFieldMany}} []any ` + "`json:\"{{.TargetSnake}}s,omitempty\" gorm:\"-\"`" + `
{{else if eq .Type "many-to-many"}}	{{.TargetFieldMany}} []any ` + "`json:\"{{.TargetSnake}}s,omitempty\" gorm:\"-\"`" + `
{{end}}
{{end}}}
`

const dtoTpl = `package {{.Snake}}

{{if .NeedsTime}}
import (
	"time"
)
{{end}}

type Create{{.Name}}Request struct {
{{range .Fields}}	{{.GoName}} {{.GoType}} ` + "`json:\"{{.JSONName}}\" validate:\"{{.ValidateCreate}}\"`" + `
{{end}}{{range .Relations}}{{if eq .Type "belongs-to"}}	{{.ForeignKeyGo}} uint ` + "`json:\"{{.ForeignKey}}\" validate:\"omitempty\"`" + `
{{end}}
{{end}}}

type Update{{.Name}}Request struct {
{{range .Fields}}	{{.GoName}} *{{.GoType}} ` + "`json:\"{{.JSONName}}\" validate:\"{{.ValidateUpdate}}\"`" + `
{{end}}{{range .Relations}}{{if eq .Type "belongs-to"}}	{{.ForeignKeyGo}} *uint ` + "`json:\"{{.ForeignKey}}\" validate:\"omitempty\"`" + `
{{end}}
{{end}}}
`

const repositoryTpl = `package {{.Snake}}

import (
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll(includes []string) ([]{{.Name}}, error)
	FindByID(id uint, includes []string) (*{{.Name}}, error)
	Create(model *{{.Name}}) error
	Update(model *{{.Name}}) error
	Delete(model *{{.Name}}) error
}

type GormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &GormRepository{db: db} }

func (r *GormRepository) FindAll(includes []string) ([]{{.Name}}, error) {
	if err := validateIncludes(includes); err != nil {
		return nil, err
	}
	var rows []{{.Name}}
	return rows, r.db.Find(&rows).Error
}

func (r *GormRepository) FindByID(id uint, includes []string) (*{{.Name}}, error) {
	if err := validateIncludes(includes); err != nil {
		return nil, err
	}
	var row {{.Name}}
	err := r.db.First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *GormRepository) Create(model *{{.Name}}) error { return r.db.Create(model).Error }
func (r *GormRepository) Update(model *{{.Name}}) error { return r.db.Save(model).Error }
func (r *GormRepository) Delete(model *{{.Name}}) error { return r.db.Delete(model).Error }

func validateIncludes(includes []string) error {
	allowed := map[string]struct{}{
{{range .Relations}}{{if .Include}}		"{{.TargetSnake}}": {},
{{end}}{{end}}	}
	for _, include := range includes {
		if _, ok := allowed[include]; !ok {
			return errors.New("invalid include: " + include)
		}
	}
	return nil
}
`

const serviceTpl = `package {{.Snake}}

import "errors"

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(includes []string) ([]{{.Name}}, error) { return s.repo.FindAll(includes) }

func (s *Service) Get(id uint, includes []string) (*{{.Name}}, error) { return s.repo.FindByID(id, includes) }

func (s *Service) Create(req Create{{.Name}}Request) (*{{.Name}}, error) {
	model := &{{.Name}}{
{{range .Fields}}		{{.GoName}}: req.{{.GoName}},
{{end}}{{range .Relations}}{{if eq .Type "belongs-to"}}		{{.ForeignKeyGo}}: req.{{.ForeignKeyGo}},
{{end}}
{{end}}	}
	return model, s.repo.Create(model)
}

func (s *Service) Update(id uint, req Update{{.Name}}Request) (*{{.Name}}, error) {
	model, err := s.repo.FindByID(id, nil)
	if err != nil || model == nil {
		return nil, errors.New("{{.Lower}} not found")
	}
{{range .Fields}}	if req.{{.GoName}} != nil {
		model.{{.GoName}} = *req.{{.GoName}}
	}
{{end}}{{range .Relations}}{{if eq .Type "belongs-to"}}	if req.{{.ForeignKeyGo}} != nil {
		model.{{.ForeignKeyGo}} = *req.{{.ForeignKeyGo}}
	}
{{end}}
{{end}}	return model, s.repo.Update(model)
}

func (s *Service) Delete(id uint) error {
	model, err := s.repo.FindByID(id, nil)
	if err != nil || model == nil {
		return errors.New("{{.Lower}} not found")
	}
	return s.repo.Delete(model)
}
`

const handlerTpl = `package {{.Snake}}

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	api "github.com/raufendro/novacore/internal/http"
	"github.com/raufendro/novacore/pkg/validation"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Index(c *gin.Context) {
	rows, err := h.service.List(parseIncludes(c))
	if err != nil {
		api.Error(c, http.StatusInternalServerError, "Failed to retrieve data", nil)
		return
	}
	api.OK(c, "Data retrieved successfully", rows, nil)
}

func (h *Handler) Show(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	row, err := h.service.Get(id, parseIncludes(c))
	if err != nil || row == nil {
		api.Error(c, http.StatusNotFound, "{{.Name}} not found", nil)
		return
	}
	api.OK(c, "Data retrieved successfully", row, nil)
}

func (h *Handler) Store(c *gin.Context) {
	var req Create{{.Name}}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	if errors := validation.Struct(req); errors != nil {
		api.Error(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	row, err := h.service.Create(req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, "Failed to create data", nil)
		return
	}
	api.Created(c, "Data created successfully", row)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req Update{{.Name}}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	if errors := validation.Struct(req); errors != nil {
		api.Error(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	row, err := h.service.Update(id, req)
	if err != nil {
		api.Error(c, http.StatusNotFound, err.Error(), nil)
		return
	}
	api.OK(c, "Data updated successfully", row, nil)
}

func (h *Handler) Destroy(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		api.Error(c, http.StatusNotFound, err.Error(), nil)
		return
	}
	api.OK(c, "Data deleted successfully", gin.H{"id": id}, nil)
}

{{range .Relations}}{{if .Nested}}
func (h *Handler) {{.TargetFieldMany}}ForParent(c *gin.Context) {
	api.OK(c, "Nested endpoint scaffold", gin.H{
		"parent_id": c.Param("id"),
		"relation": "{{.TargetSnake}}",
	}, nil)
}
{{end}}{{end}}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Error(c, http.StatusBadRequest, "Invalid id", nil)
		return 0, false
	}
	return uint(id), true
}

func parseIncludes(c *gin.Context) []string {
	raw := c.Query("include")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
`

const routesTpl = `package {{.Snake}}

import (
	"github.com/gin-gonic/gin"
{{if not .Public}}
	"github.com/raufendro/novacore/internal/middleware"
{{end}}
	"github.com/raufendro/novacore/pkg/security"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, jwt *security.JWTManager) {
{{if .Public}}
	_ = jwt
{{end}}
	_ = db.AutoMigrate(&{{.Name}}{})
	handler := NewHandler(NewService(NewRepository(db)))
{{if .Public}}
	group := router.Group("/{{.PluralPath}}")
{{else}}
	group := router.Group("/{{.PluralPath}}", middleware.Auth(jwt))
{{end}}
{{if .HasGET}}
	group.GET("", handler.Index)
	group.GET("/:id", handler.Show)
{{end}}
{{if .HasPOST}}
	group.POST("", handler.Store)
{{end}}
{{if .HasPUT}}
	group.PUT("/:id", handler.Update)
{{end}}
{{if .HasPATCH}}
	group.PATCH("/:id", handler.Update)
{{end}}
{{if .HasDELETE}}
	group.DELETE("/:id", handler.Destroy)
{{end}}
{{range .Relations}}{{if .Nested}}
	group.GET("/:id/{{.TargetPluralPath}}", handler.{{.TargetFieldMany}}ForParent)
	group.POST("/:id/{{.TargetPluralPath}}", handler.{{.TargetFieldMany}}ForParent)
{{end}}{{end}}
}
`

const testTpl = `package {{.Snake}}

import (
	"testing"
{{if .NeedsTime}}	"time"
{{end}})

type fakeRepo struct{ rows []{{.Name}} }

func (f *fakeRepo) FindAll(includes []string) ([]{{.Name}}, error) { return f.rows, nil }
func (f *fakeRepo) FindByID(id uint, includes []string) (*{{.Name}}, error) {
	for i := range f.rows {
		if f.rows[i].ID == id {
			return &f.rows[i], nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) Create(model *{{.Name}}) error {
	model.ID = uint(len(f.rows) + 1)
	f.rows = append(f.rows, *model)
	return nil
}
func (f *fakeRepo) Update(model *{{.Name}}) error { return nil }
func (f *fakeRepo) Delete(model *{{.Name}}) error { return nil }

func TestServiceCreate(t *testing.T) {
	service := NewService(&fakeRepo{})
	row, err := service.Create(Create{{.Name}}Request{
{{range .Fields}}		{{.GoName}}: {{.Example}},
{{end}}	})
	if err != nil {
		t.Fatal(err)
	}
{{if .FirstField}}	if row.{{.FirstField.GoName}} == {{.FirstField.ZeroValue}} {
		t.Fatal("expected {{.FirstField.GoName}} to be set")
	}
{{else}}	if row.ID != 1 {
		t.Fatalf("expected generated ID, got %d", row.ID)
	}
{{end}}}
`
