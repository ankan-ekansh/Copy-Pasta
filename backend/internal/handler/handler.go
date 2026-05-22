package handler

import (
	"encoding/json"
	"errors"
	"image"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/converter"
	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/store"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type Handler struct {
	store store.Store
}

type Option func(*Handler)

// WithStore configures the handler with a persistence store.
func WithStore(s store.Store) Option {
	return func(h *Handler) {
		h.store = s
	}
}

type convertResponse struct {
	ID     string `json:"id,omitempty"`
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

func New(opts ...Option) *Handler {
	h := &Handler{}
	for _, opt := range opts {
		opt(h)
	}
	return h
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
	edgeMix := 0.0  // will use converter default (0.3) when left at 0
	contrast := 0.0 // will use converter default (1.3) when left at 0
	charRamp := ""
	mode := "ascii"    // "ascii" or "braille"
	threshold := 0.0   // braille threshold (0 = auto Otsu)
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
			case "edgeMix":
				value, readErr := readField(part)
				if readErr != nil {
					err = readErr
					return
				}
				if value == "" {
					return
				}
				parsedEdge, parseErr := strconv.ParseFloat(value, 64)
				if parseErr != nil || parsedEdge < 0 || parsedEdge > 1 {
					err = errors.New("edgeMix must be a float between 0 and 1")
					return
				}
				edgeMix = parsedEdge
			case "contrast":
				value, readErr := readField(part)
				if readErr != nil {
					err = readErr
					return
				}
				if value == "" {
					return
				}
				parsedContrast, parseErr := strconv.ParseFloat(value, 64)
				if parseErr != nil || parsedContrast < 0.1 || parsedContrast > 3.0 {
					err = errors.New("contrast must be a float between 0.1 and 3.0")
					return
				}
				contrast = parsedContrast
			case "charRamp":
				value, readErr := readField(part)
				if readErr != nil {
					err = readErr
					return
				}
				if value != "" {
					charRamp = value
				}
			case "mode":
				value, readErr := readField(part)
				if readErr != nil {
					err = readErr
					return
				}
				if value == "braille" || value == "ascii" {
					mode = value
				} else if value != "" {
					err = errors.New("mode must be 'ascii' or 'braille'")
					return
				}
			case "threshold":
				value, readErr := readField(part)
				if readErr != nil {
					err = readErr
					return
				}
				if value == "" {
					return
				}
				parsedThreshold, parseErr := strconv.ParseFloat(value, 64)
				if parseErr != nil || parsedThreshold < 0 || parsedThreshold > 1 {
					err = errors.New("threshold must be a float between 0 and 1")
					return
				}
				threshold = parsedThreshold
			}
		}()

		if err != nil {
			switch err.Error() {
			case "width must be a positive integer", "invert must be a boolean",
				"invalid or unsupported image", "mode must be 'ascii' or 'braille'",
				"threshold must be a float between 0 and 1",
				"edgeMix must be a float between 0 and 1",
				"contrast must be a float between 0.1 and 3.0":
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

	var ascii string
	if mode == "braille" {
		brailleWidth := width
		if brailleWidth == converter.DefaultWidth {
			brailleWidth = 80 // better default for braille
		}
		ascii = converter.ConvertBraille(img, converter.BrailleOptions{
			Width:     brailleWidth,
			Threshold: threshold,
			Invert:    invert,
			Dither:    true,
		})
	} else {
		ascii = converter.Convert(img, converter.Options{
			Width:    width,
			Invert:   invert,
			EdgeMix:  edgeMix,
			Contrast: contrast,
			CharRamp: charRamp,
		})
	}
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
