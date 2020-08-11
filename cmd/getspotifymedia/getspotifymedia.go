package getspotifymedia

import (
	"encoding/json"
	"log"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/pixelogicdev/gruveebackend/pkg/firebase"
	"github.com/pixelogicdev/gruveebackend/pkg/mediahelpers"
	"github.com/pixelogicdev/gruveebackend/pkg/sawmill"
	"github.com/pixelogicdev/gruveebackend/pkg/social"
)

// -- Types -- //

// getSpotifyMediaReq takes in the data needed to request the media data from Spotify
type getSpotifyMediaReq struct {
	Provider  string `json:"provider"`
	MediaID   string `json:"mediaId"`
	MediaType string `json:"mediaType"`
}

// spotifyTrackResp defines the data returned and needed from the Spotify Get Track API
type spotifyTrackResp struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Artists      []spotifyArtist `json:"artists"`
	Type         string          `json:"type"`
	Album        spotifyAlbum    `json:"album"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

// spotifyPlaylistResp defines the data returned and needed from the Spotify Get Playlist API
type spotifyPlaylistResp struct {
	ID     string                  `json:"id"`
	Name   string                  `json:"name"`
	Type   string                  `json:"type"`
	Images []firebase.SpotifyImage `json:"images"`
	Owner  struct {
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

// spotifyAlbum defines the data returned and needed from the Spotify Get Track API
type spotifyAlbum struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Type         string                  `json:"type"`
	Artists      []spotifyArtist         `json:"artists"`
	Images       []firebase.SpotifyImage `json:"images"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

// spotifyArtist defines the data returned and needed from the Spotify Get Track API
type spotifyArtist struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

var httpClient *http.Client
var firestoreClient *firestore.Client
var logger sawmill.Logger
var spotifyAccessTokenURI = "https://accounts.spotify.com/api/token"
var (
	httpClient            *http.Client
	firestoreClient       *firestore.Client
	spotifyAccessTokenURI = "https://accounts.spotify.com/api/token"
	spotifyGetTrackURI    = "https://api.spotify.com/v1/tracks"
	spotifyGetPlaylistURI = "https://api.spotify.com/v1/playlists"
	spotifyGetAlbumURI    = "https://api.spotify.com/v1/albums"
)

// Draco401 - "Draco401 was here." (04/17/20)
func init() {
	log.Println("GetSpotifyMedia Initialized")
}

// GetSpotifyMedia will take in Spotify media data and get the exact media from Spotify API
func GetSpotifyMedia(writer http.ResponseWriter, request *http.Request) {
	// Initialize
	initWithEnvErr := initWithEnv()
	if initWithEnvErr != nil {
		http.Error(writer, initWithEnvErr.Error(), http.StatusInternalServerError)
		logger.LogErr("InitWithEnv", initWithEnvErr, nil)
		return
	}

	// Decode Request body to get track data
	var spotifyMediaReq social.GetMediaReq
	spotifyReqDecodeErr := json.NewDecoder(request.Body).Decode(&spotifyMediaReq)
	if spotifyReqDecodeErr != nil {
		http.Error(writer, spotifyReqDecodeErr.Error(), http.StatusInternalServerError)
		logger.LogErr("Request Decoder", spotifyReqDecodeErr, request)
		return
	}

	// Check to see if media is already part of collection, if so, just return that
	mediaData, mediaDataErr := mediahelpers.GetMediaFromFirestore(*firestoreClient, spotifyMediaReq.Provider, spotifyMediaReq.MediaID)
	if mediaDataErr != nil {
		http.Error(writer, mediaDataErr.Error(), http.StatusInternalServerError)
		logger.LogErr("GetMediaFromFirestore", mediaDataErr, request)
		return
	}

	// MediaData exists, return it to the client
	if mediaData != nil {
		log.Printf("Media already exists, returning")
		writer.WriteHeader(http.StatusOK)
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(mediaData)
		return
	}

	// Get Spotify access token (currently getting access token of user)
	creds, credErr := getCreds()
	if credErr != nil {
		http.Error(writer, credErr.Error(), http.StatusInternalServerError)
		logger.LogErr("GetCreds", credErr, nil)
		return
	}

	var (
		firestoreMediaData    interface{}
		firestoreMediaDataErr error
	)

	// Setup and call Spotify search
	switch spotifyMediaReq.MediaType {
	case "track":
		firestoreMediaData, firestoreMediaDataErr = getSpotifyTrack(spotifyMediaReq.MediaID, creds.Token)
		if firestoreMediaDataErr != nil {
			http.Error(writer, firestoreMediaDataErr.Error(), http.StatusInternalServerError)
			logger.LogErr("GetSpotifyAlbum Switch", firestoreMediaDataErr, request)
			return
		}
	case "playlist":
		firestoreMediaData, firestoreMediaDataErr = getSpotifyPlaylist(spotifyMediaReq.MediaID, creds.Token)
		if firestoreMediaDataErr != nil {
			http.Error(writer, firestoreMediaDataErr.Error(), http.StatusInternalServerError)
			logger.LogErr("GetSpotifyAlbum Switch", firestoreMediaDataErr, request)
			return
		}
	case "album":
		firestoreMediaData, firestoreMediaDataErr = getSpotifyAlbum(spotifyMediaReq.MediaID, creds.Token)
		if firestoreMediaDataErr != nil {
			http.Error(writer, firestoreMediaDataErr.Error(), http.StatusInternalServerError)
			logger.LogErr("GetSpotifyAlbum Switch", firestoreMediaDataErr, request)
			return
		}
	default:
		http.Error(writer, spotifyMediaReq.MediaType+" media type does not exist", http.StatusInternalServerError)
		log.Printf("GetSpotifyMedia [MediaTypeSwitch]: %v media type does not exist", spotifyMediaReq.MediaType)
		logger.LogErr("MediaTypeSwitch", fmt.Errorf("%v media type does not exist", spotifyMediaReq.MediaType), request)
		return
	}
}

// Helpers

// initWithEnv takes our yaml env variables and maps them properly.
// Unfortunately, we had to do this is main because in init we weren't able to access env variables
func initWithEnv() error {
	// Get paths
	var currentProject string

	if os.Getenv("ENVIRONMENT") == "DEV" {
		currentProject = os.Getenv("FIREBASE_PROJECTID_DEV")
	} else if os.Getenv("ENVIRONMENT") == "PROD" {
		currentProject = os.Getenv("FIREBASE_PROJECTID_PROD")
	}

	// Initialize Firestore
	client, err := firestore.NewClient(context.Background(), currentProject)
	if err != nil {
		return fmt.Errorf("GetSpotifyMedia [Init Firestore]: %v", err)
	}

	// Initialize Sawmill
	sawmillLogger, err := sawmill.InitClient(currentProject, os.Getenv("GCLOUD_CONFIG"), os.Getenv("ENVIRONMENT"), "GetSpotifyMedia")
	if err != nil {
		log.Printf("GetSpotifyMedia [Init Sawmill]: %v", err)
	}

	logger = sawmillLogger
	firestoreClient = client
	return nil
}

func getMediaFromFirestore(mediaID string) (*firebase.FirestoreMedia, error) {
	// Use song id to see if it's in the collection, if it is, just return the data
	mediaRef := firestoreClient.Doc("songs/" + mediaID)
	if mediaRef == nil {
		return nil, fmt.Errorf("Incorrect path formating when get media from collection")
	}

	// If uid does not exist return nil
	mediaSnap, mediaSnapErr := mediaRef.Get(context.Background())
	if status.Code(mediaSnapErr) == codes.NotFound {
		log.Printf("Media with id %s was not found", mediaID)
		return nil, nil
	}

	var firestoreMedia firebase.FirestoreMedia
	dataToErr := mediaSnap.DataTo(&firestoreMedia)
	if dataToErr != nil {
		return nil, fmt.Errorf("%v", dataToErr)
	}

	return &firestoreMedia, nil
}

// getApiToken checks to see if we have an APIToken for client credential calls
func getCreds() (*social.SpotifyClientCredsAuthResp, error) {
	// Check to see if we have env variables
	if os.Getenv("SPOTIFY_CLIENTID") == "" || os.Getenv("SPOTIFY_SECRET") == "" {
		log.Fatalln("GetSpotifyMedia [Check Env Props]: PROPS NOT HERE.")
		return nil, fmt.Errorf("getSpotifyMedia [Check Env Props]: PROPS NOT HERE")
	}

	// Generate authStr for requests
	authStr := os.Getenv("SPOTIFY_CLIENTID") + ":" + os.Getenv("SPOTIFY_SECRET")

	// Create Request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	accessTokenReq, accessTokenReqErr := http.NewRequest("POST", spotifyAccessTokenURI,
		strings.NewReader(data.Encode()))
	if accessTokenReqErr != nil {
		return nil, fmt.Errorf(accessTokenReqErr.Error())
	}

	accessTokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	accessTokenReq.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(authStr)))
	accessTokenResp, accessTokenRespErr := httpClient.Do(accessTokenReq)
	if accessTokenRespErr != nil {
		return nil, fmt.Errorf(accessTokenRespErr.Error())
	}

	// Decode the token to send back
	var spotifyClientCredsAuthResp social.SpotifyClientCredsAuthResp
	refreshTokenDecodeErr := json.NewDecoder(accessTokenResp.Body).Decode(&spotifyClientCredsAuthResp)
	if refreshTokenDecodeErr != nil {
		return nil, fmt.Errorf(refreshTokenDecodeErr.Error())
	}

	return &spotifyClientCredsAuthResp, nil

	// For now, let's just get the token directly from spotify
	/* spotifyCredsRef := firestoreClient.Doc("platform_credentials/spotify")
	if spotifyCredsRef == nil {
		return nil, fmt.Errorf("spotify credentials do not exist")
	}

	// Grab token see if it is expired
	spotifyCredSnap, spotifyCredSnapErr := spotifyCredsRef.Get(context.Background())
	if status.Code(spotifyCredSnapErr) == codes.NotFound {
		return nil, fmt.Errorf("Spotify cred was not found")
	}

	var spotifyCreds platformCredentials
	dataErr := spotifyCredSnap.DataTo(&spotifyCreds)
	if dataErr != nil {
		return nil, fmt.Errorf("doesUserExist: %v", dataErr)
	}

	log.Println(spotifyCreds)
	return &spotifyCreds, nil */
}

// getSpotifyTrack calls Spotify GET track API and converts to Golang Type
func getSpotifyTrack(trackID string, accessToken string) (*firebase.FirestoreMedia, error) {
	// Generate URI
	spotifyGetURI := spotifyGetTrackURI + "/" + trackID

	// Generate request
	spotifyTrackReq, spotifyTrackReqErr := http.NewRequest("GET", spotifyGetURI, nil)
	if spotifyTrackReqErr != nil {
		return nil, fmt.Errorf("GetSpotifyMedia [http.NewRequest]: %v", spotifyTrackReqErr)
	}

	// Add headers and call request
	spotifyTrackReq.Header.Add("Authorization", "Bearer "+accessToken)
	getTrackResp, spotifyTrackRespErr := httpClient.Do(spotifyTrackReq)
	if spotifyTrackRespErr != nil {
		return nil, fmt.Errorf("GetSpotifyMedia [client.Do]: %v", spotifyTrackRespErr)
	}

	// Check to see if request was valid
	if getTrackResp.StatusCode != http.StatusOK {
		// Convert Spotify Error Object
		var spotifyErrorObj social.SpotifyRequestError

		err := json.NewDecoder(getTrackResp.Body).Decode(&spotifyErrorObj)
		if err != nil {
			return nil, fmt.Errorf("GetSpotifyMedia [Spotify Request Decoder]: %v", err)
		}
		return nil, fmt.Errorf("GetSpotifyMedia [Spotify Track Request]: %v", spotifyErrorObj.Error.Message)
	}

	var spotifyTrackData spotifyTrackResp

	// syszen - "wait that it? #easyGo"(02/27/20)
	// LilCazza - "Why the fuck doesn't this shit work" (02/27/20)
	respDecodeErr := json.NewDecoder(getTrackResp.Body).Decode(&spotifyTrackData)
	if respDecodeErr != nil {
		return nil, fmt.Errorf("GetSpotifyMedia [Spotify Response Decoder]: %v", respDecodeErr)
	}

	// If multiple artists append to a string
	var creators string
	for index, artist := range spotifyTrackData.Artists {
		if index == 0 {
			creators = artist.Name
		} else {
			creators = creators + ", " + artist.Name
		}
	}

	// Setup FirestoreMeida object
	firestoreMedia := firebase.FirestoreMedia{
		ID:           spotifyTrackData.ID,
		Name:         spotifyTrackData.Name,
		Album:        spotifyTrackData.Album.Name,
		Type:         spotifyTrackData.Type,
		Creator:      creators,
		Images:       spotifyTrackData.Album.Images,
		ExternalURLs: map[string]string{"spotify": spotifyTrackData.ExternalURLs.Spotify},
	}

	return &firestoreMedia, nil
}

func getSpotifyPlaylist(playlistID string, accessToken string) (*firebase.FirestoreMedia, error) {
	// Generate URI
	spotifyGetURI := spotifyGetPlaylistURI + "/" + playlistID

	// Generate request
	spotifyPlaylistReq, spotifyPlaylistReqErr := http.NewRequest("GET", spotifyGetURI, nil)
	if spotifyPlaylistReqErr != nil {
		return nil, fmt.Errorf("GetSpotifyMedia [http.NewRequest]: %v", spotifyPlaylistReqErr)
	}

	// Add headers and call request
	spotifyPlaylistReq.Header.Add("Authorization", "Bearer "+accessToken)
	getPlaylistResp, spotifyPlaylistRespErr := httpClient.Do(spotifyPlaylistReq)
	if spotifyPlaylistRespErr != nil {
		return nil, fmt.Errorf("GetSpotifyMedia [client.Do]: %v", spotifyPlaylistRespErr)
	}

	// Check to see if request was valid
	if getPlaylistResp.StatusCode != http.StatusOK {
		// Convert Spotify Error Object
		var spotifyErrorObj social.SpotifyRequestError

		err := json.NewDecoder(getPlaylistResp.Body).Decode(&spotifyErrorObj)
		if err != nil {
			return nil, fmt.Errorf("GetSpotifyMedia [Spotify Request Decoder]: %v", err)
		}
		return nil, fmt.Errorf("GetSpotifyMedia [Spotify Track Request]: %v", spotifyErrorObj.Error.Message)
	}

	var playlistData spotifyPlaylistResp

	// syszen - "wait that it? #easyGo"(02/27/20)
	// LilCazza - "Why the fuck doesn't this shit work" (02/27/20)
	respDecodeErr := json.NewDecoder(getPlaylistResp.Body).Decode(&playlistData)
	if respDecodeErr != nil {
		return nil, fmt.Errorf("GetSpotifyMedia [Spotify Response Decoder]: %v", respDecodeErr)
	}

	// Setup FirestoreMeida object
	firestoreMedia := firebase.FirestoreMedia{
		ID:           playlistData.ID,
		Name:         playlistData.Name,
		Album:        "Playlist",
		Type:         playlistData.Type,
		Creator:      playlistData.Owner.DisplayName,
		Images:       playlistData.Images,
		ExternalURLs: map[string]string{"spotify": playlistData.ExternalURLs.Spotify},
	}

	writer.WriteHeader(http.StatusOK)
	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(firestoreMediaData)
}
