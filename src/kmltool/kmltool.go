package main

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

const kmlDir = "/home/ewt/Downloads/legs"
const imageDir = "/home/ewt/Downloads/images"

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
	Name        string          `xml:"name,omitempty"`
	Description string          `xml:"description,omitempty"`
	Style       *kmlStyle       `xml:",omitempty"`
	LineString  []kmlLineString `xml:",omitempty"`
	Point       *kmlPoint       `xml:",omitempty"`
}

type kmlPoint struct {
	latitude  float64
	longitude float64
	altitude  float64
}

func (k *kmlPoint) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var p struct {
		C string `xml:"coordinates"`
	}

	p.C = fmt.Sprintf("%f,%f", k.longitude, k.latitude)
	if k.altitude != 0 {
		p.C = fmt.Sprintf("%s,%f", p.C, k.altitude)
	}

	return e.EncodeElement(p, start)
}

func (k *kmlPoint) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var p struct {
		C string `xml:"coordinates"`
	}

	err := d.DecodeElement(&p, &start)
	if err != nil {
		return err
	}

	parts := strings.Split(p.C, ",")
	k.longitude, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("parsing longitude %s (%s): %s", parts[0], parts, err)
	}

	k.latitude, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return fmt.Errorf("parsing latitude %s (%s): %s", parts[1], parts, err)
	}

	if len(parts) == 3 {
		k.altitude, err = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return fmt.Errorf("parsing altitudfe %s (%s): %s", parts[2], parts, err)
		}
	}

	return nil
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

type imageInfo struct {
	File                string
	Latitude, Longitude float64
	Timestamp           time.Time
	//Bytes               []byte
	Config image.Config
}

type imageSet []imageInfo

func buildKml(dir string) [][]kmlPoint {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("reading directory %s: %s", dir, err)
	}

	var kmls []kml
	var tracks [][]kmlPoint

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".kml") {
			continue
		}
		path := dir + "/" + entry.Name()
		r, err := os.Open(path)
		if err != nil {
			log.Fatalf("opening %s: %s", path, err)
		}

		xmlBytes, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			log.Fatalf("reading %s: %s", path, err)
		}

		var kml kml
		if err := xml.Unmarshal(xmlBytes, &kml); err != nil {
			log.Fatalf("parsing %s: %s", path, err)
		}

		var leg []kmlPoint
		for _, folder := range kml.Folder.Folders {
			if folder.Name != "Track Points" {
				continue
			}

			for _, placemark := range folder.Placemarks {
				leg = append(leg, *placemark.Point)
			}
		}

		kmls = append(kmls, kml)
		tracks = append(tracks, leg)
	}

	/*
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
	*/

	return tracks
}

func loadImages(dir string) imageSet {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("reading directory %s: %s", dir, err)
	}

	images := make([]imageInfo, 0, len(entries))

	for _, entry := range entries {
		name := entry.Name()
		if name[0:1] == "." {
			continue
		} else if !strings.HasSuffix(name, ".jpg") {
			continue
		}

		path := dir + "/" + name
		rawExif, err := exif.SearchFileAndExtractExif(path)
		if err != nil {
			log.Fatalf("reading exif from %s: %s", path, err)
		}

		im, _ := exifcommon.NewIfdMappingWithStandard()
		ti := exif.NewTagIndex()

		_, index, err := exif.Collect(im, ti, rawExif)
		if err != nil {
			log.Fatalf("collecing exif from %s: %s", path, err)
		}

		ifd, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdGpsInfoStandardIfdIdentity)
		if err != nil {
			log.Fatalf("finding identifb from %s: %s", path, err)
		}

		gi, err := ifd.GpsInfo()
		if err != nil {
			log.Fatalf("findind gps location from %s: %s", path, err)
		}

		r, err := os.Open(dir + "/scaled/" + name)
		if err != nil {
			log.Fatalf("failed to open %s: %s", path, err)
		}

		/*
			imageBytes, err := ioutil.ReadAll(r)
			r.Close()
			if err != nil {
				log.Fatalf("failed to read %s: %s", path, err)
			}
		*/

		config, err := jpeg.DecodeConfig(r)
		if err != nil {
			log.Fatalf("failed to read %s: %s", path, err)
		}

		images = append(images, imageInfo{
			File:      name,
			Latitude:  gi.Latitude.Decimal(),
			Longitude: gi.Longitude.Decimal(),
			Timestamp: gi.Timestamp,
			Config:    config,
			//Bytes:     imageBytes,
		})
	}

	return images
}

func (images imageSet) folder() kmlFolder {
	f := kmlFolder{Name: "Photographs"}
	f.Placemarks = make([]kmlPlacemark, 0, len(images))

	icon := &kmlStyle{
		IconStyle: &kmlIconStyle{
			Icon: &kmlHref{
				Href: "http://maps.google.com/mapfiles/kml/shapes/camera.png",
			},
		},
	}

	tz, _ := time.LoadLocation("Europe/Madrid")

	for _, i := range images {
		url := fmt.Sprintf("https://storage.googleapis.com/oot-photos-public/Camino-2024/%s", i.File)
		scaledUrl := fmt.Sprintf("https://storage.googleapis.com/oot-photos-public/Camino-2024/scaled/%s", i.File)
		description := fmt.Sprintf(`<p><a href="%s"><img src="%s" width="%d" height="%d"></a></p><p>%s</p>`, url, scaledUrl, i.Config.Width, i.Config.Height, i.Timestamp.In(tz).Format("Mon, 02 Jan 2006 15:04:05"))
		//description := fmt.Sprintf(`<p><a href="%s"><img src="data:image/jpeg;data:image/jpeg;base64,%s"></a></p><p>%s</p>`, url, base64.StdEncoding.EncodeToString(i.Bytes), i.Timestamp.In(tz).Format("Mon, 02 Jan 2006 15:04:05"))
		//description = url
		f.Placemarks = append(f.Placemarks, kmlPlacemark{
			Description: description,
			Point:       &kmlPoint{longitude: i.Longitude, latitude: i.Latitude},
			Style:       icon,
		})
	}

	return f
}

func buildTrack(legs [][]kmlPoint) kmlFolder {
	legColors := []string{"FF0000FF", "FF00FF00"}

	f := kmlFolder{Name: "Track"}
	for legNum, leg := range legs {
		coordinates := make([]string, 0, len(leg))
		for i, point := range leg {
			if i%5 == 0 || i == len(leg)-1 {
				coordinates = append(coordinates, fmt.Sprintf("%.10f,%.10f", point.longitude, point.latitude))
			}
		}

		f.Placemarks = append(f.Placemarks, kmlPlacemark{
			Name: fmt.Sprintf("Leg %d", legNum),
			LineString: []kmlLineString{{
				Tessellate:   true,
				AltitudeMode: "clampToGround",
				Coordinates:  strings.Join(coordinates, " "),
			}},
			Style: &kmlStyle{
				LineStyle: &kmlLineStyle{
					Width: 3,
					Color: legColors[legNum%len(legColors)],
				},
			},
		})
	}

	return f
}

func main() {
	legs := buildKml(kmlDir)
	imageFolder := loadImages(imageDir).folder()
	trackFolder := buildTrack(legs)
	//finalKml.Folder.Folders = append(finalKml.Folder.Folders, imageFolder)
	finalKml := kml{
		Folder: kmlFolder{
			Name:    "route",
			Folders: []kmlFolder{trackFolder, imageFolder},
		},
	}

	if s, err := xml.MarshalIndent(finalKml, "", "    "); err != nil {
		log.Fatalf("marshaling: %s", err)
	} else {
		fmt.Printf("%s%s\n", xml.Header, s)
	}
}
