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
	Criteria   SearchCriteria     `json:"search_criteria"`
	Metadata   Metadata           `json:"metadata"`
	Flights    []domain.Flight    `json:"flights"`
	RoundTrips []domain.RoundTrip `json:"roundtrips"`
}

type SearchRequestBody struct {
	Origin        string `json:"origin" binding:"required"`
	Destination   string `json:"destination" binding:"required"`
	DepartureDate string `json:"departureDate" binding:"required"`
	ReturnDate    string `json:"returnDate"`
	Passengers    int    `json:"passengers" binding:"required,min=1"`
	CabinClass    string `json:"cabinClass"`
}

type queryFilters struct {
	priceMin, priceMax                 float64
	maxStops, durationMin, durationMax int
	airlines                           []string
	departureTimeMin, departureTimeMax string
	arrivalTimeMin, arrivalTimeMax     string
}

func parseQueryFilters(c *gin.Context) queryFilters {
	var filters queryFilters

	if q := c.Query("priceMin"); q != "" {
		if v, err := strconv.ParseFloat(q, 64); err == nil {
			filters.priceMin = v
		}
	}
	if q := c.Query("priceMax"); q != "" {
		if v, err := strconv.ParseFloat(q, 64); err == nil {
			filters.priceMax = v
		}
	}
	if q := c.Query("maxStops"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			filters.maxStops = v
		}
	}
	if q := c.Query("durationMin"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			filters.durationMin = v
		}
	}
	if q := c.Query("durationMax"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			filters.durationMax = v
		}
	}
	if q := c.Query("airlines"); q != "" {
		filters.airlines = strings.Split(q, ",")
		for i, a := range filters.airlines {
			filters.airlines[i] = strings.TrimSpace(a)
		}
	}

	filters.departureTimeMin = c.Query("departureTimeMin")
	filters.departureTimeMax = c.Query("departureTimeMax")
	filters.arrivalTimeMin = c.Query("arrivalTimeMin")
	filters.arrivalTimeMax = c.Query("arrivalTimeMax")

	return filters
}

func buildDomainRequest(
	origin string,
	destination string,
	departureDate string,
	returnDate *string,
	passengers int,
	cabinClass string,
	sortBy string,
	sortOrder string,
	filters queryFilters,
) (domain.SearchRequest, error) {
	if origin == destination {
		return domain.SearchRequest{}, errBadRequest{"origin and destination cannot be the same"}
	}
	if passengers <= 0 {
		return domain.SearchRequest{}, errBadRequest{"passengers must be greater than 0"}
	}

	depDate, err := time.Parse("2006-01-02", departureDate)
	if err != nil {
		return domain.SearchRequest{}, errBadRequest{"failed to parse departure date"}
	}

	var retDate time.Time
	if returnDate != nil && *returnDate != "" {
		retDate, err = time.Parse("2006-01-02", *returnDate)
		if err != nil {
			return domain.SearchRequest{}, errBadRequest{"failed to parse return date"}
		}
		if !retDate.After(depDate) {
			return domain.SearchRequest{}, errBadRequest{"return date must be after departure date"}
		}
	}

	return domain.SearchRequest{
		Origin:           origin,
		Destination:      destination,
		DepartureDate:    depDate,
		ReturnDate:       retDate,
		Passengers:       passengers,
		CabinClass:       cabinClass,
		SortBy:           sortBy,
		SortOrder:        sortOrder,
		PriceMin:         filters.priceMin,
		PriceMax:         filters.priceMax,
		MaxStops:         filters.maxStops,
		DepartureTimeMin: filters.departureTimeMin,
		DepartureTimeMax: filters.departureTimeMax,
		ArrivalTimeMin:   filters.arrivalTimeMin,
		ArrivalTimeMax:   filters.arrivalTimeMax,
		Airlines:         filters.airlines,
		DurationMin:      filters.durationMin,
		DurationMax:      filters.durationMax,
	}, nil
}

type errBadRequest struct {
	msg string
}

func (e errBadRequest) Error() string {
	return e.msg
}

func (h *FlightHandler) Search(c *gin.Context) {
	var req SearchRequestBody
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Printf("failed to parse body: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to parse request body"})
		return
	}

	sortBy := ""
	if q := c.Query("sortBy"); q != "" {
		sortBy = q
	}

	sortOrder := ""
	if q := c.Query("sortOrder"); q != "" {
		sortOrder = q
	}

	filters := parseQueryFilters(c)
	returnDate := req.ReturnDate
	domReq, err := buildDomainRequest(
		req.Origin,
		req.Destination,
		req.DepartureDate,
		&returnDate,
		req.Passengers,
		req.CabinClass,
		sortBy,
		sortOrder,
		filters,
	)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
		Criteria:   criteria,
		Metadata:   result.Meta,
		Flights:    result.Flights,
		RoundTrips: result.RoundTrips,
	})
}
