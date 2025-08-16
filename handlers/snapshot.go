package handlers

import (
	"bytes"
	"context"
	"fmt"
	jpeg "image/jpeg"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

// SnapshotHandler ...
func (h *Handler) SnapshotHandler(w http.ResponseWriter, r *http.Request) {
	var (
		body   []byte
		err    error
		client = http.DefaultClient
	)

	query := r.URL.Query()
	placeID := query.Get("placeID")
	accessControlID := query.Get("accessControlID")

	url := fmt.Sprintf(API_VIDEO_SNAPSHOT, placeID, accessControlID)
	// log.Println("/snapshotHandler", url)

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("snapshotHandler", err)

		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	request = request.WithContext(ctx)

	operator := strconv.Itoa(h.Config.Operator)

	// Конвертируем placeID из строки в int64
	placeIDInt, _ := strconv.ParseInt(placeID, 10, 64)

	rt := WithHeader(client.Transport)
	rt.Set("Host", API_HOST)
	rt.Set("User-Agent", GenerateUserAgent(h.Config.Operator, h.Config.UUID, placeIDInt))
	rt.Set("Operator", operator)
	rt.Set("Authorization", "Bearer "+h.Config.Token)
	client.Transport = rt

	resp, err := client.Do(request)
	if err != nil {
		log.Println("snapshotHandler", "connect error")

		return
	}

	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Println(err2)
		}
	}()

	body, err = io.ReadAll(resp.Body)

	if err == nil {
		contentType := http.DetectContentType(body)

		if resp.StatusCode != 200 {
			err = fmt.Errorf("wrong response, code: %d, result: %s", resp.StatusCode, string(body))
		} else if contentType != "image/jpeg" {
			err = fmt.Errorf("wrong response, code: %d, Content-Type: %s, result: %s", resp.StatusCode, contentType, string(body))
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
