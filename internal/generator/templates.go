package generator

const modelTpl = `package {{.Snake}}

import (
{{if .NeedsTime}}	"time"
{{end}}	"gorm.io/gorm"
)

type {{.Name}} struct {
	gorm.Model
{{range .Fields}}	{{.GoName}} {{.GoType}} ` + "`json:\"{{.JSONName}}\" gorm:\"{{.GormTag}}\"`" + `
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
{{end}}}

type Update{{.Name}}Request struct {
{{range .Fields}}	{{.GoName}} *{{.GoType}} ` + "`json:\"{{.JSONName}}\" validate:\"{{.ValidateUpdate}}\"`" + `
{{end}}}
`

const repositoryTpl = `package {{.Snake}}

import (
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll() ([]{{.Name}}, error)
	FindByID(id uint) (*{{.Name}}, error)
	Create(model *{{.Name}}) error
	Update(model *{{.Name}}) error
	Delete(model *{{.Name}}) error
}

type GormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &GormRepository{db: db} }

func (r *GormRepository) FindAll() ([]{{.Name}}, error) {
	var rows []{{.Name}}
	return rows, r.db.Find(&rows).Error
}

func (r *GormRepository) FindByID(id uint) (*{{.Name}}, error) {
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
`

const serviceTpl = `package {{.Snake}}

import "errors"

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List() ([]{{.Name}}, error) { return s.repo.FindAll() }

func (s *Service) Get(id uint) (*{{.Name}}, error) { return s.repo.FindByID(id) }

func (s *Service) Create(req Create{{.Name}}Request) (*{{.Name}}, error) {
	model := &{{.Name}}{
{{range .Fields}}		{{.GoName}}: req.{{.GoName}},
{{end}}	}
	return model, s.repo.Create(model)
}

func (s *Service) Update(id uint, req Update{{.Name}}Request) (*{{.Name}}, error) {
	model, err := s.repo.FindByID(id)
	if err != nil || model == nil {
		return nil, errors.New("{{.Lower}} not found")
	}
{{range .Fields}}	if req.{{.GoName}} != nil {
		model.{{.GoName}} = *req.{{.GoName}}
	}
{{end}}	return model, s.repo.Update(model)
}

func (s *Service) Delete(id uint) error {
	model, err := s.repo.FindByID(id)
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

	"github.com/gin-gonic/gin"
	api "github.com/raufendro/novacore/internal/http"
	"github.com/raufendro/novacore/pkg/validation"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Index(c *gin.Context) {
	rows, err := h.service.List()
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
	row, err := h.service.Get(id)
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

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Error(c, http.StatusBadRequest, "Invalid id", nil)
		return 0, false
	}
	return uint(id), true
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
	group.GET("", handler.Index)
	group.GET("/:id", handler.Show)
	group.POST("", handler.Store)
	group.PUT("/:id", handler.Update)
	group.PATCH("/:id", handler.Update)
	group.DELETE("/:id", handler.Destroy)
}
`

const testTpl = `package {{.Snake}}

import (
	"testing"
{{if .NeedsTime}}	"time"
{{end}})

type fakeRepo struct{ rows []{{.Name}} }

func (f *fakeRepo) FindAll() ([]{{.Name}}, error) { return f.rows, nil }
func (f *fakeRepo) FindByID(id uint) (*{{.Name}}, error) {
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
