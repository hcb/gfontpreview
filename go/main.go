package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

type GoogleFontFiles struct {
	Regular string `json:"regular"`
	Italic  string `json:"italic"`
}

type GoogleFontFilesMap map[string][]byte

var googleFontFiles GoogleFontFilesMap

type GoogleFont struct {
	Family       string          `json:"family"`
	Variants     []string        `json:"variants"`
	Subsets      []string        `json:"subsets"`
	Version      string          `json:"version"`
	LastModified string          `json:"lastModified"`
	Files        GoogleFontFiles `json:"files"`
	Category     string          `json: category`
	Kind         string          `json:"kind"`
	Menu         string          `json:"menu"`
}

type GoogleFontsList []GoogleFont

var fonts GoogleFontsList

func handler(w http.ResponseWriter, r *http.Request) {
	// Extract URL parameters using r.URL.Query().Get("paramName")
	font := r.URL.Query().Get("font")

	if font == "" {
		// Return the list of fonts
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fonts)
		return
	}

	// Check if parameters are present
	if font == "" {
		http.Error(w, "Missing font parameter", http.StatusBadRequest)
		return
	}

	// Check if the font is available
	found := false
	var gf GoogleFont
	for _, f := range fonts {
		if f.Family == font {
			found = true
			gf = f
			break
		}
	}

	if !found {
		http.Error(w, "Font not found", http.StatusNotFound)
		// fmt.Printf("Font not found: %s\n", font)
		return
	}

	// Check if the font file is already downloaded
	if googleFontFiles[font] == nil {
		// Download the font file
		client := http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}

		resp, err := client.Get(gf.Files.Regular)
		if err != nil {
			log.Fatal(err)
		}

		ba, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		googleFontFiles[font] = ba

	} else {
		log.Printf("Font file already downloaded for %s", font)
	}

	ba := googleFontFiles[font]

	f, err := truetype.Parse(ba)
	if err != nil {
		log.Printf("Error parsing font file: %s", googleFontFiles[font])
	}
	fg, bg := image.Black, image.White
	rgba := image.NewRGBA(image.Rect(0, 0, 200, 50))
	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)
	c := freetype.NewContext()
	// Draw the text
	c.SetDPI(72)
	c.SetFont(f)
	c.SetFontSize(24)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)

	// Draw the text, finally!

	c.DrawString(font, freetype.Pt(10, 25))

	// https://stackoverflow.com/questions/29105540/aligning-text-in-golang-with-truetype
	// Return image as png
	fb := new(bytes.Buffer)
	err = png.Encode(fb, rgba)
	if err != nil {
		log.Printf("Error encoding image: %s", err)
	}

	// return the image
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fb.Bytes())))
	w.Write(fb.Bytes())
}

// Fetch the Google Fonts
func getGoogleFonts(apiKey string) {
	// https://developers.google.com/fonts/docs/developer_api
	log.Print("Fetching Google Fonts API font list")
	// Fetch Google Fonts API
	url := fmt.Sprintf("https://www.googleapis.com/webfonts/v1/webfonts?key=%s", apiKey)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	} else {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body) // response body is []byte
		if err != nil {
			log.Fatal(err)
		}

		type GoogleFontsListResponse struct {
			Items []GoogleFont `json:"items"`
		}

		// Parse the JSON response into the fonts list
		var result GoogleFontsListResponse
		json.Unmarshal(body, &result)
		fonts = result.Items
	}

}

func main() {
	// Load .env file
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the Google Fonts
	GOOGLE_FONTS_API_KEY := os.Getenv("GOOGLE_FONTS_API_KEY")
	getGoogleFonts(GOOGLE_FONTS_API_KEY)
	googleFontFiles = make(GoogleFontFilesMap)

	// Register the handler function for the root route
	http.HandleFunc("/fonts", handler)

	// Start the server on port 8080
	fmt.Println("Server is listening on port http://localhost:8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
