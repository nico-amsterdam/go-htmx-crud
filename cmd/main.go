package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Request headers: https://htmx.org/docs/#request-headers
const (
	HeaderBoosted               = "HX-Boosted"
	HeaderCurrentUrl            = "HX-Current-URL"
	HeaderHistoryRestoreRequest = "HX-History-Restore-Request"
	HeaderPrompt                = "HX-Prompt"
	HeaderRequest               = "HX-Request"
	HeaderTarget                = "HX-Target"
	HeaderTriggerName           = "HX-Trigger-Name"
	HeaderTrigger               = "HX-Trigger"
)

// Response headers: https://htmx.org/docs/#response-headers
const (
	HeaderLocation   = "HX-Location"
	HeaderPushURL    = "HX-Push-Url"
	HeaderRedirect   = "HX-Redirect"
	HeaderRefresh    = "HX-Refresh"
	HeaderReplaceURL = "HX-Replace-Url"
	HeaderReswap     = "HX-Reswap"
	HeaderRetarget   = "HX-Retarget"
	HeaderReselect   = "HX-Reselect"
	//	HeaderTrigger            = "HX-Trigger"  already defined
	HeaderTriggerAfterSettle = "HX-Trigger-After-Settle"
	HeaderTriggerAfterSwap   = "HX-Trigger-After-Swap"
)

const productSearchCookie = "product-search"

func htmxEnabled(c echo.Context) bool {
	return c.Request().Header.Get(HeaderRequest) == "true"
}

type Templates struct {
	templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() *Templates {
	return &Templates{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
}

var id = 0

type Product struct {
	Name  string
	Descr string
	Price int
	Id    int
}

func (p Product) EuroPrice() float64 {
	return float64(p.Price) * float64(0.01)
}

func newProduct(name, descr string, price int) Product {
	id++
	return Product{
		Name:  name,
		Descr: descr,
		Price: price, // in cents
		Id:    id,
	}
}

type Products = []Product

type Data struct {
	Products Products
}

func (d *Data) indexOf(id int) int {
	for i, product := range d.Products {
		if product.Id == id {
			return i
		}
	}
	return -1
}

func (d *Data) hasName(name string) bool {
	for _, product := range d.Products {
		if product.Name == name {
			return true
		}
	}
	return false
}

func newData() Data {
	return Data{
		Products: []Product{
			newProduct("Hammer", "Smashing hammer", 1000),
		},
	}
}

type FormData struct {
	Values map[string]string
	Errors map[string]string
}

func newFormData() FormData {
	return FormData{
		Values: make(map[string]string),
		Errors: make(map[string]string),
	}
}

type Page struct {
	Data             Data
	Form             FormData
	SearchText       string
	FilteredProducts Products
}

func caseInsensitiveContains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

func (page *Page) filteredProducts() Products {
	if page.SearchText == "" {
		return page.Data.Products
	}
	var result Products = []Product{}
	for _, product := range page.Data.Products {
		if caseInsensitiveContains(product.Name, page.SearchText) || caseInsensitiveContains(product.Descr, page.SearchText) {
			result = append(result, product)
		}
	}
	return result
}

func newPage() Page {
	data := newData()
	return Page{
		Data:             data,
		Form:             newFormData(),
		SearchText:       "",
		FilteredProducts: data.Products,
	}
}

func validateProductID(idStr string, page Page) (int, int, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, http.StatusBadRequest, fmt.Errorf("Invalid id")
	}

	index := page.Data.indexOf(id)
	if index == -1 {
		return -1, http.StatusNotFound, fmt.Errorf("Product not found")
	}

	return index, http.StatusOK, nil
}

func validateProductForm(name, descr, price, idStr string, checkName bool, page *Page) (float64, bool) {
	formData := newFormData()
	formData.Values["name"] = name
	formData.Values["descr"] = descr
	formData.Values["price"] = price
	if idStr != "" {
		formData.Values["id"] = idStr
	}

	validName := true
	if checkName {
		validName = !page.Data.hasName(name)
	}

	f64Price, err := strconv.ParseFloat(price, 64)
	validPrice := err == nil

	if !validPrice {
		formData.Errors["price"] = "Invalid price"
	}
	if !validName {
		formData.Errors["name"] = "Name already exists"
	}

	isValid := validPrice && validName
	if !isValid {
		page.Form = formData
	}
	return f64Price, isValid
}

func renderProductList(c echo.Context, page Page) error {
	c.Response().Header().Set(HeaderReplaceURL, "/product-list")
	c.Response().Header().Set(HeaderRetarget, "#main")
	c.Response().Header().Set(HeaderReswap, "outerHTML")
	return c.Render(200, "index_main", page)
}

func main() {

	e := echo.New()
	e.Use(middleware.Logger())

	page := newPage()
	e.Renderer = newTemplate()

	e.Static("/images", "images")
	e.Static("/css", "css")

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/product-list")
	})

	e.GET("/product-list", func(c echo.Context) error {
		cookie, err := c.Cookie(productSearchCookie)
		if err != nil {
			page.SearchText = ""
		} else {
			page.SearchText = cookie.Value
		}
		index_page := "index"
		if htmxEnabled(c) {
			index_page = "index_main"
		}
		return c.Render(200, index_page, page)
	})

	e.GET("/add-product", func(c echo.Context) error {
		if !htmxEnabled(c) {
			return c.Redirect(http.StatusTemporaryRedirect, "/product-list")
		}
		page.Form = newFormData()
		return c.Render(200, "add-product", page)
	})

	e.GET("/product/:id/delete", func(c echo.Context) error {
		if !htmxEnabled(c) {
			return c.Redirect(http.StatusTemporaryRedirect, "/product-list")
		}
		idStr := c.Param("id")
		index, status, err := validateProductID(idStr, page)
		if err != nil {
			return c.String(status, err.Error())
		}
		product := page.Data.Products[index]

		page.Form = newFormData()
		page.Form.Values["id"] = idStr
		page.Form.Values["descr"] = product.Descr
		page.Form.Values["name"] = product.Name

		return c.Render(200, "del-product", page)
	})

	e.GET("/product/:id/edit", func(c echo.Context) error {
		if !htmxEnabled(c) {
			return c.Redirect(http.StatusTemporaryRedirect, "/product-list")
		}
		idStr := c.Param("id")
		index, status, err := validateProductID(idStr, page)
		if err != nil {
			return c.String(status, err.Error())
		}
		page.Form = newFormData()
		product := page.Data.Products[index]
		price := strconv.Itoa(product.Price / 100)

		page.Form.Values["id"] = idStr
		page.Form.Values["price"] = price
		page.Form.Values["descr"] = product.Descr
		page.Form.Values["name"] = product.Name
		return c.Render(200, "edit-product", page)
	})

	e.POST("/product-list/search", func(c echo.Context) error {
		page.SearchText = strings.TrimRight(c.FormValue("search"), "\t\n\r ")
		page.FilteredProducts = page.filteredProducts()
		cookie := new(http.Cookie)
		cookie.Name = productSearchCookie
		cookie.Value = page.SearchText
		c.SetCookie(cookie)
		return c.Render(200, "product-list-search-results", page)
	})

	e.POST("/add-product", func(c echo.Context) error {
		name := strings.TrimSpace(c.FormValue("name"))
		descr := strings.TrimRight(c.FormValue("descr"), "\t\n\r ")
		price := c.FormValue("price")

		f64Price, isValid := validateProductForm(name, descr, price, "", true, &page)
		if !isValid {
			return c.Render(422, "add-product-form", page)
		}

		priceInCents := int(f64Price * 100)
		product := newProduct(name, descr, priceInCents)
		page.Data.Products = append(page.Data.Products, product)
		page.FilteredProducts = page.filteredProducts()

		return renderProductList(c, page)
	})

	e.POST("/product/:id/edit", func(c echo.Context) error {
		idStr := c.Param("id")
		index, status, err := validateProductID(idStr, page)
		if err != nil {
			return c.String(status, err.Error())
		}
		product := &page.Data.Products[index]

		name := strings.TrimSpace(c.FormValue("name"))
		descr := strings.TrimRight(c.FormValue("descr"), "\t\n\r ")
		price := c.FormValue("price")

		checkName := product.Name != name
		f64Price, isValid := validateProductForm(name, descr, price, idStr, checkName, &page)
		if !isValid {
			return c.Render(422, "edit-product-form", page)
		}

		priceInCents := int(f64Price * 100)
		product.Name = name
		product.Descr = descr
		product.Price = priceInCents

		page.FilteredProducts = page.filteredProducts()

		return renderProductList(c, page)
	})

	e.POST("/product/:id/delete", func(c echo.Context) error {
		idStr := c.Param("id")
		index, status, err := validateProductID(idStr, page)
		if err != nil {
			return c.String(status, err.Error())
		}
		page.Data.Products = append(page.Data.Products[:index], page.Data.Products[index+1:]...)

		c.Response().Header().Set(HeaderReplaceURL, "/product-list")

		return c.Render(200, "index_main", page)
	})

	e.Logger.Fatal(e.Start(":8778"))

}
