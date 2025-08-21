package handlers

import (
	"bytes"
	"fmt"
	jpeg "image/jpeg"
	"log"
	"net/http"
	"strconv"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

// SnapshotHandler ...
func (h *Handler) SnapshotHandler(w http.ResponseWriter, r *http.Request) {
	var (
		body []byte
		err  error
	)

	query := r.URL.Query()
	placeID := query.Get("placeID")
	accessControlID := query.Get("accessControlID")
	if placeID == "" || accessControlID == "" {
		err = fmt.Errorf("provide placeID and accessControlID")
	} else {
		if h.API == nil {
			err = fmt.Errorf("api not initialized")
		} else {
			bodyBytes, upstreamErr, e := h.API.Snapshot(r.Context(), placeID, accessControlID)
			if e != nil {
				err = e
			} else if upstreamErr != nil {
				err = fmt.Errorf("snapshot upstream status %d", upstreamErr.StatusCode)
			} else {
				body = bodyBytes
			}
			if err == nil {
				ct := http.DetectContentType(body)
				if ct != "image/jpeg" {
					err = fmt.Errorf("wrong content-type: %s", ct)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "image/jpeg")

	if err != nil {
		width := 500
		height := 281

		dc := gg.NewContext(width, height)

		font, errParse := truetype.Parse(goregular.TTF)
		if errParse == nil {
			face := truetype.NewFace(font, &truetype.Options{Size: 16})
			dc.SetFontFace(face)
		}

		dc.SetHexColor("#000000")
		dc.DrawRectangle(0, 0, float64(width), float64(height))
		dc.Fill()
		dc.SetHexColor("#ffffff")
		dc.DrawStringWrapped(
			err.Error(),       // text
			float64(width/2),  // block x pos
			float64(height/2), // block y pos
			0.5,               // text x pos
			0,                 // text y pos
			float64(width/2),  // width
			1,                 // linespacing
			gg.AlignCenter,    // align
		)
		dc.Clip()

		b := bytes.Buffer{}

		var opt jpeg.Options
		opt.Quality = 75
		err = jpeg.Encode(&b, dc.Image(), &opt)
		if err != nil {
			log.Printf("png.Encode: %s", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		if _, err := w.Write(b.Bytes()); err != nil {
			log.Println("snapshotHandler", "unable to write image.")
		}

		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(body)))

	if _, err := w.Write(body); err != nil {
		log.Println("snapshotHandler", "unable to write image.")
	}
}
