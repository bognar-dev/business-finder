package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/jomei/notionapi"
	"googlemaps.github.io/maps"
	"log"
	"os"
	"time"
)

// Business represents a business entity
type Business struct {
	Name          string
	Address       string
	PlaceID       string
	Type          []string
	WebsiteStatus string
	Urgency       string
	Contacted     string
	URL           string
}

// NotionClient handles interactions with the Notion API
type NotionClient struct {
	client     *notionapi.Client
	databaseID notionapi.DatabaseID
	pageID     notionapi.PageID
}

// NewNotionClient initializes a new NotionClient
func NewNotionClient(apiKey, databaseID string, pageID string) *NotionClient {
	client := notionapi.NewClient(notionapi.Token(apiKey))
	return &NotionClient{
		client:     client,
		databaseID: notionapi.DatabaseID(databaseID),
		pageID:     notionapi.PageID(pageID),
	}
}

// CheckDatabaseExists checks if the Notion database exists
func (nc *NotionClient) CheckDatabaseExists() bool {
	res, err := nc.client.Database.Get(context.Background(), nc.databaseID)
	fmt.Println(res)
	return err == nil
}

// CreateDatabase creates a Notion database
func (nc *NotionClient) CreateDatabase() error {
	properties := notionapi.PropertyConfigs{
		"Name": notionapi.TitlePropertyConfig{
			Type: notionapi.PropertyConfigTypeTitle,
		},
		"Address": notionapi.RichTextPropertyConfig{
			Type: notionapi.PropertyConfigTypeRichText,
		},
		"PlaceID": notionapi.RichTextPropertyConfig{
			Type: notionapi.PropertyConfigTypeRichText,
		},
		"Type": notionapi.MultiSelectPropertyConfig{
			Type: notionapi.PropertyConfigTypeMultiSelect,
			MultiSelect: notionapi.Select{
				Options: []notionapi.Option{
					{Name: "Restaurant"},
					{Name: "Shop"},
					{Name: "Business"},
				},
			},
		},
		"WebsiteStatus": notionapi.SelectPropertyConfig{
			Type: notionapi.PropertyConfigTypeSelect,
			Select: notionapi.Select{
				Options: []notionapi.Option{
					{Name: "Has Website"},
					{Name: "No Website"},
				},
			},
		},
		"Urgency": notionapi.SelectPropertyConfig{
			Type: notionapi.PropertyConfigTypeSelect,
			Select: notionapi.Select{
				Options: []notionapi.Option{
					{Name: "High"},
					{Name: "Medium"},
					{Name: "Low"},
				},
			},
		},
		"Contacted": notionapi.SelectPropertyConfig{
			Type: notionapi.PropertyConfigTypeSelect,
			Select: notionapi.Select{
				Options: []notionapi.Option{
					{Name: "Not Contacted"},
					{Name: "Contacted"},
				},
			},
		},
		"URL": notionapi.URLPropertyConfig{
			Type: notionapi.PropertyConfigTypeURL,
		},
	}

	dbCreateRequest := notionapi.DatabaseCreateRequest{
		Parent:     notionapi.Parent{Type: notionapi.ParentTypePageID, PageID: nc.pageID},
		Title:      []notionapi.RichText{{Text: &notionapi.Text{Content: "Businesses"}}},
		Properties: properties,
		IsInline:   false,
	}

	db, err := nc.client.Database.Create(context.Background(), &dbCreateRequest)
	nc.databaseID = notionapi.DatabaseID(db.ID)
	return err
}

// Add a method to check if a business already exists in the Notion database
func (nc *NotionClient) BusinessExists(placeID string) (bool, error) {
	query := &notionapi.DatabaseQueryRequest{
		Filter: &notionapi.PropertyFilter{
			Property: "PlaceID",
			RichText: &notionapi.TextFilterCondition{
				Equals: placeID,
			},
		},
	}

	res, err := nc.client.Database.Query(context.Background(), nc.databaseID, query)
	if err != nil {
		return false, err
	}

	return len(res.Results) > 0, nil
}

func (nc *NotionClient) InsertBusiness(business Business) error {
	exists, err := nc.BusinessExists(business.PlaceID)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("Business with PlaceID %s already exists, skipping...\n", business.PlaceID)
		return nil
	}

	var multiSelectOptions []notionapi.Option
	for _, t := range business.Type {
		multiSelectOptions = append(multiSelectOptions, notionapi.Option{Name: t})
	}

	page := notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			DatabaseID: nc.databaseID,
		},
		Properties: notionapi.Properties{
			"Name": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: business.Name,
						},
					},
				},
			},
			"Address": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: business.Address,
						},
					},
				},
			},
			"PlaceID": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: business.PlaceID,
						},
					},
				},
			},
			"Type": notionapi.MultiSelectProperty{
				MultiSelect: multiSelectOptions,
			},
			"WebsiteStatus": notionapi.SelectProperty{
				Select: notionapi.Option{
					Name: business.WebsiteStatus,
				},
			},
			"Urgency": notionapi.SelectProperty{
				Select: notionapi.Option{
					Name: business.Urgency,
				},
			},
			"Contacted": notionapi.SelectProperty{
				Select: notionapi.Option{
					Name: business.Contacted,
				},
			},
			"URL": notionapi.URLProperty{
				URL: business.URL,
			},
		},
	}

	_, err = nc.client.Page.Create(context.Background(), &page)
	return err
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := os.Getenv("GOOGLE_PLACES_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_PLACES_API_KEY must be set")
	}
	notionAPIKey := os.Getenv("NOTION_API_KEY")
	notionDatabaseID := os.Getenv("NOTION_DATABASE_ID")
	if notionAPIKey == "" || notionDatabaseID == "" {
		log.Fatal("NOTION_API_KEY and NOTION_DATABASE_ID must be set")
	}
	notionPageID := os.Getenv("NOTION_PAGE_ID")

	// Initialize Notion client
	notionClient := NewNotionClient(notionAPIKey, notionDatabaseID, notionPageID)

	// Check if the Notion database exists
	if !notionClient.CheckDatabaseExists() {
		fmt.Println("Database does not exist, creating it...")
		err := notionClient.CreateDatabase()
		if err != nil {
			log.Fatalf("Failed to create Notion database: %v", err)
		}
	}

	// Initialize Google Maps client
	mapsClient, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create Google Maps client: %v", err)
	}

	placeTypes := []maps.PlaceType{
		maps.PlaceTypeArtGallery,
		maps.PlaceTypeBakery,
		maps.PlaceTypeBank,
		maps.PlaceTypeBar,
		maps.PlaceTypeBeautySalon,
		maps.PlaceTypeBicycleStore,
		maps.PlaceTypeBookStore,
		maps.PlaceTypeBowlingAlley,
		maps.PlaceTypeCafe,
		maps.PlaceTypeCampground,
		maps.PlaceTypeClothingStore,
		maps.PlaceTypeConvenienceStore,
		maps.PlaceTypeDepartmentStore,
		maps.PlaceTypeElectrician,
		maps.PlaceTypeElectronicsStore,
		maps.PlaceTypeFlorist,
		maps.PlaceTypeFuneralHome,
		maps.PlaceTypeGym,
		maps.PlaceTypeHairCare,
		maps.PlaceTypeHomeGoodsStore,
		maps.PlaceTypeJewelryStore,
		maps.PlaceTypeLaundry,
		maps.PlaceTypeLibrary,
		maps.PlaceTypeLiquorStore,
		maps.PlaceTypeLocksmith,
		maps.PlaceTypeLodging,
		maps.PlaceTypeMealDelivery,
		maps.PlaceTypeMealTakeaway,
		maps.PlaceTypeMovieRental,
		maps.PlaceTypeMovingCompany,
		maps.PlaceTypeMuseum,
		maps.PlaceTypeNightClub,
		maps.PlaceTypePainter,
		maps.PlaceTypePetStore,
		maps.PlaceTypePhysiotherapist,
		maps.PlaceTypePlumber,
		maps.PlaceTypeRestaurant,
		maps.PlaceTypeRoofingContractor,
		maps.PlaceTypeRvPark,
		maps.PlaceTypeShoeStore,
		maps.PlaceTypeShoppingMall,
		maps.PlaceTypeSpa,
		maps.PlaceTypeStorage,
		maps.PlaceTypeStore,
		maps.PlaceTypeSupermarket,
		maps.PlaceTypeTravelAgency,
		maps.PlaceTypeVeterinaryCare,
	}

	for _, placeType := range placeTypes {
		fmt.Printf("Searching for places of type: %s\n", placeType)

		req := &maps.NearbySearchRequest{
			Location: &maps.LatLng{
				Lat: 50.152573,
				Lng: -5.066270,
			},
			Radius: 50000,
			Type:   placeType,
		}

		pageCount := 0
		for {
			pageCount++
			fmt.Printf("Fetching page %d for %s\n", pageCount, placeType)

			places, err := mapsClient.NearbySearch(context.Background(), req)
			if err != nil {
				log.Printf("Failed to perform nearby search for %s: %v", placeType, err)
				break
			}

			fmt.Printf("Found %d results on this page\n", len(places.Results))

			for _, place := range places.Results {
				placeDetailsReq := &maps.PlaceDetailsRequest{
					PlaceID: place.PlaceID,
				}

				details, err := mapsClient.PlaceDetails(context.Background(), placeDetailsReq)
				if err != nil {
					log.Printf("Failed to get place details for %s: %v", place.Name, err)
					continue
				}

				websiteStatus := "No Website"
				urgency := "High"
				url := ""

				if details.Website != "" {
					websiteStatus = "Has Website"
					url = details.Website
					urgency = "Medium"
				}

				businessType := []string{"Other"}
				if len(place.Types) > 0 {
					businessType = place.Types
				}

				business := Business{
					Name:          place.Name,
					Address:       place.FormattedAddress,
					PlaceID:       place.PlaceID,
					Type:          businessType,
					WebsiteStatus: websiteStatus,
					Urgency:       urgency,
					Contacted:     "Not Contacted",
					URL:           url,
				}
				if business.WebsiteStatus == "No Website" {
					business.URL = "https://www.google.com/maps/search/?api=1&query=" + business.Address
				}

				// Insert into Notion
				err = notionClient.InsertBusiness(business)
				if err != nil {
					log.Printf("Failed to insert into Notion: %v", err)
				} else {
					fmt.Printf("Inserted: Name: %s, Address: %s, Types: %v, WebsiteStatus: %s, Urgency: %s\n", place.Name, place.FormattedAddress, businessType, websiteStatus, urgency)
				}
			}

			if places.NextPageToken == "" {
				fmt.Printf("No more pages for %s\n", placeType)
				break
			}

			fmt.Printf("Waiting before fetching next page...\n")
			time.Sleep(5 * time.Second) // Increased delay to avoid rate limiting
			req.PageToken = places.NextPageToken
		}
	}
}
