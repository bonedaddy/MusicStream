/*
 * MusicStream - Listen to music together with your friends from everywhere, at the same time.
 * Copyright (C) 2020 Nguyễn Hoàng Trung(TrungNguyen1909)
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package deezer

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/TrungNguyen1909/MusicStream/spotify"
	"github.com/TrungNguyen1909/MusicStream/streamdecoder"

	"github.com/TrungNguyen1909/MusicStream/common"

	"golang.org/x/crypto/blowfish"
	"golang.org/x/text/encoding/charmap"
)

const (
	deezerURL          = "https://www.deezer.com"
	ajaxActionURL      = "https://www.deezer.com/ajax/action.php"
	unofficialAPIURL   = "https://www.deezer.com/ajax/gw-light.php"
	searchResultsLimit = 4
	trackQualityID     = 3
)

//Artist represents an artist on Deezer
type Artist struct {
	Name string `json:"name"`
}

//Album represents an album on Deezer
type Album struct {
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	CoverSmall  string `json:"cover_small"`
	CoverMedium string `json:"cover_medium"`
	CoverBig    string `json:"cover_big"`
	CoverXL     string `json:"cover_xl"`
}

//Track represents a track on Deezer
type Track struct {
	deezerTrack
	StreamURL   string
	BlowfishKey []byte
	playID      string
}

//ID returns the track's ID number on Deezer
func (track *Track) ID() string {
	return strconv.Itoa(track.deezerTrack.ID)
}

//Title returns the track's title
func (track *Track) Title() string {
	return track.deezerTrack.Title
}

//Album returns the track's album title
func (track *Track) Album() string {
	return track.deezerTrack.Album.Title
}

//Source returns the track's source
func (track *Track) Source() int {
	return common.Deezer
}

//Artist returns the track's main artist
func (track *Track) Artist() string {
	return track.deezerTrack.Artist.Name
}

//Artists returns the track's contributors' name, comma-separated
func (track *Track) Artists() string {
	artists := ""
	if len(track.deezerTrack.Contributors) > 0 {
		for _, v := range track.deezerTrack.Contributors {
			artists = strings.Join([]string{artists, v.Name}, ", ")
		}
		artists = artists[2:]
	} else {
		artists = track.deezerTrack.Artist.Name
	}
	return artists
}

//Duration returns the track's duration
func (track *Track) Duration() int {
	return track.deezerTrack.Duration
}

//ISRC returns the track's ISRC ID
func (track *Track) ISRC() string {
	return track.deezerTrack.ISRC
}

//CoverURL returns the URL to track's cover art
func (track *Track) CoverURL() string {
	return track.deezerTrack.Album.CoverXL
}

//Download returns a mp3 stream of the track
func (track *Track) Download() (stream io.ReadCloser, err error) {
	if track.StreamURL == "" || len(track.BlowfishKey) == 0 {
		err = errors.New("Metadata not yet populated")
		return
	}
	response, err := http.DefaultClient.Get(track.StreamURL)
	if err != nil {
		return
	}
	if response.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprint("deezerTrack Download failed: ", track.StreamURL, " ", response.Status))
		return
	}
	stream, err = streamdecoder.NewMP3Decoder(&trackDecrypter{r: response.Body, BlowfishKey: track.BlowfishKey})
	if err != nil {
		return
	}
	return
}

//Populate populates track metadata for Download
func (track *Track) Populate() (err error) {
	return track.client.PopulateMetadata(track)
}

//SpotifyURI returns the track's equivalent spotify song, if known
func (track *Track) SpotifyURI() string {
	return track.deezerTrack.SpotifyURI
}

//PlayID returns a random string which is unique to this instance of Track
func (track *Track) PlayID() string {
	return track.playID
}

//SetSpotifyURI set the track's SpotifyURI with the provided one
func (track *Track) SetSpotifyURI(sURI string) {
	track.deezerTrack.SpotifyURI = sURI
}

type getUserDataResults struct {
	CheckForm string `json:"checkForm"`
}
type getUserDataResponse struct {
	Error   []interface{}      `json:"error"`
	Results getUserDataResults `json:"results"`
}
type pageTrackData struct {
	AlbID      string `json:"ALB_ID"`
	AlbPicture string `json:"ALB_PICTURE"`
	AlbTitle   string `json:"ALB_TITLE"`
	Artists    []struct {
		ArtID      string `json:"ART_ID"`
		ArtName    string `json:"ART_NAME"`
		ArtPicture string `json:"ART_PICTURE"`
	} `json:"ARTISTS"`
	ArtID              string `json:"ART_ID"`
	ArtName            string `json:"ART_NAME"`
	DigitalReleaseDate string `json:"DIGITAL_RELEASE_DATE"`
	DiskNumber         string `json:"DISK_NUMBER"`
	Duration           string `json:"DURATION"`
	ExplicitLyrics     string `json:"EXPLICIT_LYRICS"`
	Filesize           string `json:"FILESIZE"`
	FileSizeAAC64      string `json:"FILESIZE_AAC_64"`
	FilesizeFlac       string `json:"FILESIZE_FLAC"`
	FilesizeMP3_128    string `json:"FILESIZE_MP3_128"`
	FilesizeMP3_256    string `json:"FILESIZE_MP3_256"`
	FilesizeMP3_320    string `json:"FILESIZE_MP3_320"`
	FilesizeMP3_64     string `json:"FILESIZE_MP3_64"`
	FilesizeMP4RA1     string `json:"FILESIZE_MP4_RA1"`
	FilesizeMP4RA2     string `json:"FILESIZE_MP4_RA2"`
	FilesizeMP4RA3     string `json:"FILESIZE_MP4_RA3"`
	Gain               string `json:"GAIN"`
	GenreID            string `json:"GENRE_ID"`
	Isrc               string `json:"ISRC"`
	LyricsID           int64  `json:"LYRICS_ID"`
	MD5Origin          string `json:"MD5_ORIGIN"`
	Media              []struct {
		Href string `json:"HREF"`
		Type string `json:"TYPE"`
	} `json:"MEDIA"`
	MediaVersion        string `json:"MEDIA_VERSION"`
	PhysicalReleaseDate string `json:"PHYSICAL_RELEASE_DATE"`
	ProviderID          string `json:"PROVIDER_ID"`
	RankSng             string `json:"RANK_SNG"`
	Smartradio          int64  `json:"SMARTRADIO"`
	SngID               string `json:"SNG_ID"`
	SngTitle            string `json:"SNG_TITLE"`
	Status              int64  `json:"STATUS"`
	TrackNumber         string `json:"TRACK_NUMBER"`
	TrackToken          string `json:"TRACK_TOKEN"`
	TrackTokenExpire    int64  `json:"TRACK_TOKEN_EXPIRE"`
	UploadID            int64  `json:"UPLOAD_ID"`
	UserID              int64  `json:"USER_ID"`
	Version             string `json:"VERSION"`
	Type                string `json:"__TYPE__"`
}
type pageTrackResults struct {
	Data  []pageTrackData `json:"data"`
	Count int64           `json:"count"`
}
type pageTrackResponse struct {
	Error   []interface{}    `json:"error"`
	Results pageTrackResults `json:"results"`
}
type searchTrackResponse struct {
	Data []deezerTrack `json:"data"`
}

type deezerTrack struct {
	ID           int      `json:"id"`
	Title        string   `json:"title"`
	Artist       Artist   `json:"artist"`
	Artists      string   `json:"artists"`
	Contributors []Artist `json:"contributors"`
	Album        Album    `json:"album"`
	Duration     int      `json:"duration"`
	Rank         int      `json:"rank"`
	ISRC         string   `json:"isrc"`
	SpotifyURI   string
	MD5Origin    string
	MediaVersion string
	client       *Client
}

type trackDecrypter struct {
	r           io.ReadCloser
	BlowfishKey []byte
	counter     int
	byteCounter int
	buffer      bytes.Buffer
	ended       bool
}

func (decrypter *trackDecrypter) createCipher() cipher.BlockMode {

	blowfishEngine, err := blowfish.NewCipher(decrypter.BlowfishKey)
	if err != nil {
		log.Panic(err)
	}
	blowfishCBC := cipher.NewCBCDecrypter(blowfishEngine, []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07})
	return blowfishCBC
}
func (decrypter *trackDecrypter) decrypt(n int) (err error) {
	for decrypter.buffer.Len() < n && !decrypter.ended {
		buf := make([]byte, 2048)
		var size int
		size, err = io.ReadFull(decrypter.r, buf)
		buf = buf[:size]
		if decrypter.counter%3 == 0 && len(buf) == 2048 {
			blowfish := decrypter.createCipher()
			blowfish.CryptBlocks(buf, buf)
		}
		decrypter.counter++
		decrypter.buffer.Write(buf)
		if err != nil {
			decrypter.ended = true
			break
		}
	}
	return
}
func (decrypter *trackDecrypter) Read(p []byte) (n int, err error) {
	err = decrypter.decrypt(len(p))
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		log.Println("trackDecrypter.decrypt failed: ", err)
	}
	n, err = decrypter.buffer.Read(p)
	if err == nil && decrypter.ended {
		err = io.EOF
	}
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		log.Println("trackDecrypter.Read failed: ", err)
	}
	return n, err
}
func (decrypter *trackDecrypter) Close() error {
	return decrypter.r.Close()
}

//Client represents a Deezer client
type Client struct {
	httpHeaders        http.Header
	arlCookie          string
	httpClient         *http.Client
	ajaxActionURL      *url.URL
	unofficialAPIURL   *url.URL
	unofficialAPIQuery url.Values
	deezerURL          *url.URL
	spotifyClient      *spotify.Client
}

//NewClient returns a new Deezer Client
func NewClient(deezerARL, spotifyClientID, spotifyClientSecret string) (client *Client) {
	client = &Client{}
	cookiesJar, _ := cookiejar.New(nil)
	client.httpClient = &http.Client{Jar: cookiesJar}
	client.httpHeaders = http.Header{}
	client.unofficialAPIQuery = make(url.Values)
	client.httpHeaders.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36")
	client.httpHeaders.Set("cache-control", "max-age=0")
	client.httpHeaders.Set("accept-language", "en-US,en;q=0.9,en-US;q=0.8,en;q=0.7")
	client.httpHeaders.Set("accept-charset", "utf-8,ISO-8859-1;q=0.8,*;q=0.7")
	client.httpHeaders.Set("content-type", "text/plain;charset=UTF-8")
	client.httpHeaders.Set("cookie", "arl="+deezerARL)
	client.arlCookie = deezerARL
	client.ajaxActionURL, _ = url.Parse(ajaxActionURL)
	client.unofficialAPIURL, _ = url.Parse(unofficialAPIURL)
	client.unofficialAPIQuery.Set("api_version", "1.0")
	client.unofficialAPIQuery.Set("input", "3")
	client.unofficialAPIQuery.Set("api_token", "")
	client.deezerURL, _ = url.Parse(deezerURL)
	spotifyClient, err := spotify.NewClient(spotifyClientID, spotifyClientSecret)
	if err != nil {
		log.Printf("spotify.NewClient() failed: %v\n", err)
	} else {
		client.spotifyClient = spotifyClient
	}
	client.initDeezerAPI()
	return
}

func getAPICID() string {
	return strconv.Itoa(int(math.Floor(rand.Float64() * 1e9)))
}
func (client *Client) cleanupCookieJar() {
	cookiesJar, _ := cookiejar.New(nil)
	client.httpClient.Jar = cookiesJar
}
func (client *Client) makeRequest(method, url string, body []byte) *http.Request {
	request, _ := http.NewRequest(method, url, bytes.NewReader(body))
	request.Header = client.httpHeaders
	return request
}
func (client *Client) makeUnofficialAPIRequest(method string, body []byte) *http.Request {
	client.unofficialAPIQuery.Set("method", method)
	client.unofficialAPIQuery.Set("cid", getAPICID())
	client.unofficialAPIURL.RawQuery = client.unofficialAPIQuery.Encode()
	return client.makeRequest("POST", client.unofficialAPIURL.String(), body)
}
func (client *Client) initDeezerAPI() {
	client.unofficialAPIQuery.Set("api_token", "")
	request := client.makeUnofficialAPIRequest("deezer.getUserData", []byte(""))
	client.cleanupCookieJar()
	response, err := client.httpClient.Do(request)
	if err != nil {
		log.Println("deezer.initDeezerAPI() failed: ", err)
		return
	}
	defer response.Body.Close()
	var resp getUserDataResponse
	err = json.NewDecoder(response.Body).Decode(&resp)
	if err != nil || len(resp.Results.CheckForm) <= 0 {
		return
	}
	client.unofficialAPIQuery.Set("api_token", resp.Results.CheckForm)
	log.Printf("Successfully initiated Deezer API. Checkform: \"%s\"\n", resp.Results.CheckForm)
}
func getSongFileName(trackInfo deezerTrack) string {
	encoder := charmap.Windows1252.NewEncoder()
	step1 := strings.Join([]string{trackInfo.MD5Origin, strconv.Itoa(trackQualityID), strconv.Itoa(trackInfo.ID), trackInfo.MediaVersion}, "¤")
	step1encoded, _ := encoder.Bytes([]byte(step1))
	step2 := fmt.Sprintf("%x¤%s¤", md5.Sum([]byte(step1encoded)), step1)

	step2encoded, _ := encoder.Bytes([]byte(step2))
	for ; len(step2encoded)%16 != 0; step2encoded = append(step2encoded, byte(' ')) {
	}
	cipher, _ := aes.NewCipher([]byte("jo6aey6haid2Teih"))
	result := make([]byte, len(step2encoded))
	for bs, be := 0, 16; bs < len(step2encoded); bs, be = bs+16, be+16 {
		cipher.Encrypt(result[bs:be], step2encoded[bs:be])
	}
	return fmt.Sprintf("%x", result)
}
func getBlowfishKey(trackInfo deezerTrack) (bfKey []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			bfKey = nil
			err = errors.New("getBlowfishKey: Panicked")
			log.Println(r)
		}
	}()
	SECRET := "g4el58wc0zvf9na1"
	encoder := charmap.Windows1252.NewEncoder()
	sngid, _ := encoder.Bytes([]byte(strconv.Itoa(trackInfo.ID)))
	idMd5 := fmt.Sprintf("%x", md5.Sum(sngid))
	bfKey = make([]byte, 16)
	for i := range bfKey {
		bfKey[i] = idMd5[i] ^ idMd5[i+16] ^ SECRET[i]
	}
	return
}
func getTrackDownloadURL(trackInfo deezerTrack) (url string, err error) {
	defer func() {
		if r := recover(); r != nil {
			url = ""
			err = errors.New("getTrackDownloadURL Panicked")
			log.Println(r)
		}
	}()
	cdn := trackInfo.MD5Origin[0]
	url = strings.Join([]string{"https://e-cdns-proxy-", string(cdn), ".dzcdn.net/mobile/1/", getSongFileName(trackInfo)}, "")
	return
}

//PopulateMetadata populates the required metadata for downloading the track
func (client *Client) PopulateMetadata(dTrack *Track) (err error) {
	if len(dTrack.deezerTrack.MD5Origin) <= 0 {
		if client == nil {
			err = errors.New("PopulateMetadata: nil Deezer Client")
		}
		err = client.populateTracks([]deezerTrack{dTrack.deezerTrack})
		if err != nil {
			return
		}
	}
	dTrack.StreamURL, err = getTrackDownloadURL(dTrack.deezerTrack)
	if err != nil {
		return
	}
	dTrack.BlowfishKey, err = getBlowfishKey(dTrack.deezerTrack)
	if err != nil {
		return
	}
	return
}

//GetTrackByID returns the populated track with the provided ID on Deezer
func (client *Client) GetTrackByID(trackID string) (track common.Track, err error) {
	var url string
	var dTrack deezerTrack
	url = fmt.Sprintf("https://api.deezer.com/track/%s", trackID)
	response, err := http.Get(url)
	if err != nil {
		return
	}
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&dTrack)
	_, _, _, _, sURI, err := client.spotifyClient.SearchTrack("", "", "", dTrack.ISRC)
	if err == nil && len(sURI) > 0 {
		dTrack.SpotifyURI = sURI
	}
	dTrack.client = client
	itrack := &Track{deezerTrack: dTrack, playID: common.GenerateID()}
	err = client.PopulateMetadata(itrack)
	track = itrack
	return
}

func (client *Client) populateTracks(tracks []deezerTrack) (err error) {
	sngIDs := make([]string, len(tracks))
	for i, track := range tracks {
		sngIDs[i] = strconv.Itoa(track.ID)
	}
	body, err := json.Marshal(map[string][]string{
		"sng_ids": sngIDs,
	})
	if err != nil {
		return
	}
	req := client.makeUnofficialAPIRequest("song.getListData", body)
	response, err := client.httpClient.Do(req)
	if err != nil {
		client.initDeezerAPI()
		req = client.makeUnofficialAPIRequest("song.getListData", body)
		response, err = client.httpClient.Do(req)
		if err != nil {
			return
		}
	}
	var resp pageTrackResponse
	err = json.NewDecoder(response.Body).Decode(&resp)
	if err != nil {
		client.initDeezerAPI()
		req = client.makeUnofficialAPIRequest("song.getListData", body)
		response, err = client.httpClient.Do(req)
		if err != nil {
			return
		}
		err = json.NewDecoder(response.Body).Decode(&resp)
		if err != nil {
			return
		}
	}
	for i := range tracks {
		tracks[i].MD5Origin = resp.Results.Data[i].MD5Origin
		tracks[i].MediaVersion = resp.Results.Data[i].MediaVersion
	}
	return
}

//SearchTrack takes the track title and optional track's artist query and returns the best match track on Deezer
func (client *Client) SearchTrack(track, artist string) ([]common.Track, error) {
	var url string
	var sTrack, sArtist, sAlbum, sISRC, sURI string
	var err error
	withSpotify := client.spotifyClient != nil
	withISRC := withSpotify
start:
	if len(artist) == 0 && withSpotify {
		sTrack, sArtist, sAlbum, sISRC, sURI, err = client.spotifyClient.SearchTrackQuery(track)
		if err != nil {
			log.Printf("spotifyClient.SearchTrack() failed: %v\n", err)
			withSpotify = false
			goto start
		} else {
			if withISRC && len(sISRC) > 0 {
				url = fmt.Sprint("https://api.deezer.com/2.0/track/isrc:", sISRC)
			} else {
				url = fmt.Sprintf("https://api.deezer.com/search/track/?limit=%d&q=track:\"%s\"artist:\"%s\"album:\"%s\"", searchResultsLimit, template.URLQueryEscaper(sTrack), template.URLQueryEscaper(sArtist), template.URLQueryEscaper(sAlbum))
			}
		}
	} else {
		if len(artist) == 0 {
			url = fmt.Sprintf("https://api.deezer.com/search/track/?limit=%d&q=%s", searchResultsLimit, template.URLQueryEscaper(track))
		} else {
			url = fmt.Sprintf("https://api.deezer.com/search/track/?limit=%d&q=track:\"%s\"artist:\"%s\"", searchResultsLimit, template.URLQueryEscaper(track), template.URLQueryEscaper(artist))
		}
	}
	response, err := http.Get(url)
	if err != nil {
		if withSpotify {
			if withISRC {
				log.Println("Search with spotify ISRC failed")
				withISRC = false
				goto start
			}
			log.Println("Search with spotify failed")
			withSpotify = false
			goto start
		}
		return nil, err
	}
	defer response.Body.Close()
	var resp searchTrackResponse
	if withSpotify && withISRC {
		resp = searchTrackResponse{Data: make([]deezerTrack, 1)}
		err = json.NewDecoder(response.Body).Decode(&resp.Data[0])
		if resp.Data[0].ID == 0 {
			err = errors.New("ISRC not found on deezer")
		}
	} else {
		err = json.NewDecoder(response.Body).Decode(&resp)
	}
	if err != nil {
		if withSpotify {
			if withISRC {
				log.Println("Search with spotify ISRC failed")
				withISRC = false
			} else {
				log.Println("Search with spotify failed")
				withSpotify = false
			}
			goto start
		}
		return nil, err
	}
	itracks := resp.Data
	if len(itracks) <= 0 {
		if withISRC {
			log.Println("Search with spotify ISRC failed")
			withISRC = false
			goto start
		}
		if withSpotify {
			log.Println("Search with spotify failed")
			withSpotify = false
			goto start
		}
		return nil, errors.New("No track found")
	}
	tracks := make([]common.Track, len(itracks))
	err = client.populateTracks(itracks)
	if err != nil {
		return nil, err
	}
	for i, v := range itracks {
		if withSpotify && (v.ISRC == sISRC || (v.Title == sTrack && v.Artist.Name == sArtist && v.Album.Title == sAlbum)) {
			v.SpotifyURI = sURI
			if withISRC && i == 0 {
				var sURI string
				_, _, _, _, sURI, err = client.spotifyClient.SearchTrack(v.Title, v.Artist.Name, v.Album.Title, v.ISRC)
				if err == nil && len(sURI) > 0 {
					v.SpotifyURI = sURI
				}
			}
		}
		tracks[i] = &Track{deezerTrack: v, playID: common.GenerateID()}
	}
	return tracks, nil
}
