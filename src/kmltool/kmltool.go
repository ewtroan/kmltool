package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const DIR = "/home/ewt/Downloads/legs"

type kmlFolder struct {
	//XMLName xml.Name
	Name       string         `xml:"name"`
	Placemarks []kmlPlacemark `xml:"Placemark"`
	Folders    []kmlFolder    `xml:"Folder,omitempty"`
}

type kmlStyle struct {
	LineStyle *kmlLineStyle `xml:",omitempty"`
	IconStyle *kmlIconStyle `xml:",omitempty"`
}

type kmlIconStyle struct {
	Scale float64  `xml:"scale,omitempty"`
	Icon  *kmlHref `xml:"Icon,omitempty"`
}

type kmlHref struct {
	Href string `xml:"href"`
}

type kmlLineStyle struct {
	Color string  `xml:"color"`
	Width float64 `xml:"width"`
}

type kmlPlacemark struct {
	Name       string          `xml:"name"`
	Style      *kmlStyle       `xml:",omitempty"`
	LineString []kmlLineString `xml:",omitempty"`
	Point      *kmlPoint       `xml:",omitempty"`
}

type kmlPoint struct {
	Coordinates string `xml:"coordinates"`
}

type kmlLineString struct {
	Extrude      bool   `xml:"extrude"`
	Tessellate   bool   `xml:"tessellate"`
	AltitudeMode string `xml:"altitudeMode"`
	Coordinates  string `xml:"coordinates"`
}

type kml struct {
	XMLName xml.Name
	Folder  kmlFolder `xml:"Folder"`
}

func buildKml() kml {
	entries, err := os.ReadDir(DIR)
	if err != nil {
		log.Fatalf("reading directory: %s", err)
	}

	var kmls []kml

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".kml") {
			continue
		}
		path := DIR + "/" + entry.Name()
		r, err := os.Open(path)
		if err != nil {
			log.Fatalf("opening %s: %s", path, err)
		}

		xmlBytes, err := io.ReadAll(r)
		if err != nil {
			log.Fatalf("reading %s: %s", path, err)
		}

		var kml kml
		if err := xml.Unmarshal(xmlBytes, &kml); err != nil {
			log.Fatalf("parsing %s: %s", path, err)
		}

		kmls = append(kmls, kml)
	}

	for i := range kmls {
		for j := range kmls[i].Folder.Placemarks {
			kmls[i].Folder.Placemarks[j].Name = kmls[i].Folder.Name
			if i%2 == 1 {
				kmls[i].Folder.Placemarks[j].Style.LineStyle.Color = "FF00FF00"
			}
		}

		kmls[i].Folder.Folders = []kmlFolder{
			{
				Placemarks: []kmlPlacemark{
					{
						Name: kmls[i].Folder.Name + " Start",
						Point: &kmlPoint{
							Coordinates: strings.Split(kmls[i].Folder.Placemarks[0].LineString[0].Coordinates, " ")[0],
						},
					},
				},
			},
		}

		if i == len(kmls)-1 {
			coordinates := strings.Split(kmls[i].Folder.Placemarks[0].LineString[0].Coordinates, " ")
			kmls[i].Folder.Folders[0].Placemarks = append(kmls[i].Folder.Folders[0].Placemarks, kmlPlacemark{
				Name: "End",
				Point: &kmlPoint{
					Coordinates: coordinates[len(coordinates)-1],
				},
			})
		}
	}

	finalKml := kmls[0]
	finalKml.Folder.Name = "Garmin Tracks"
	finalKml.Folder.Folders[0].Name = "Start/Ends"
	for _, kml := range kmls[1:] {
		finalKml.Folder.Placemarks = append(finalKml.Folder.Placemarks, kml.Folder.Placemarks...)
		finalKml.Folder.Folders[0].Placemarks = append(finalKml.Folder.Folders[0].Placemarks, kml.Folder.Folders[0].Placemarks...)
	}

	return finalKml
}

func main() {
	finalKml := buildKml()

	if s, err := xml.MarshalIndent(finalKml, "", "    "); err != nil {
		log.Fatalf("marshaling: %s", err)
	} else {
		fmt.Printf("%s%s\n", xml.Header, s)
	}
}
