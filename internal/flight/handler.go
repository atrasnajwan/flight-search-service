package flight

import (
	"flight-search-service/internal/domain"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type FlightHandler struct {
	service *FlightService
}

func NewHandler(s *FlightService) *FlightHandler {
	return &FlightHandler{
		service: s,
	}
}

type SearchCriteria struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    int    `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

type ResultDTO struct {
	Criteria SearchCriteria  `json:"search_criteria"`
	Metadata Metadata        `json:"metadata"`
	Flights  []domain.Flight `json:"flights"`
}

type SearchRequestBody struct {
	Origin        string `json:"origin" binding:"required"`
	Destination   string `json:"destination" binding:"required"`
	DepartureDate string `json:"departureDate" binding:"required"`
	ReturnDate    string `json:"returnDate"`
	Passengers    int    `json:"passengers" binding:"required,min=1"`
	CabinClass    string `json:"cabinClass"`
	SortBy        string `json:"sortBy"`
	SortOrder     string `json:"sortOrder"`
}

func (h *FlightHandler) Search(c *gin.Context) {
	var req SearchRequestBody
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Printf("failed to parse body: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to parse request body"})
		return
	}

	depDate, err := time.Parse("2006-01-02", req.DepartureDate)
	if err != nil {
		log.Printf("failed to parse departure date %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to parse departure date"})
		return
	}

	var retDate time.Time
	if req.ReturnDate != "" {
		retDate, err = time.Parse("2006-01-02", req.ReturnDate)
		if err != nil {
			log.Printf("failed to parse return date %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to parse return date"})
			return
		}
	}

	if req.Origin == req.Destination {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "origin and destination cannot be the same"})
		return
	}

	if req.Passengers <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "passengers must be greater than 0"})
		return
	}

	if req.ReturnDate != "" && retDate.Before(depDate) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "return date must be after departure date"})
		return
	}

	sortBy := req.SortBy
	if q := c.Query("sortBy"); q != "" {
		sortBy = q
	}

	sortOrder := req.SortOrder
	if q := c.Query("sortOrder"); q != "" {
		sortOrder = q
	}

	// Get filter params from URL query
	var priceMin, priceMax float64
	var maxStops int
	var durationMin, durationMax int
	var airlines []string
	var departureTimeMin, departureTimeMax, arrivalTimeMin, arrivalTimeMax string

	if q := c.Query("priceMin"); q != "" {
		if v, err := strconv.ParseFloat(q, 64); err == nil {
			priceMin = v
		}
	}
	if q := c.Query("priceMax"); q != "" {
		if v, err := strconv.ParseFloat(q, 64); err == nil {
			priceMax = v
		}
	}
	if q := c.Query("maxStops"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			maxStops = v
		}
	}
	if q := c.Query("durationMin"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			durationMin = v
		}
	}
	if q := c.Query("durationMax"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			durationMax = v
		}
	}
	if q := c.Query("airlines"); q != "" {
		airlines = strings.Split(q, ",")
		for i, a := range airlines {
			airlines[i] = strings.TrimSpace(a)
		}
	}
	departureTimeMin = c.Query("departureTimeMin")
	departureTimeMax = c.Query("departureTimeMax")
	arrivalTimeMin = c.Query("arrivalTimeMin")
	arrivalTimeMax = c.Query("arrivalTimeMax")

	domReq := domain.SearchRequest{
		Origin:           req.Origin,
		Destination:      req.Destination,
		DepartureDate:    depDate,
		ReturnDate:       retDate,
		Passengers:       req.Passengers,
		CabinClass:       req.CabinClass,
		SortBy:           sortBy,
		SortOrder:        sortOrder,
		PriceMin:         priceMin,
		PriceMax:         priceMax,
		MaxStops:         maxStops,
		DepartureTimeMin: departureTimeMin,
		DepartureTimeMax: departureTimeMax,
		ArrivalTimeMin:   arrivalTimeMin,
		ArrivalTimeMax:   arrivalTimeMax,
		Airlines:         airlines,
		DurationMin:      durationMin,
		DurationMax:      durationMax,
	}

	result, err := h.service.AggregateSearch(c.Request.Context(), domReq)
	if err != nil {
		log.Printf("failed get flights %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to get flights"})
		return
	}
	criteria := SearchCriteria{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: req.DepartureDate,
		Passengers:    req.Passengers,
		CabinClass:    req.CabinClass,
	}

	c.JSON(http.StatusOK, &ResultDTO{
		Criteria: criteria,
		Metadata: result.Meta,
		Flights:  result.Flights,
	})
}
