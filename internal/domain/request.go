package domain

import "time"

type SearchRequest struct {
    Origin         string    `json:"origin"`          
    Destination    string    `json:"destination"`     
    DepartureDate  time.Time `json:"departureDate"`   
    ReturnDate     time.Time `json:"returnDate"`     
    Passengers     int       `json:"passengers"`      
    CabinClass     string    `json:"cabinClass"`      
    SortBy         string    `json:"sortBy,omitempty"`
    SortOrder      string    `json:"sortOrder,omitempty"`
}