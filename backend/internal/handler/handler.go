package handler

import (
	"encoding/json"
	"errors"
	"image"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ankan/copy-pasta/internal/converter"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type Handler struct{}

type convertResponse struct {
	ASCII  string `json:"ascii"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)

	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "request must be multipart/form-data")
		return
	}

	width := converter.DefaultWidth
	invert := false
	var img image.Image
	foundImage := false

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid multipart payload")
			return
		}

		func() {
			defer part.Close()

			switch part.FormName() {
			case "image":
				if foundImage || part.FileName() == "" {
					return
				}

				decoded, _, decodeErr := image.Decode(part)
				if decodeErr != nil {
					err = errors.New("invalid or unsupported image")
					return
				}

				img = decoded
				foundImage = true
			case "width":
				value, readErr := readField(part)
				if readErr != nil {
					err = readErr
					return
				}
				if value == "" {
					return
				}

				parsedWidth, parseErr := strconv.Atoi(value)
				if parseErr != nil || parsedWidth <= 0 {
					err = errors.New("width must be a positive integer")
					return
				}
				width = parsedWidth
			case "invert":
				value, readErr := readField(part)
				if readErr != nil {
					err = readErr
					return
				}
				if value == "" {
					return
				}

				parsedInvert, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					err = errors.New("invert must be a boolean")
					return
				}
				invert = parsedInvert
			}
		}()

		if err != nil {
			switch err.Error() {
			case "width must be a positive integer", "invert must be a boolean", "invalid or unsupported image":
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}
	}

	if !foundImage {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}

	ascii := converter.Convert(img, converter.Options{Width: width, Invert: invert})
	trimmed := strings.TrimRight(ascii, "\n")
	height := 0
	if trimmed != "" {
		height = strings.Count(trimmed, "\n") + 1
	}

	writeJSON(w, http.StatusOK, convertResponse{
		ASCII:  ascii,
		Width:  width,
		Height: height,
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func readField(part io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(part, 1024))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
