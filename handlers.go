package main

import "C"
import (
	"context"
	"encoding/json"
	"github.com/NYTimes/nyt-explore-articles/cluster"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func GetImages(response http.ResponseWriter, request *http.Request) {
	response.Header().Add("content-type", "application/json")

	images, err :=  articleStore.GetImages(request.Context())
	if err != nil {
 		log.Printf("error getting images from store %v", err)
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte(`{"message": "error getting images from store"}`))
		return
	}

	if len(images) == 0 {
		log.Printf("no images found")
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte(`{"message": "no images found"}`))
		return
	}

	markers := make([]*Marker, len(images))
	for i, v := range images {
		m := Marker(v)
		markers[i] = &m
	}

	groups := groupMarkers(markers)

	groupsToSend := MarkerResponse{
		Groups: groups,
	}

	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(groupsToSend)
}

func GetArticles(response http.ResponseWriter, request *http.Request) {
	response.Header().Add("content-type", "application/json")

	articles, err := articleStore.GetArticles(request.Context())
	if err != nil {
		log.Printf("error getting articles from store %v", err)
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte(`{"message": "error getting articles from store"}`))
		return
	}

	if len(articles) == 0 {
		log.Println("No articles found")
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte(`{"message": "no articles found"}`))
		return
	}

	markers := make([]*Marker, len(articles))
	for i, v := range articles {
		m := Marker(v)
		markers[i] = &m
	}

	groups := groupMarkers(markers)

	groupsToSend := MarkerResponse{
		Groups: groups,
	}

	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(groupsToSend)
}

func CreateArticle(response http.ResponseWriter, request *http.Request) {
	response.Header().Add("content-type", "application/json")
	decoder := json.NewDecoder(request.Body)

	var article = &Article{}
	err := decoder.Decode(article)
	if err != nil {
		log.Println("Error decoding json: ", err)
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write([]byte(`{"message": "invalid article"}`))
		return
	}

	article.CreatedAt = time.Now()

	err = articleStore.SaveArticle(request.Context(), article)
	if err != nil {
		log.Println("Error saving article to db: ", err)
		response.WriteHeader(http.StatusInternalServerError)
		_, _ = response.Write([]byte(`{"message": "error saving article to database"}`))
		return
	}

	cityExists, err := redisClient.HExists("citiesHash", article.GeneralLocation).Result()
	if err != nil {
		log.Println("error getting cityExists: " + article.GeneralLocation, err)
		response.WriteHeader(http.StatusInternalServerError)
		_, _ = response.Write([]byte(`{"message": "error checking if city exists"}`))
		return
	}

	if !cityExists {
		_, err := redisClient.HSet("citiesHash", article.GeneralLocation, 1).Result()
		if err != nil {
			log.Println("error setting city: " + article.GeneralLocation, err)
			response.WriteHeader(http.StatusInternalServerError)
			_, _ = response.Write([]byte(`{"message": "error setting city to 1"}`))
			return
		}
	} else {
		_, err := redisClient.HIncrBy("citiesHash", article.GeneralLocation, 1).Result()
		if err != nil {
			log.Println("error incrementing city: " + article.GeneralLocation, err)
			response.WriteHeader(http.StatusInternalServerError)
			_, _ = response.Write([]byte(`{"message": "error incrementing city"}`))
			return
		}
	}

	num, err := redisClient.HGet("citiesHash", article.GeneralLocation).Result()
	if err != nil {
		log.Println("error getting city num: " + article.GeneralLocation, err)
		response.WriteHeader(http.StatusInternalServerError)
		_, _ = response.Write([]byte(`{"message": "error getting number from city"}`))
		return
	}

	numAsInt, err := strconv.Atoi(num)
	if err != nil {
		log.Println("error incrementing city: " + article.GeneralLocation, err)
		response.WriteHeader(http.StatusInternalServerError)
		_, _ = response.Write([]byte(`{"message": "error converting number to int"}`))
		return
	}

	if numAsInt % 3 == 0 {
		go sendEmailNotification(article.GeneralLocation)
	}

	response.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(response).Encode(article)
}

func sendEmailNotification(generalLocation string) {
	// these subscriptions need to move to a config file or something
	// but this works for now
	emails := map[string][]string{
		"Null Island": {"al.madireddy@nytimes.com"},
		"New York City": {"nnamdi.ojibe@nytimes.com", "olivia.sturman@nytimes.com"},
	}

	sendGridKey := os.Getenv("SENDGRID_API_KEY")
	if sendGridKey == "" {
		log.Println("no sendgrid key in email handler")
		return
	}

	articles, err := articleStore.GetArticlesByLocation(context.Background(), generalLocation)
	if err != nil {
		log.Println("Error getting articles by location", err)
		return
	}

	articles = articles[:3]
	messageBody := "<h2>Here are new locations in " + generalLocation + " for you to check out</h2> <br><br>"

	for _, i := range articles {
		messageBody += `<a href="` + i.URL + `">` + i.Headline + `</a> <br>`
		messageBody += i.FirstParagraph + "<br><br>"
	}
	for _, i := range emails[generalLocation] {
		from := mail.NewEmail("NYT Explore", "test@nytimes.com")
		subject := "New Articles in " + generalLocation
		to := mail.NewEmail(i, i)
		plainTextContent := messageBody
		htmlContent := messageBody
		message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

		client := sendgrid.NewSendClient(sendGridKey)
		response, err := client.Send(message)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("Email response status code: ", response.StatusCode, response.Body)
		}
	}
}

func groupMarkers(ungrouped []*Marker) []*MarkerGroup {
	// DBSCAN grouping criteria
	eps := float64(.35) // n kilometer clustering radius
	minPoints := 2 // x points minimum in eps-neighborhood
	log.Printf("eps: %v, minPoints: %v", eps, minPoints) //TODO: Delete
	// Declare list of points for DBSCAN algorithm
	pointList := make(cluster.PointList, len(ungrouped));

	// Make map of article URLs to boolean values (when visited, set to true)
	visited := make(map[string]bool)

	// Create a point in PointList for each article
	for i, a := range ungrouped {
		article := *a
		if article.GetCoordinates().Lng == 0 && article.GetCoordinates().Lat == 0 {
			continue  // Move on if article has null coords
		}

		// Points are (Longitude, Latitude)
		pointList[i][0] = article.GetCoordinates().Lng
		pointList[i][1] = article.GetCoordinates().Lat

		visited[article.GetURL()] = false	// Set article visited status to false
	}

	// Run DBSCAN algorithm
	clusters, noise := cluster.DBScan(pointList, eps, minPoints) // noise = list of point indexes which don't fit into any cluster

	// Declare slice of article groups
	var groups []*MarkerGroup

	// Make map of clusters to group
	clusterToGroup := make(map[int]*MarkerGroup)
	// 1. Assign each cluster centroid to a group
	for _, c := range clusters {
		centroid, _, _  := c.CentroidAndBounds(pointList)

		groups = append(groups, &MarkerGroup{
			Markers: []*Marker{},
			Coordinates: Coordinates{
				Lat: centroid[1],	// Points are (Longitude, Latitude)
				Lng: centroid[0],
			},
			NumberItems: 0,
		})
		log.Printf(c.String())
		clusterToGroup[c.C] = groups[len(groups)-1]	// Map the cluster to the just appended group
	}

	// 2. Assign each point of noise to a group
	for _, n := range noise {
		groups = append(groups, &MarkerGroup{
			Markers: []*Marker{},
			Coordinates: Coordinates{
				Lat: pointList[n][1], // Points are (Longitude, Latitude)
				Lng: pointList[n][0],
			},
			NumberItems: 0,
		})
	}

	// 3. If article is in cluster assign to that group
	for _, a := range ungrouped {
		article := *a

		if (article.GetCoordinates().Lng == 0 && article.GetCoordinates().Lat == 0) || visited[article.GetURL()] {
			continue  // Move on if article has null coords
		}

		for _, c := range clusters {
			for _, point := range c.Points {
				if (article.GetCoordinates().Lng == pointList[point][0] && article.GetCoordinates().Lat == pointList[point][1]) && !visited[article.GetURL()] {
					visited[article.GetURL()] = true
					clusterToGroup[c.C].Markers = append(clusterToGroup[c.C].Markers, &article)
					clusterToGroup[c.C].NumberItems = clusterToGroup[c.C].NumberItems + 1
				}
			}
		}
	}

	// 4. If article is NOT in cluster assign to appropriate group
	for _, a := range ungrouped {
		article := *a

		if (article.GetCoordinates().Lng == 0 && article.GetCoordinates().Lat == 0) || visited[article.GetURL()] {
			continue  // Move on if article has null coords or has already been visited (if it were in a cluster)
		}

		for _, group := range groups {
			if coordinatesMatch(article.GetCoordinates(), group.Coordinates) {
				visited[article.GetURL()] = true
				group.Markers = append(group.Markers, &article)
				group.NumberItems = group.NumberItems + 1
			}
		}

	}

	return groups
}
