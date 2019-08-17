package main

import "time"

type Marker interface {
	Type() string
	GetCoordinates() Coordinates
	GetURL() string
}

//Articles from database
type Article struct {
	Byline string				`json:"byLine" firestore:"byLine"`
	Coordinates Coordinates		`json:"coordinates" firestore:"coordinates"`
	FirstParagraph string		`json:"firstParagraph" firestore:"firstParagraph"`
	GeneralLocation string		`json:"generalLocation" firestore:"generalLocation"`
	Headline string				`json:"headline" firestore:"headline"`
	ImageSmallURL string		`json:"imageSmall" firestore:"imageSmall"`
	PublishDate string			`json:"pubDate" firestore:"pubDate"`
	Snippet string				`json:"snippet" firestore:"snippet"`
	URL string					`json:"url" firestore:"url"`
	CreatedAt time.Time			`json:"createdAt,omitempty" firestore:"createdAt,omitempty"`

}

func (a *Article) Type() string {
	return "article"
}

func (a *Article) GetCoordinates() Coordinates {
	return a.Coordinates
}

func (a *Article) GetURL() string {
	return a.URL
}

//Images from database
type Image struct {
	Byline      string      `json:"byLine,omitempty" firestore:"byLine"`
	BylineTitle string      `json:"byLineTitle,omitempty" firestore:"byLineTitle"`
	Credit      string      `json:"credit" firestore:"credit"`
	Caption     string      `json:"caption" firestore:"caption"`
	DateTaken   time.Time   `json:"dateTaken" firestore:"dateTaken"`
	City        string      `json:"city,omitempty" firestore:"city"`
	State       string      `json:"state,omitempty" firestore:"state"`
	Country     string      `json:"country" firestore:"country"`
	URL         string      `json:"url" firestore:"url"`
	Coordinates Coordinates `json:"coordinates" firestore:"coordinates"`
}

func (i *Image) Type() string {
	return "image"
}

func (i *Image) GetCoordinates() Coordinates {
	return i.Coordinates
}

func (i *Image) GetURL() string {
	return i.URL
}

type Coordinates struct {
	Lat float64 `json:"lat" firestore:"lat"`
	Lng float64 `json:"lng" firestore:"lng"`
}

type MarkerGroup struct {
	Markers     []*Marker   `json:"markers"`
	Coordinates Coordinates `json:"coordinates"`
	NumberItems int         `json:"numberItems"`
}

type MarkerResponse struct {
	Groups []*MarkerGroup `json:"groups"`
}

func coordinatesMatch(coo1 Coordinates, coo2 Coordinates) bool {
	if coo2.Lat == coo1.Lat && coo2.Lng == coo1.Lng {
		return true
	}

	return false
}
