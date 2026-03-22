package flight

import (
	"flight-search-service/internal/domain"
	"log"
	"net/http"
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
type Metadata struct {
	TotalResults      int  `json:"total_results"`
	ProviderQueried   int  `json:"providers_queried"`
	ProviderSucceeded int  `json:"providers_succeeded"`
	ProviderFailed    int  `json:"providers_failed"`
	SearchTime        int  `json:"search_time_ms"`
	CacheHit          bool `json:"cache_hit"`
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
	Passengers    int    `json:"passengers"`
	CabinClass    string `json:"cabinClass"`
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

	domReq := domain.SearchRequest{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: depDate,
		ReturnDate:    retDate,
		Passengers:    req.Passengers,
		CabinClass:    req.CabinClass,
	}

	flights, meta := h.service.AggregateSearch(c.Request.Context(), domReq)
	criteria := SearchCriteria{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: req.DepartureDate,
		Passengers:    req.Passengers,
		CabinClass:    req.CabinClass,
	}

	c.JSON(http.StatusOK, &ResultDTO{
		Criteria: criteria,
		Metadata: meta,
		Flights:  flights,
	})
}
